package mold

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/nimble-giant/ailloy/pkg/safepath"
)

// IngotResolver resolves {{ingot "name"}} template function calls by searching
// for ingot content across a list of directories. It supports both manifest-based
// ingots (directory with ingot.yaml) and bare file ingots (name.md).
//
// FS, when non-nil, is searched (at "ingots/<name>") before the disk SearchPaths.
// This is how a stuffed-binary cast resolves the mold's own ingots: the mold
// lives in an embedded fs.FS, not on disk, so a purely path-based search would
// miss them.
type IngotResolver struct {
	FS          fs.FS
	SearchPaths []string
	Flux        map[string]any
	resolving   map[string]bool
}

// NewIngotResolver creates a resolver that searches the given paths in order.
func NewIngotResolver(searchPaths []string, flux map[string]any) *IngotResolver {
	return &IngotResolver{
		SearchPaths: searchPaths,
		Flux:        flux,
		resolving:   make(map[string]bool),
	}
}

// NewIngotResolverWithFS is NewIngotResolver plus an fs.FS (e.g. the mold's
// embedded filesystem) searched before the disk SearchPaths.
func NewIngotResolverWithFS(fsys fs.FS, searchPaths []string, flux map[string]any) *IngotResolver {
	r := NewIngotResolver(searchPaths, flux)
	r.FS = fsys
	return r
}

// Resolve finds and renders an ingot by name. It searches each path for a
// directory with an ingot.yaml manifest first, then falls back to a bare .md file.
// The ingot content is rendered through the same template engine with the same
// flux context. Circular references are detected and reported as errors.
func (r *IngotResolver) Resolve(name string) (string, error) {
	if r.resolving[name] {
		return "", fmt.Errorf("circular ingot reference detected: %s", name)
	}
	r.resolving[name] = true
	defer delete(r.resolving, name)

	// Search the embedded/mold fs.FS first (stuffed-binary casts: the mold and
	// its ingots live in this FS, not on disk).
	if r.FS != nil {
		if content, err := r.resolveManifestFS(path.Join("ingots", name, "ingot.yaml"), name); err == nil {
			return r.render(content)
		}
		if content, err := fs.ReadFile(r.FS, path.Join("ingots", name+".md")); err == nil {
			return r.render(string(content))
		}
	}

	for _, base := range r.SearchPaths {
		// Try directory with manifest first
		manifestPath := filepath.Join(base, "ingots", name, "ingot.yaml")
		if content, err := r.resolveManifest(manifestPath, name); err == nil {
			return r.render(content)
		}

		// Fall back to bare file
		barePath := filepath.Join(base, "ingots", name+".md")
		if content, err := r.readFile(barePath); err == nil {
			return r.render(string(content))
		}
	}

	searched := make([]string, len(r.SearchPaths))
	for i, p := range r.SearchPaths {
		searched[i] = filepath.Join(p, "ingots")
	}
	return "", fmt.Errorf("ingot %q not found (searched: %s)", name, strings.Join(searched, ", "))
}

// resolveManifest loads an ingot.yaml manifest and concatenates all listed files.
func (r *IngotResolver) resolveManifest(manifestPath, name string) (string, error) {
	cleanPath, err := safepath.Clean(manifestPath)
	if err != nil {
		return "", err
	}

	data, err := os.ReadFile(cleanPath) // #nosec G304 -- path sanitized by safepath.Clean
	if err != nil {
		return "", err
	}

	ingot, err := ParseIngot(data)
	if err != nil {
		return "", fmt.Errorf("parsing ingot %q manifest: %w", name, err)
	}

	ingotDir := filepath.Dir(cleanPath)
	var combined strings.Builder
	for _, f := range ingot.Files {
		filePath, err := safepath.Join(ingotDir, f)
		if err != nil {
			return "", fmt.Errorf("ingot %q file %q: %w", name, f, err)
		}
		content, err := os.ReadFile(filePath) // #nosec G304 -- path sanitized by safepath.Join
		if err != nil {
			return "", fmt.Errorf("reading ingot %q file %q: %w", name, f, err)
		}
		combined.Write(content)
	}

	return combined.String(), nil
}

// resolveManifestFS loads an ingot.yaml manifest from r.FS and concatenates all
// listed files. The fs.FS analogue of resolveManifest for stuffed-binary casts.
func (r *IngotResolver) resolveManifestFS(manifestPath, name string) (string, error) {
	data, err := fs.ReadFile(r.FS, manifestPath)
	if err != nil {
		return "", err
	}
	ingot, err := ParseIngot(data)
	if err != nil {
		return "", fmt.Errorf("parsing ingot %q manifest: %w", name, err)
	}
	ingotDir := path.Dir(manifestPath)
	var combined strings.Builder
	for _, f := range ingot.Files {
		fp := path.Join(ingotDir, f)
		if !fs.ValidPath(fp) {
			return "", fmt.Errorf("ingot %q file %q: invalid path", name, f)
		}
		content, err := fs.ReadFile(r.FS, fp)
		if err != nil {
			return "", fmt.Errorf("reading ingot %q file %q: %w", name, f, err)
		}
		combined.Write(content)
	}
	return combined.String(), nil
}

// readFile reads a file with path sanitization.
func (r *IngotResolver) readFile(path string) ([]byte, error) {
	cleanPath, err := safepath.Clean(path)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(cleanPath) // #nosec G304 -- path sanitized by safepath.Clean
}

// render processes ingot content through the template engine with the same context.
func (r *IngotResolver) render(content string) (string, error) {
	return ProcessTemplate(content, r.Flux, WithIngotResolver(r))
}
