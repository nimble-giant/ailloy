package templates

import (
	"fmt"
	"io/fs"
	"os"
	"strings"

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

// GetTemplate returns the content of a command template file.
func (r *MoldReader) GetTemplate(name string) ([]byte, error) {
	path := fmt.Sprintf(".claude/commands/%s", name)
	return fs.ReadFile(r.fsys, path)
}

// ListTemplates returns a list of available command template files.
func (r *MoldReader) ListTemplates() ([]string, error) {
	entries, err := fs.ReadDir(r.fsys, ".claude/commands")
	if err != nil {
		return nil, err
	}

	var templates []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
			templates = append(templates, entry.Name())
		}
	}

	return templates, nil
}

// GetSkill returns the content of a skill file.
func (r *MoldReader) GetSkill(name string) ([]byte, error) {
	path := fmt.Sprintf(".claude/skills/%s", name)
	return fs.ReadFile(r.fsys, path)
}

// ListSkills returns a list of available skill files.
func (r *MoldReader) ListSkills() ([]string, error) {
	entries, err := fs.ReadDir(r.fsys, ".claude/skills")
	if err != nil {
		return nil, err
	}

	var skills []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
			skills = append(skills, entry.Name())
		}
	}

	return skills, nil
}

// GetWorkflowTemplate returns the content of a GitHub workflow template file.
func (r *MoldReader) GetWorkflowTemplate(name string) ([]byte, error) {
	path := fmt.Sprintf(".github/workflows/%s", name)
	return fs.ReadFile(r.fsys, path)
}

// ListWorkflowTemplates returns a list of available GitHub workflow template files.
func (r *MoldReader) ListWorkflowTemplates() ([]string, error) {
	entries, err := fs.ReadDir(r.fsys, ".github/workflows")
	if err != nil {
		return nil, err
	}

	var templates []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".yml") {
			templates = append(templates, entry.Name())
		}
	}

	return templates, nil
}
