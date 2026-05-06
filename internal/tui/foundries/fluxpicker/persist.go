package fluxpicker

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	yaml "github.com/goccy/go-yaml"

	"github.com/nimble-giant/ailloy/pkg/mold"
)

// writeFluxFile writes overrides into a YAML file at path, merging with any
// existing content (overrides win for keys present in both). Dotted override
// keys are expanded into nested maps. The write is atomic: the new contents
// land in a sibling temp file and are renamed into place, so a crash mid-write
// cannot leave the user's flux file truncated.
func writeFluxFile(path string, overrides map[string]any) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return err
	}
	existing := map[string]any{}
	// #nosec G304 -- path is built from a sanitized mold name plus a fixed
	// project- or home-rooted prefix; not user-controlled at the moment of read.
	if b, err := os.ReadFile(path); err == nil {
		_ = yaml.Unmarshal(b, &existing)
	}
	for k, v := range overrides {
		if err := setDottedKey(existing, k, v); err != nil {
			return err
		}
	}
	out, err := yaml.Marshal(existing)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".flux-*.yaml")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(out); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	if err := os.Chmod(tmpPath, 0o600); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return nil
}

// setDottedKey writes value into m at the nested location implied by a dotted
// key (e.g. "agents.targets" → m["agents"]["targets"] = value).
func setDottedKey(m map[string]any, key string, value any) error {
	parts := strings.Split(key, ".")
	cur := m
	for i, p := range parts {
		if i == len(parts)-1 {
			cur[p] = value
			return nil
		}
		next, _ := cur[p].(map[string]any)
		if next == nil {
			next = map[string]any{}
			cur[p] = next
		}
		cur = next
	}
	return nil
}

// resolveProjectPath returns the project-scoped flux file path for a mold.
func resolveProjectPath(moldName string) string {
	return filepath.Join(".ailloy", "flux", moldName+".yaml")
}

// resolveGlobalPath returns the global-scoped flux file path for a mold.
func resolveGlobalPath(moldName string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".ailloy", "flux", moldName+".yaml"), nil
}

// fluxFileSlug delegates to the shared slug helper. Kept as a package-local
// alias so existing call sites stay readable.
var fluxFileSlug = mold.FluxFileSlug

// persistOverrides routes overrides to the chosen save target. Returns the
// path written (empty string for SaveTargetSession).
func persistOverrides(moldName string, target SaveTarget, overrides map[string]any) (string, error) {
	switch target {
	case SaveTargetSession:
		return "", nil
	case SaveTargetProject:
		path := resolveProjectPath(moldName)
		return path, writeFluxFile(path, overrides)
	case SaveTargetGlobal:
		path, err := resolveGlobalPath(moldName)
		if err != nil {
			return "", err
		}
		return path, writeFluxFile(path, overrides)
	}
	return "", fmt.Errorf("unknown save target %v", target)
}

// persistFoundryOverrides writes foundry-scope picker output to per-mold flux
// files. Returns the list of paths written.
//
// `unified` carries values from non-conflicting picker rows (apply to every
// mold whose schema declares the key). `perMold` carries values from
// conflict-expanded sub-rows (apply to that single mold only). Per-mold
// values win over unified values for the same key on the same mold. The slug
// used for each file is mold.FluxFileSlug(perMoldSourceRefs[moldName]) so the
// cast pipeline's PersistedFluxPaths picks them up on subsequent casts.
//
// SaveTargetSession is a no-op (returns nil paths) because session overrides
// live in the picker model and are forwarded by the App via FluxOverridesMsg.
func persistFoundryOverrides(
	target SaveTarget,
	perMoldSchemas map[string][]mold.FluxVar,
	perMoldSourceRefs map[string]string,
	unified map[string]any,
	perMold map[string]map[string]any,
) ([]string, error) {
	if target == SaveTargetSession {
		return nil, nil
	}

	// Build per-mold combined override map.
	combined := map[string]map[string]any{}
	for moldName, schema := range perMoldSchemas {
		for _, v := range schema {
			if val, ok := unified[v.Name]; ok {
				if combined[moldName] == nil {
					combined[moldName] = map[string]any{}
				}
				combined[moldName][v.Name] = val
			}
		}
	}
	for moldName, kv := range perMold {
		if combined[moldName] == nil {
			combined[moldName] = map[string]any{}
		}
		for k, v := range kv {
			combined[moldName][k] = v
		}
	}

	var written []string
	for moldName, overrides := range combined {
		if len(overrides) == 0 {
			continue
		}
		ref := perMoldSourceRefs[moldName]
		if ref == "" {
			// Fall back to mold name as a deterministic slug input. The cast
			// pipeline won't pick this up via PersistedFluxPaths, but the
			// file is still readable for manual inspection.
			ref = moldName
		}
		slug := fluxFileSlug(ref)
		var path string
		switch target {
		case SaveTargetProject:
			path = filepath.Join(".ailloy", "flux", slug+".yaml")
		case SaveTargetGlobal:
			home, err := os.UserHomeDir()
			if err != nil {
				return written, err
			}
			path = filepath.Join(home, ".ailloy", "flux", slug+".yaml")
		default:
			return written, fmt.Errorf("unknown save target %v", target)
		}
		if err := writeFluxFile(path, overrides); err != nil {
			return written, err
		}
		written = append(written, path)
	}
	sort.Strings(written)
	return written, nil
}

// mergeOverrides returns a new map combining defaults and overrides
// (overrides win at any nested key).
func mergeOverrides(defaults map[string]any, overrides map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range defaults {
		out[k] = v
	}
	for k, v := range overrides {
		if err := setDottedKey(out, k, v); err != nil {
			continue
		}
	}
	return out
}
