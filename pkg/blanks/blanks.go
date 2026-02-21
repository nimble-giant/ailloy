package blanks

import (
	"fmt"
	"io/fs"
	"os"

	"github.com/nimble-giant/ailloy/pkg/mold"
)

// MoldReader reads mold content from any fs.FS implementation.
type MoldReader struct {
	fsys fs.FS
}

// NewMoldReader creates a MoldReader from an fs.FS.
func NewMoldReader(fsys fs.FS) *MoldReader {
	return &MoldReader{fsys: fsys}
}

// NewMoldReaderFromPath creates a MoldReader rooted at a filesystem directory.
func NewMoldReaderFromPath(moldDir string) (*MoldReader, error) {
	info, err := os.Stat(moldDir)
	if err != nil {
		return nil, fmt.Errorf("mold directory %q: %w", moldDir, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("mold path %q is not a directory", moldDir)
	}
	return &MoldReader{fsys: os.DirFS(moldDir)}, nil
}

// FS returns the underlying filesystem.
func (r *MoldReader) FS() fs.FS {
	return r.fsys
}

// LoadManifest loads and parses the mold.yaml manifest.
func (r *MoldReader) LoadManifest() (*mold.Mold, error) {
	return mold.LoadMoldFromFS(r.fsys, "mold.yaml")
}

// LoadFluxDefaults loads the flux.yaml default values.
func (r *MoldReader) LoadFluxDefaults() (map[string]any, error) {
	return mold.LoadFluxFile(r.fsys, "flux.yaml")
}

// LoadFluxSchema loads the flux.schema.yaml validation schema.
// Returns nil if no schema file exists.
func (r *MoldReader) LoadFluxSchema() ([]mold.FluxVar, error) {
	return mold.LoadFluxSchema(r.fsys, "flux.schema.yaml")
}
