package fluxpicker

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	yaml "github.com/goccy/go-yaml"

	"github.com/nimble-giant/ailloy/internal/tui/foundries/data"
)

// writeFluxFile writes overrides into a YAML file at path, merging with any
// existing content (overrides win for keys present in both). Dotted override
// keys are expanded into nested maps.
func writeFluxFile(path string, overrides map[string]any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	existing := map[string]any{}
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
	return os.WriteFile(path, out, 0o644)
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
func persistOverrides(scope data.Scope, moldName string, target SaveTarget, overrides map[string]any) (string, error) {
	_ = scope // reserved for future use
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
	return "", fmt.Errorf("unknown save target %d", target)
}

// lastPathSegment returns the last segment after '/'. For "official/agents"
// returns "agents". For "trailing/" returns "".
func lastPathSegment(s string) string {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == '/' {
			return s[i+1:]
		}
	}
	return s
}
