package engine

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/dop251/goja"
)

type (
	// console implements a restricted form of javascript console global
	console struct {
		e *E
	}
)

func (c *console) ToObject() goja.Value {
	obj := c.e.runtime.NewObject()

	obj.Set("info", c.info)
	obj.Set("debug", c.info)
	obj.Set("error", c.info)

	return obj
}

func (c *console) info(call goja.FunctionCall) goja.Value {
	buf := &bytes.Buffer{}
	for idx, v := range call.Arguments {
		if idx > 0 {
			buf.WriteRune(' ')
		}
		exported := v.Export()
		jsonBytes, err := json.Marshal(exported)
		if err != nil {
			fmt.Fprintf(buf, "%v", exported)
			continue
		}
		buf.Write(jsonBytes)
	}
	_, err := io.Copy(c.e.stderr, buf)
	if err != nil {
		errID := c.e.logError("console.info", err)
		panic(c.e.runtime.NewGoError(errors.New("Unable to process call to console: " + errID)))
	}
	return goja.Undefined()
}
