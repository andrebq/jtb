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

		dangerous map[string]struct{}

		restricted map[string]struct{}
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

func (r *rootRequire) mustNotBeRestricted(name string) {
	if r.restricted == nil {
		return
	}
	if _, ok := r.restricted[name]; ok {
		panic(r.e.runtime.NewGoError(errModuleIsRestricted(fmt.Sprintf("Module %v is restricted", name))))
	}
}

func (r *rootRequire) require(call goja.FunctionCall) goja.Value {
	r.init()
	name := call.Argument(0).ToString().Export().(string)
	r.mustNotBeRestricted(name)
	switch {
	case r.isBuiltin(name):
		return r.requireBuiltin(name)
	case r.isLocal(name):
		return r.requireLocal(name)
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

func (r *rootRequire) markAsDangerous(name string) {
	if r.dangerous == nil {
		r.dangerous = make(map[string]struct{})
	}
	r.dangerous[name] = struct{}{}
}

func (r *rootRequire) markAsRestricted(name string, restricted bool) {
	if r.restricted == nil {
		r.restricted = map[string]struct{}{}
	}
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

func (r *rootRequire) hasModule(name string) *moduleDef {
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
	r.initDone = true
}

func (r *rootRequire) saveModule(name string, md *moduleDef) {
	r.init()
	r.modules[name] = md
}
