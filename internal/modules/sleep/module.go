package sleep

import (
	"fmt"
	"time"

	"github.com/dop251/goja"
)

type (
	Module struct{}
)

func (m Module) DefineModule(exports *goja.Object, runtime *goja.Runtime) error {
	exports.Set("sleep", func(call goja.FunctionCall) goja.Value {
		val := call.Argument(0).Export()
		switch val := val.(type) {
		case string:
			dur, err := time.ParseDuration(val)
			if err != nil {
				panic(runtime.NewGoError(fmt.Errorf("%v is not a valid duration: %w", val, err)))
			}
			time.Sleep(dur)
		case int64:
			time.Sleep(time.Second * time.Duration(val))
		case float64:
			time.Sleep(time.Duration(float64(time.Second) * val))
		default:
			panic(runtime.NewGoError(fmt.Errorf("the argument for the sleep function must be either: a duration string or a fractional number of seconds to sleep")))
		}
		return goja.Undefined()
	})
	return nil
}
