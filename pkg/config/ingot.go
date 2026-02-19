package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/nimble-giant/ailloy/pkg/mold"
	"github.com/nimble-giant/ailloy/pkg/safepath"
)

// IngotResolver resolves {{ingot "name"}} template function calls by searching
// for ingot content across a list of directories. It supports both manifest-based
// ingots (directory with ingot.yaml) and bare file ingots (name.md).
type IngotResolver struct {
	SearchPaths []string
	Flux        map[string]string
	Ore         *Ore
	resolving   map[string]bool
}

// NewIngotResolver creates a resolver that searches the given paths in order.
func NewIngotResolver(searchPaths []string, flux map[string]string, ore *Ore) *IngotResolver {
	return &IngotResolver{
		SearchPaths: searchPaths,
		Flux:        flux,
		Ore:         ore,
		resolving:   make(map[string]bool),
	}
}

// Resolve finds and renders an ingot by name. It searches each path for a
// directory with an ingot.yaml manifest first, then falls back to a bare .md file.
// The ingot content is rendered through the same template engine with the same
// flux and ore context. Circular references are detected and reported as errors.
func (r *IngotResolver) Resolve(name string) (string, error) {
	if r.resolving[name] {
		return "", fmt.Errorf("circular ingot reference detected: %s", name)
	}
	r.resolving[name] = true
	defer delete(r.resolving, name)

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

	ingot, err := mold.ParseIngot(data)
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
	return ProcessTemplate(content, r.Flux, r.Ore, WithIngotResolver(r))
}
