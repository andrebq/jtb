package engine

import "github.com/dop251/goja"

func (e *E) IsRestrictedModule(err error) (error, bool) {
	ex, ok := err.(*goja.Exception)
	if !ok {
		return nil, false
	}
	value := ex.Value().ToObject(e.runtime).Get("value")
	if value == nil {
		return nil, false
	}
	_, ok = value.Export().(errModuleIsRestricted)
	if !ok {
		return nil, false
	}
	return err, ok
}
