package stdio

import (
	"fmt"
	"io"

	"github.com/dop251/goja"
)

type (
	Module struct {
		Stdout func() io.Writer
		Stderr func() io.Writer
		Stdin  func() io.Reader
	}
)

func (m *Module) DefineModule(exports *goja.Object, runtime *goja.Runtime) error {
	exports.Set("print", m.printIO(runtime, " ", false, false))
	exports.Set("println", m.printIO(runtime, "\n", true, false))

	exports.Set("eprint", m.printIO(runtime, " ", false, true))
	exports.Set("eprintln", m.printIO(runtime, " ", false, true))
	return nil
}

func (m *Module) printIO(runtime *goja.Runtime, sep string, sepAtTheEnd bool, useStderr bool) func(goja.FunctionCall) goja.Value {
	return func(fc goja.FunctionCall) goja.Value {
		out := m.Stdout()
		if useStderr {
			out = m.Stderr()
		}
		for i, v := range fc.Arguments {
			sv := v.ToString().Export()
			if i != 0 && !sepAtTheEnd {
				fmt.Fprint(out, sep)
			}
			fmt.Fprint(out, sv)
		}
		if sepAtTheEnd {
			fmt.Fprint(out, sep)
		}
		return runtime.ToValue(true)
	}
}
