package rawexec

import (
	"bytes"
	"fmt"
	"os/exec"
	"strconv"

	"github.com/andrebq/jtb/internal/modules/modutils"
	"github.com/dop251/goja"
	"github.com/rs/zerolog"
)

type (
	Module struct {
		Logger zerolog.Logger
	}
)

func (m *Module) DefineModule(exports *goja.Object, runtime *goja.Runtime) error {
	exports.Set("call_strict", m.callBinary(runtime, true, m.Logger.With().Str("method", "call_strict").Logger()))
	exports.Set("call", m.callBinary(runtime, false, m.Logger.With().Str("method", "call").Logger()))
	return nil
}

func (m *Module) callBinary(runtime *goja.Runtime, strict bool, logger zerolog.Logger) func(goja.FunctionCall) goja.Value {
	return func(fc goja.FunctionCall) goja.Value {
		cmd := exec.Command(fc.Argument(0).ToString().Export().(string))
		if len(fc.Arguments) == 2 {
			obj := fc.Arguments[1].ToObject(runtime)
			if obj.Get("args") != nil {
				args := obj.Get("args").ToObject(runtime)
				for i := 0; i < int(args.Get("length").ToInteger()); i++ {
					str := args.Get(strconv.Itoa(i)).ToString().Export().(string)
					cmd.Args = append(cmd.Args, str)
				}
			}
			if obj.Get("input") != nil {
				var buf []byte
				runtime.ExportTo(obj.Get("input"), &buf)
				cmd.Stdin = bytes.NewBuffer(buf)
			} else {
				cmd.Stdin = nil
			}
		}
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}
		cmd.Stdout = stdout
		cmd.Stderr = stderr
		err := cmd.Run()
		if err != nil && strict {
			modutils.AppendCallStack(logger.Error().Err(err), runtime).Strs("args", cmd.Args).Str("exec_path", cmd.Path).Interface("pid", cmd.ProcessState.Pid()).Msg("Command failed with unexpected error")
			panic(runtime.NewGoError(fmt.Errorf("Command failed with status code %v", cmd.ProcessState.ExitCode())))
		}
		obj := runtime.NewObject()
		obj.Set("exitCode", runtime.ToValue(cmd.ProcessState.ExitCode()))
		obj.Set("stdout", runtime.ToValue(stdout.Bytes()))
		obj.Set("stderr", runtime.ToValue(stderr.Bytes()))
		return obj
	}
}
