package engine

import (
	"errors"
	"fmt"

	"github.com/dop251/goja"
)

type (
	// rootRequire implements the require function used by the interactive evaluator
	rootRequire struct {
		e *E

		builtins map[string]*moduleDef

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
	name := call.Argument(0).ToString().Export().(string)
	r.mustNotBeRestricted(name)
	def := r.builtins[name]
	if def == nil {
		panic(r.e.runtime.NewGoError(fmt.Errorf("Module %v not defined", name)))
	}
	return def.exports
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
	if r.builtins == nil {
		r.builtins = make(map[string]*moduleDef)
	}
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
