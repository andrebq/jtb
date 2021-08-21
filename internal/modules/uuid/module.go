package uuid

import (
	"crypto/rand"
	"errors"
	"io"

	"github.com/dop251/goja"
	"github.com/google/uuid"
	"github.com/maruel/fortuna"
)

type (
	Module struct {
		rng fortuna.Fortuna
	}
)

func (m *Module) DefineModule(exports *goja.Object, runtime *goja.Runtime) error {
	var seed [128]byte
	_, err := io.ReadFull(rand.Reader, seed[:])
	if err != nil {
		return err
	}
	m.rng, err = fortuna.NewFortuna(seed[:])
	if err != nil {
		return err
	}
	exports.Set("v4", m.newv4(runtime))
	exports.Set("v5", newv5(runtime))
	return nil
}

func (m *Module) newv4(runtime *goja.Runtime) func(goja.FunctionCall) goja.Value {
	return func(fc goja.FunctionCall) goja.Value {
		id, err := uuid.NewRandomFromReader(m.rng)
		if err != nil {
			panic(runtime.NewGoError(errors.New("unable to compute a new random uuid")))
		}
		return runtime.ToValue(id.String())
	}
}

func newv5(runtime *goja.Runtime) func(goja.FunctionCall) goja.Value {
	return func(fc goja.FunctionCall) goja.Value {
		ns := fc.Argument(0).ToString().Export().(string)
		salt := fc.Argument(1).Export()

		id, err := uuid.Parse(ns)
		if err != nil {
			panic(runtime.NewGoError(errors.New("value is not a valid UUID")))
		}
		switch salt := salt.(type) {
		case []byte:
			return runtime.ToValue(uuid.NewSHA1(id, salt))
		case string:
			return runtime.ToValue(uuid.NewSHA1(id, []byte(salt)))
		}
		return runtime.ToValue(uuid.NewSHA1(id, []byte(fc.Argument(1).ToString().Export().(string))))
	}
}
