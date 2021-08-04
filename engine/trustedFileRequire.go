package engine

import (
	"errors"
	"fmt"
	"path"
	"path/filepath"

	"github.com/dop251/goja"
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

func (tf *trustedFileRequire) resolvePathTo(name string) (absPath string, relativePath string, err error) {
	relativePath = path.Join(tf.dir, name)
	// TODO: think if this extra precaution is really useful
	// tf.root.anchor should be absolute already
	// but it doesn't hurt to add it here
	absPath, err = filepath.Abs(filepath.Join(tf.root.anchor, filepath.FromSlash(relativePath)))
	return
}

func (tf *trustedFileRequire) loadModule(name string, relativePath string, absPath string) *moduleDef {
	panic(tf.root.e.runtime.NewGoError(errors.New("not implemented yet cuz it is time to got to bed!")))
}
