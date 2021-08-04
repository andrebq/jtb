package unsafe_fetch

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/andrebq/jtb/internal/modules/modutils"
	"github.com/dop251/goja"
	"github.com/rs/zerolog"
)

type (
	Module struct {
		Logger zerolog.Logger
	}
)

// DefineModule populates exports with all functions exposed by this package
func (m *Module) DefineModule(exports *goja.Object, runtime *goja.Runtime, logger zerolog.Logger) error {
	exports.Set("getJSON", m.getJSON(runtime, logger.With().Str("function", "getJSON").Logger()))
	exports.Set("doHTTP", m.doHTTP(runtime, logger.With().Str("function", "doHTTP").Logger()))
	return nil
}

func (m *Module) getJSON(runtime *goja.Runtime, log zerolog.Logger) func(goja.FunctionCall) goja.Value {
	return func(fc goja.FunctionCall) goja.Value {
		target := fc.Arguments[0].Export().(string)
		resp, err := http.Get(target)
		if err != nil {
			entry := log.Error().Err(err).Str("target", target)
			modutils.AppendCallStack(entry, runtime).Msgf("Unable to call HTTP GET %v", target)
			panic(runtime.NewGoError(errors.New("Unable to fetch resources from HTTP endpoint. Check logs for more information")))
		}
		defer resp.Body.Close()
		var out interface{}
		err = json.NewDecoder(resp.Body).Decode(&out)
		if err != nil {
			entry := log.Error().Err(err).Str("target", target)
			modutils.AppendCallStack(entry, runtime).Msgf("Unable to decode response from HTTP GET %v", target)
			panic(runtime.NewGoError(errors.New("Unable to fetch resources from HTTP endpoint. Check logs for more information")))
		}
		return runtime.ToValue(out)
	}
}

func (m *Module) doHTTP(runtime *goja.Runtime, log zerolog.Logger) func(goja.FunctionCall) goja.Value {
	return func(fc goja.FunctionCall) goja.Value {
		target := fc.Arguments[0].Export().(string)
		opts := struct {
			Headers   map[string][]string
			BodyStr   string
			Method    string
			BodyBytes *goja.ArrayBuffer
		}{}
		if len(fc.Arguments) > 0 {
			runtime.ExportTo(fc.Argument(1), &opts)
		}

		var buf io.Reader
		if len(opts.BodyStr) > 0 {
			buf = bytes.NewBufferString(opts.BodyStr)
		} else if opts.BodyBytes != nil && len(opts.BodyBytes.Bytes()) > 0 {
			buf = bytes.NewBuffer(append([]byte(nil), opts.BodyBytes.Bytes()...))
		}

		if opts.Method == "" {
			opts.Method = "GET"
		}

		req, err := http.NewRequest(opts.Method, target, buf)
		if err != nil {
			entry := log.Error().Err(err).Str("target", target).Str("method", opts.Method)
			modutils.AppendCallStack(entry, runtime).Msgf("Unable to prepare request object HTTP %v %v", opts.Method, target)
			panic(runtime.NewGoError(errors.New("Unable to fetch resources from HTTP endpoint. Check logs for more information")))
		}
		if len(opts.Headers) > 0 {
			for k, valueList := range opts.Headers {
				for _, v := range valueList {
					req.Header.Add(k, v)
				}
			}
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			entry := log.Error().Err(err).Str("target", target).Str("method", opts.Method)
			modutils.AppendCallStack(entry, runtime).Msgf("Unable to perform request HTTP %v %v", opts.Method, target)
			panic(runtime.NewGoError(errors.New("Unable to fetch resources from HTTP endpoint. Check logs for more information")))
		}
		defer resp.Body.Close()
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			entry := log.Error().Err(err).Str("target", target).Str("method", opts.Method)
			modutils.AppendCallStack(entry, runtime).Msgf("Unable to read body from HTTP %v %v", opts.Method, target)
			panic(runtime.NewGoError(errors.New("Unable to fetch resources from HTTP endpoint. Check logs for more information")))
		}
		obj := runtime.NewObject()
		obj.Set("statusCode", resp.StatusCode)
		obj.Set("status", resp.Status)
		obj.Set("headers", runtime.ToValue(resp.Header))
		obj.Set("bytes", runtime.ToValue(bodyBytes))
		return obj
	}
}
