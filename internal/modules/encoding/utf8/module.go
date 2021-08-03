package utf8

import (
	"errors"
	"unicode/utf8"

	"github.com/dop251/goja"
)

type (
	Module struct {
	}
)

func (m *Module) DefineModule(exports *goja.Object, runtime *goja.Runtime) error {
	exports.Set("decode", decodeUTF8(runtime))
	exports.Set("encode", encodeUTF8(runtime))
	return nil
}

func decodeUTF8(runtime *goja.Runtime) func(goja.FunctionCall) goja.Value {
	return func(fc goja.FunctionCall) goja.Value {
		var buf []byte
		runtime.ExportTo(fc.Argument(0), &buf)
		if !utf8.Valid(buf) {
			panic(runtime.NewGoError(errors.New("buffer is not a valid utf8 stream")))
		}
		str := string(buf)
		return runtime.ToValue(str)
	}
}

func encodeUTF8(runtime *goja.Runtime) func(goja.FunctionCall) goja.Value {
	return func(fc goja.FunctionCall) goja.Value {
		str := fc.Argument(0).ToString().Export().(string)
		if !utf8.ValidString(str) {
			panic(runtime.NewGoError(errors.New("string is not encoded using utf-8")))
		}
		return runtime.ToValue([]byte(str))
	}
}
