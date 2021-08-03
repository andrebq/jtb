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
	}

	moduleDef struct {
		exports goja.Value
	}

	moduleDefiner interface {
		// DefineModule by setting all public properties to the goja.Object.
		//
		// The runtime is passed to avoid having to export it directly form the Engine.
		DefineModule(*goja.Object, *goja.Runtime, *E) error
	}
)

func (r *rootRequire) ToValue() goja.Value {
	return r.e.runtime.ToValue(r.require)
}

func (r *rootRequire) require(call goja.FunctionCall) goja.Value {
	name := call.Argument(0).ToString().Export().(string)
	def := r.builtins[name]
	if def == nil {
		panic(r.e.runtime.NewGoError(fmt.Errorf("Module %v not defined", name)))
	}
	return def.exports
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
	err := definer.DefineModule(exports, r.e.runtime, r.e)
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
