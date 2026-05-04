package mold

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"sort"
)

// LoadOreOverlaysFromFS scans <root>/*/ for ore packages and returns one
// OverlaySchema per ore (with entries prefixed by ore.<namespace>.)
// plus a defaults map of the form { "ore": { <namespace>: <flux.yaml contents> } }.
//
// The namespace is resolved with the consumer-overrides-publisher precedence
// chain documented on Ore.Namespace:
//   - When the install-dir name differs from ore.Name (i.e. an alias was
//     applied via mold.yaml `as:` or `ailloy ore add --as`), the install-dir
//     name wins — that's the alias.
//   - Otherwise, the publisher's declared namespace wins (Ore.Namespace if
//     set, else Ore.Name via EffectiveNamespace).
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
		ore, err := LoadOreFromFS(fsys, manifestPath)
		if err != nil {
			return nil, nil, fmt.Errorf("loading ore at %s: %w", oreDir, err)
		}

		ns := resolveOreNamespace(name, ore)

		// Schema overlay
		schema, err := LoadFluxSchema(fsys, path.Join(oreDir, "flux.schema.yaml"))
		if err != nil {
			return nil, nil, fmt.Errorf("loading ore schema at %s: %w", oreDir, err)
		}
		prefix := "ore." + ns + "."
		prefixed := make([]FluxVar, 0, len(schema))
		for _, e := range schema {
			pe := e
			pe.Name = prefix + e.Name
			prefixed = append(prefixed, pe)
		}
		overlays = append(overlays, OverlaySchema{
			Source:  "ore:" + ns,
			Entries: prefixed,
		})

		// Defaults overlay (wrap unprefixed flux.yaml under ore.<ns>:)
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
		oreNs[ns] = oreDefaults
	}
	return overlays, defaults, nil
}

// resolveOreNamespace implements the publisher-vs-consumer precedence chain
// for an ore loaded from .ailloy/ores/<installDirName>/. Highest-priority
// layers (mold.yaml `as:` and `--as <alias>`) are reflected in
// installDirName; this helper layers the publisher's manifest on top:
//
//   - If installDirName != ore.Name, an alias was applied — return the
//     install-dir name (the alias).
//   - Otherwise, return EffectiveNamespace (Ore.Namespace if set, else
//     Ore.Name).
func resolveOreNamespace(installDirName string, o *Ore) string {
	if o == nil {
		return installDirName
	}
	if installDirName != o.Name {
		return installDirName
	}
	return o.EffectiveNamespace()
}
