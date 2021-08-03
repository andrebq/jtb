package engine

import (
	"fmt"
	"io"
	"io/ioutil"
	"strconv"

	"github.com/dop251/goja"
	"github.com/rs/zerolog"
)

type (
	// E contains the engine used to run all javascript code
	E struct {
		runtime *goja.Runtime

		stdin  io.Reader
		stderr io.Writer
		stdout io.Writer

		interactiveEval int64
		errCount        int64

		logger zerolog.Logger
	}

	noInput struct{}

	toObject interface {
		ToObject() goja.Value
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
	return e, nil
}

func (e *E) protectGlobals() error {
	_, err := e.runtime.RunScript("__goja__boot.js", `
	Object.freeze(Object);
	Object.freeze(Array);
	Object.freeze(String);
	Object.freeze(Number);
	Object.freeze(Date);
	`)
	return err
}

func (e *E) registerGlobal(name string, value toObject) error {
	gojaValue := value.ToObject()
	fn, err := e.runtime.RunScript("__goja__register_global.js", `(function(obj){
		Object.freeze(obj);
		return obj;
	})`)
	if err != nil {
		return err
	}
	callable, ok := goja.AssertFunction(fn)
	if !ok {
		panic("All bets are off and there is something reall really weird with goja! It is not safe to proceed!")
	}
	obj, err := callable(e.runtime.GlobalObject(), gojaValue)
	if err != nil {
		panic("All bets are off and there is something really really weird with Object.freez or function evaluation! It is not safe to proceed!")
	}
	return e.runtime.GlobalObject().Set(name, obj)
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
