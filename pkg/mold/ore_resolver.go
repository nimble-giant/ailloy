package mold

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"sort"
)

// LoadOreOverlaysFromFS scans <root>/*/ for ore packages and returns one
// OverlaySchema per ore (with entries prefixed by ore.<install-dir-name>.)
// plus a defaults map of the form { "ore": { <install-dir-name>: <flux.yaml contents> } }.
//
// The install directory name (not the ore.yaml's "name" field) is used as
// the namespace, so callers that installed an ore --as <alias> get the alias
// reflected in the prefix.
//
// `seen` lets the caller skip ores already loaded from a higher-priority
// search path. Pass nil to load everything.
func LoadOreOverlaysFromFS(fsys fs.FS, root string, seen map[string]struct{}) ([]OverlaySchema, map[string]any, error) {
	entries, err := fs.ReadDir(fsys, root)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			// No ores dir — empty result, not an error.
			return nil, map[string]any{}, nil
		}
		return nil, nil, fmt.Errorf("reading ore search root %s: %w", root, err)
	}

	defaults := map[string]any{}
	var overlays []OverlaySchema
	var dirNames []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dirNames = append(dirNames, e.Name())
	}
	sort.Strings(dirNames)

	for _, name := range dirNames {
		if seen != nil {
			if _, dup := seen[name]; dup {
				continue
			}
			seen[name] = struct{}{}
		}
		oreDir := path.Join(root, name)
		manifestPath := path.Join(oreDir, "ore.yaml")
		if _, err := LoadOreFromFS(fsys, manifestPath); err != nil {
			return nil, nil, fmt.Errorf("loading ore at %s: %w", oreDir, err)
		}

		// Schema overlay
		schema, err := LoadFluxSchema(fsys, path.Join(oreDir, "flux.schema.yaml"))
		if err != nil {
			return nil, nil, fmt.Errorf("loading ore schema at %s: %w", oreDir, err)
		}
		prefix := "ore." + name + "."
		prefixed := make([]FluxVar, 0, len(schema))
		for _, e := range schema {
			pe := e
			pe.Name = prefix + e.Name
			prefixed = append(prefixed, pe)
		}
		overlays = append(overlays, OverlaySchema{
			Source:  "ore:" + name,
			Entries: prefixed,
		})

		// Defaults overlay (wrap unprefixed flux.yaml under ore.<name>:)
		oreDefaults, err := LoadFluxFile(fsys, path.Join(oreDir, "flux.yaml"))
		if err != nil {
			return nil, nil, fmt.Errorf("loading ore defaults at %s: %w", oreDir, err)
		}
		if oreDefaults == nil {
			oreDefaults = map[string]any{}
		}
		oreNs, _ := defaults["ore"].(map[string]any)
		if oreNs == nil {
			oreNs = map[string]any{}
			defaults["ore"] = oreNs
		}
		oreNs[name] = oreDefaults
	}
	return overlays, defaults, nil
}
