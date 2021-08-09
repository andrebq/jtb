package engine

import (
	"fmt"
	"path"
	"path/filepath"

	"github.com/dop251/goja"
	"github.com/spf13/afero"
)

type (
	trustedFileRequire struct {
		root *rootRequire
		dir  string
	}
)

func (tf *trustedFileRequire) require(name string) goja.Value {
	absPath, relativePath, err := tf.resolvePathTo(name)
	if err != nil {
		panic(tf.root.e.runtime.NewGoError(fmt.Errorf("Unable to resolve path to %v", name)))
	}
	if def := tf.root.hasModule(absPath); def != nil {
		return def.exports
	}
	tf.root.saveModule(absPath, tf.loadModule(name, relativePath, absPath))
	return tf.root.hasModule(absPath).exports
}

func (tf *trustedFileRequire) javascriptRequire(call goja.FunctionCall) goja.Value {
	name := call.Argument(0).ToString().Export().(string)
	return tf.require(name)
}

func (tf *trustedFileRequire) resolvePathTo(name string) (absPath string, relativePath string, err error) {
	relativePath = path.Join(tf.dir, name)
	// TODO: think if this extra precaution is really useful
	// tf.root.anchor should be absolute already
	// but it doesn't hurt to add it here
	absPath, err = filepath.Abs(filepath.Join(tf.root.anchor, filepath.FromSlash(path.Clean(relativePath))))
	return
}

func (tf *trustedFileRequire) loadModule(name string, relativePath string, absPath string) *moduleDef {
	code, err := tf.parseCode(name, relativePath)
	if err != nil {
		panic(tf.root.e.runtime.NewGoError(fmt.Errorf("Unable to parse %v, cause: %w", name, err)))
	}
	moduleOutput, err := tf.root.e.runtime.RunProgram(code)
	if err != nil {
		panic(tf.root.e.runtime.NewGoError(fmt.Errorf("Unable to load %v, cause %w", name, err)))
	}
	loader, ok := goja.AssertFunction(moduleOutput)
	if !ok {
		panic("This should never ever happen! There some really really wrong with jtb!!!")
	}
	this := tf.root.e.runtime.NewObject()
	exports := tf.root.e.runtime.NewObject()
	sub := tf.sub(relativePath)
	requireFn := tf.root.e.runtime.ToValue(sub.javascriptRequire)
	this.Set("require", requireFn)
	this.Set("exports", exports)
	_, err = loader(this, exports, requireFn)
	if err != nil {
		// err is a GoError
		panic(err)
	}
	return &moduleDef{
		exports: exports,
	}
}

func (tf *trustedFileRequire) sub(relpath string) *trustedFileRequire {
	return &trustedFileRequire{
		root: tf.root,
		dir:  path.Dir(relpath),
	}
}

func (tf *trustedFileRequire) parseCode(name string, relPath string) (*goja.Program, error) {
	bytes, err := afero.ReadFile(tf.root.e.fs, relPath)
	if err != nil {
		return nil, err
	}

	safeCode := fmt.Sprintf(`(function(exports, require) {
		Object.freeze(require);
		(function(){
			%v
			;
		}).apply(this);
		Object.freeze(exports);
	})`, string(bytes))

	program, err := goja.Compile(name, safeCode, true)
	if err != nil {
		return nil, err
	}
	return program, nil
}
