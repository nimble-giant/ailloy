package smelt

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/knadh/stuffbin"
)

// ErrNoEmbeddedMold indicates the current binary does not contain a stuffed mold.
var ErrNoEmbeddedMold = errors.New("no embedded mold in binary")

// HasEmbeddedMold returns true if the current binary has a stuffed mold.
func HasEmbeddedMold() bool {
	execPath, err := resolveExecutable()
	if err != nil {
		return false
	}
	_, err = stuffbin.GetFileID(execPath)
	return err == nil
}

// OpenEmbeddedMold returns an fs.FS backed by the mold files stuffed into
// the current binary. Returns ErrNoEmbeddedMold if the binary is not stuffed.
func OpenEmbeddedMold() (fs.FS, error) {
	execPath, err := resolveExecutable()
	if err != nil {
		return nil, ErrNoEmbeddedMold
	}

	fsys, err := UnstuffFS(execPath)
	if err != nil {
		if errors.Is(err, stuffbin.ErrNoID) {
			return nil, ErrNoEmbeddedMold
		}
		return nil, err
	}
	return fsys, nil
}

// resolveExecutable returns the resolved path to the current executable.
func resolveExecutable() (string, error) {
	execPath, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(execPath)
}
