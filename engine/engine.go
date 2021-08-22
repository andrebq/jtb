package engine

import (
	"fmt"
	"io"
	"io/ioutil"
	"path/filepath"
	"strconv"

	"github.com/andrebq/jtb/internal/modules/encoding/utf8"
	"github.com/andrebq/jtb/internal/modules/rawexec"
	"github.com/andrebq/jtb/internal/modules/sleep"
	"github.com/andrebq/jtb/internal/modules/stdio"
	"github.com/andrebq/jtb/internal/modules/uuid"
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
	{
		// ONLY SAFE MODULES SHOULD BE LISTED HERE
		if err := e.AddRemoteBuiltin("@jtb", &jtbModule{version: jtbVersion}); err != nil {
			return nil, err
		}
		if err := e.AddRemoteBuiltin("@encoding/utf8", &utf8.Module{}); err != nil {
			return nil, err
		}
		if err := e.AddRemoteBuiltin("@uuid", &uuid.Module{}); err != nil {
			return nil, err
		}
	}

	// Although it might seem that @stdio is safe for remote
	// this would allow a remote module to print arbitrary content
	// in the stdout/stdin
	if err := e.AddBuiltin("@stdio", false, &stdio.Module{
		Stdout: func() io.Writer { return e.stdout },
		Stderr: func() io.Writer { return e.stderr },
		Stdin:  func() io.Reader { return e.stdin },
	}); err != nil {
		return nil, err
	}

	if err := e.AddBuiltin("@sleep", false, &sleep.Module{}); err != nil {
		return nil, err
	}

	if err := e.AddBuiltin("@rawexec", true, &rawexec.Module{
		Logger: e.logger.With().Str("module", "@rawexec").Logger(),
	}); err != nil {
		return nil, err
	}

	err = e.registerGlobal("require", rr)
	err = e.AnchorModules(".")
	if err != nil {
		return nil, err
	}
	return e, nil
}

// ConnectStdio changes the std in/out/err streams from the default descard ones
// to ones that connect to the given ones.
//
// If an entry is nil, the one already configured in the engine is kept
func (e *E) ConnectStdio(in io.Reader, out, err io.Writer) {
	e.stdin = in
	e.stdout = out
	e.stderr = err
}

// AddBuiltin module, if the module is marked as sensitve, the module will be marked as
// dangerous and restrict (users need to call Unrestrict to enable the module for local/trusted scripts).
//
// If senstive is false, the module will be available for local/trusted scripts right after this function
// returns.
//
// To add a module for remote use, call AddRemoteBuiltin.
func (e *E) AddBuiltin(name string, sensitive bool, module moduleDefiner) error {
	if err := e.require.canRegisterBuiltin(name); err != nil {
		return err
	}
	if err := e.require.registerBuiltin(name, module); err != nil {
		return err
	}
	if sensitive {
		e.require.markAsDangerous(name)
	}
	return nil
}

// AddRemoteBuiltin adds the given module and exposes it to all scripts (local/trusted/remote)
//
// Be careful with which types of modules are defined for remote scripts as there won't be any
// restrictions on what functions a remote script can make.
func (e *E) AddRemoteBuiltin(name string, module moduleDefiner) error {
	if err := e.require.canRegisterBuiltin(name); err != nil {
		return err
	}
	if err := e.require.registerBuiltin(name, module); err != nil {
		return err
	}
	e.require.markAsSafeForRemote(true, name)
	return nil
}

// Unrestrict the given module and allows access to it from local sources or
// other trusted sources.
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
