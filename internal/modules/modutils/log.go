package modutils

import (
	"fmt"

	"github.com/dop251/goja"
	"github.com/rs/zerolog"
)

func AppendCallStack(entry *zerolog.Event, runtime *goja.Runtime) *zerolog.Event {
	stack := runtime.CaptureCallStack(-1, nil)
	arr := zerolog.Arr()
	for _, v := range stack {
		arr.Str(fmt.Sprintf("%v @ %v from %v", v.FuncName(), v.Position(), v.SrcName()))
	}
	return entry.Array("goja-call-stack", arr)
}
