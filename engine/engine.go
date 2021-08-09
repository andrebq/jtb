package engine

import (
	"fmt"
	"io"
	"io/ioutil"
	"path/filepath"
	"strconv"

	"github.com/andrebq/jtb/internal/modules/encoding/utf8"
	"github.com/andrebq/jtb/internal/modules/rawexec"
	"github.com/andrebq/jtb/internal/modules/stdio"
	"github.com/dop251/goja"
	"github.com/rs/zerolog"
	"github.com/spf13/afero"
)

const (
	jtbVersion = "--bleeding-edge--"
)

type (
	// E contains the engine used to run all javascript code
	E struct {
		runtime *goja.Runtime

		fs afero.Fs

		stdin  io.Reader
		stderr io.Writer
		stdout io.Writer

		interactiveEval int64
		errCount        int64

		logger zerolog.Logger

		require *rootRequire
	}

	noInput struct{}

	toValue interface {
		ToValue() goja.Value
	}
)

func (_ noInput) Read(_ []byte) (int, error) {
	return 0, io.EOF
}

// New returns an empty engine with the bare minimum global objects required to run
// any workload
func New() (*E, error) {
	e := &E{
		runtime: goja.New(),
		stdin:   noInput{},
		stderr:  ioutil.Discard,
		stdout:  ioutil.Discard,
		logger:  zerolog.Nop(),
	}
	err := e.protectGlobals()
	if err != nil {
		return nil, err
	}
	err = e.registerGlobal("console", &console{e: e})
	if err != nil {
		return nil, err
	}
	base, err := filepath.Abs(".")
	if err != nil {
		return nil, err
	}
	rr := &rootRequire{e: e, anchor: base}
	e.require = rr
	rr.registerBuiltin("@jtb", &jtbModule{version: jtbVersion})
	rr.registerBuiltin("@encoding/utf8", &utf8.Module{})
	rr.registerBuiltin("@stdio", &stdio.Module{
		Stdout: func() io.Writer { return e.stdout },
		Stderr: func() io.Writer { return e.stderr },
		Stdin:  func() io.Reader { return e.stdin },
	})
	rr.registerBuiltin("@rawexec", &rawexec.Module{
		Logger: e.logger.With().Str("module", "@rawexec").Logger(),
	})
	rr.markAsDangerous("@rawexec")
	rr.markAsRestricted("@rawexec", true)
	err = e.registerGlobal("require", rr)
	err = e.AnchorModules(".")
	if err != nil {
		return nil, err
	}
	return e, nil
}

// Unrestrict the given module and allows access to it from local sources or
// other trusted sources.
//
// Untrusted source are not affected and will never be able to access restricted
// modules.
func (e *E) Unrestrict(name string) {
	e.require.markAsRestricted(name, false)
}

func (e *E) InteractiveEval(code string) (interface{}, error) {
	e.interactiveEval++
	val, err := e.runtime.RunScript(fmt.Sprintf("__eval_statement_%v.js", e.interactiveEval), code)
	if err != nil {
		return nil, err
	}
	if val == nil {
		return nil, nil
	}
	return val.Export(), nil
}

func (e *E) SetStderr(buf io.Writer) error {
	err := e.closeAll(e.stderr)
	e.stderr = buf
	return err
}

func (e *E) Close() error {
	return e.closeAll(e.stdin, e.stdout, e.stderr)
}

func (e *E) AnchorModules(path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	e.require.anchor = abs
	e.require.e.fs = OpenOSFilesystem(abs)
	return nil
}

func (e *E) closeAll(objs ...interface{}) error {
	var firstErr error
	for _, v := range objs {
		if closer, ok := v.(io.Closer); ok {
			err := closer.Close()
			if firstErr == nil && err != nil {
				firstErr = err
			}
		}
	}
	return firstErr
}

func (e *E) logError(tag string, err error) string {
	e.errCount++
	e.logger.Error().Err(err).Str("tag", tag).Int64("errCount", e.errCount).Send()
	return strconv.FormatInt(e.errCount, 16)
}

func (e *E) freeze(gojaValue goja.Value) (goja.Value, error) {
	fn, err := e.runtime.RunScript("__goja__freeze.js", `(function(obj){
		Object.freeze(obj);
		return obj;
	})`)
	if err != nil {
		return nil, err
	}
	callable, ok := goja.AssertFunction(fn)
	if !ok {
		panic("All bets are off and there is something reall really weird with goja! It is not safe to proceed!")
	}
	obj, err := callable(e.runtime.GlobalObject(), gojaValue)
	if err != nil {
		panic("All bets are off and there is something really really weird with Object.freeze or function evaluation! It is not safe to proceed!")
	}
	return obj, nil
}

func (e *E) protectGlobals() error {
	_, err := e.runtime.RunScript("__goja__boot.js", `
	Object.freeze(Object);
	Object.freeze(Array);
	Object.freeze(String);
	Object.freeze(Number);
	Object.freeze(Date);
	Object.freeze(Function);
	`)
	return err
}

func (e *E) registerGlobal(name string, value toValue) error {
	obj, err := e.freeze(value.ToValue())
	if err != nil {
		return err
	}
	return e.runtime.GlobalObject().Set(name, obj)
}
