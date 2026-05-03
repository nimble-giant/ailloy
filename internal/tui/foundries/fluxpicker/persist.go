package fluxpicker

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	yaml "github.com/goccy/go-yaml"
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

// fluxFileSlug derives a deterministic, filesystem-safe filename stem from a
// mold ref. The full source is preserved (with separators replaced) so molds
// with the same final segment under different foundries — including nested
// foundries that can re-export molds — don't collide on disk.
//
//	"github.com/nimble-giant/agents@v1"  → "github.com_nimble-giant_agents_v1"
//	"nimble-giant/agents"                → "nimble-giant_agents"
//	"agents"                             → "agents"
//
// Anything that isn't [A-Za-z0-9._-] is collapsed to '_'.
func fluxFileSlug(ref string) string {
	ref = strings.TrimSuffix(strings.TrimSpace(ref), ".git")
	if ref == "" {
		return "mold"
	}
	out := make([]byte, 0, len(ref))
	for i := 0; i < len(ref); i++ {
		c := ref[i]
		switch {
		case c >= 'a' && c <= 'z',
			c >= 'A' && c <= 'Z',
			c >= '0' && c <= '9',
			c == '.', c == '-':
			out = append(out, c)
		default:
			if len(out) > 0 && out[len(out)-1] != '_' {
				out = append(out, '_')
			}
		}
	}
	// Trim trailing underscore from collapsed runs at the tail.
	for len(out) > 0 && out[len(out)-1] == '_' {
		out = out[:len(out)-1]
	}
	if len(out) == 0 {
		return "mold"
	}
	return string(out)
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
