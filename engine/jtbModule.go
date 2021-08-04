package engine

import "github.com/dop251/goja"

type (
	jtbModule struct {
		version string
	}
)

// DefineModule by setting any fields to the given exports object,
// the engine instance is passed to provide access to any special method
// that might be required.
//
// As well as access to the runtime engine
func (j *jtbModule) DefineModule(exports *goja.Object, runtime *goja.Runtime) error {
	exports.Set("version", j.version)
	return nil
}
