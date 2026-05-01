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
	root string
}

// NewMoldReader creates a MoldReader from an fs.FS. The mold has no on-disk
// root path (e.g., embedded or in-memory).
func NewMoldReader(fsys fs.FS) *MoldReader {
	return &MoldReader{fsys: fsys}
}

// NewMoldReaderFromFS creates a MoldReader from an fs.FS that is known to be
// rooted at an on-disk directory. The root is used to locate sibling
// directories (e.g., bundled ingots) during template rendering.
func NewMoldReaderFromFS(fsys fs.FS, root string) *MoldReader {
	return &MoldReader{fsys: fsys, root: root}
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
	return &MoldReader{fsys: os.DirFS(moldDir), root: moldDir}, nil
}

// FS returns the underlying filesystem.
func (r *MoldReader) FS() fs.FS {
	return r.fsys
}

// Root returns the on-disk path the mold is rooted at, or an empty string
// for readers backed by an in-memory or embedded filesystem.
func (r *MoldReader) Root() string {
	return r.root
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
