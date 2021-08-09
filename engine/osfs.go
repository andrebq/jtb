package engine

import (
	"github.com/spf13/afero"
)

func OpenOSFilesystem(base string) afero.Fs {
	return afero.NewBasePathFs(afero.NewReadOnlyFs(afero.NewOsFs()), base)
}
