package engine

import (
	"errors"
	"fmt"
	"net/url"
	"path"
	"strings"

	"github.com/dop251/goja"
)

type (
	// rootRequire implements the require function used by the interactive evaluator
	rootRequire struct {
		e *E

		initDone bool

		anchor string

		builtins map[string]*moduleDef
		modules  map[string]*moduleDef

		dangerous               map[string]struct{}
		builtinsAllowedOnRemote map[string]struct{}
		restricted              map[string]struct{}
	}

	moduleDef struct {
		exports goja.Value
	}

	moduleDefiner interface {
		// DefineModule by setting all public properties to the goja.Object.
		//
		// The runtime is passed to avoid having to export it directly form the Engine.
		DefineModule(*goja.Object, *goja.Runtime) error
	}

	errModuleIsRestricted string
)

func (e errModuleIsRestricted) Error() string { return string(e) }

func (r *rootRequire) ToValue() goja.Value {
	return r.e.runtime.ToValue(r.require)
}

func (r *rootRequire) mustBeSafeForRemote(name string) {
	if !r.isAllowedForRemote(name) {
		panic(r.e.runtime.NewGoError(errModuleIsRestricted(fmt.Sprintf("Module %v is not allowed from remote hosts", name))))
	}
}

func (r *rootRequire) mustNotBeRestricted(name string) {
	if r.isRestricted(name) {
		panic(r.e.runtime.NewGoError(errModuleIsRestricted(fmt.Sprintf("Module %v is restricted", name))))
	}
}

func (r *rootRequire) require(call goja.FunctionCall) goja.Value {
	r.init()
	name := call.Argument(0).ToString().Export().(string)
	r.mustNotBeRestricted(name)
	return r.doRequire(name)
}

func (r *rootRequire) requireFromRemote(name string) goja.Value {
	r.mustBeSafeForRemote(name)
	return r.doRequire(name)
}

func (r *rootRequire) doRequire(name string) goja.Value {
	switch {
	case r.isBuiltin(name):
		return r.requireBuiltin(name)
	case r.isLocal(name):
		return r.requireLocal(name)
	case r.isRemote(name):
		return r.requireRemote(name)
	default:
		panic(r.e.runtime.NewGoError(fmt.Errorf("Path %v is not understood as a valid module path", name)))
	}
}

func (r *rootRequire) requireBuiltin(name string) goja.Value {
	def := r.builtins[name]
	if def == nil {
		panic(r.e.runtime.NewGoError(fmt.Errorf("Module %v not defined", name)))
	}
	return def.exports
}

func (r *rootRequire) requireLocal(name string) goja.Value {
	fr := &trustedFileRequire{root: r, dir: ""}
	return fr.require(name)
}

func (r *rootRequire) requireRemote(name string) goja.Value {
	remote, err := newRemote(r, name)
	if err != nil {
		panic(r.e.runtime.NewGoError(errors.New("unable to create a untrusted remote")))
	}
	return remote.require(name)
}

func (r *rootRequire) markAsDangerous(name string) {
	r.init()
	r.dangerous[name] = struct{}{}
	r.restricted[name] = struct{}{}
}

func (r *rootRequire) isAllowedForRemote(name string) bool {
	r.init()
	_, isAllowed := r.builtinsAllowedOnRemote[name]
	return r.isBuiltin(name) &&
		isAllowed &&
		!r.isRestricted(name) &&
		!r.isDangerous(name)
}

func (r *rootRequire) isRestricted(name string) bool {
	_, is := r.restricted[name]
	return is
}

func (r *rootRequire) isDangerous(name string) bool {
	_, is := r.dangerous[name]
	return is
}

func (r *rootRequire) markAsSafeForRemote(safe bool, names ...string) {
	r.init()
	for _, name := range names {
		if !safe {
			delete(r.builtinsAllowedOnRemote, name)
		}
		r.builtinsAllowedOnRemote[name] = struct{}{}
	}
}

func (r *rootRequire) markAsRestricted(name string, restricted bool) {
	r.init()
	if restricted {
		r.restricted[name] = struct{}{}
	} else {
		delete(r.restricted, name)
	}
}

func (r *rootRequire) registerBuiltin(name string, definer moduleDefiner) error {
	r.init()
	if r.builtins[name] != nil {
		return errors.New("module is already defined!")
	}
	exports := r.e.runtime.CreateObject(nil)
	df := &moduleDef{
		exports: exports,
	}
	err := definer.DefineModule(exports, r.e.runtime)
	if err != nil {
		return err
	}
	df.exports, err = r.e.freeze(exports)
	if err != nil {
		return err
	}
	r.builtins[name] = df
	return nil
}

func (r *rootRequire) isBuiltin(name string) bool { return strings.HasPrefix(name, "@") }

func (r *rootRequire) isLocal(name string) bool {
	u, err := url.Parse(name)
	if err != nil {
		return false
	}
	return u.Scheme == "" && path.Ext(path.Clean(u.Path)) == ".js"
}

func (r *rootRequire) isRemote(name string) bool {
	u, err := url.Parse(name)
	if err != nil {
		return false
	}
	return u.Scheme != "" && path.Ext(path.Clean(u.Path)) == ".js"
}
func (r *rootRequire) hasModule(name string) *moduleDef {
	r.init()
	if r.isBuiltin(name) {
		return r.builtins[name]
	}
	return r.modules[name]
}

func (r *rootRequire) init() {
	if r.initDone {
		return
	}
	r.builtins = make(map[string]*moduleDef)
	r.dangerous = make(map[string]struct{})
	r.modules = make(map[string]*moduleDef)
	r.restricted = make(map[string]struct{})
	r.builtinsAllowedOnRemote = make(map[string]struct{})
	r.initDone = true
}

func (r *rootRequire) saveModule(name string, md *moduleDef) {
	r.init()
	r.modules[name] = md
}

func (e *rootRequire) canRegisterBuiltin(name string) error {
	if !strings.HasPrefix(name, "@") {
		return errors.New("builtin modules must start with @")
	}
	e.init()
	if m := e.hasModule(name); m != nil {
		return fmt.Errorf("module %v already registered", name)
	}
	return nil
}
