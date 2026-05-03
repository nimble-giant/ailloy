package mold

import (
	"os"
	"path/filepath"
	"strings"
)

// FluxFileSlug derives a deterministic, filesystem-safe filename stem from a
// mold ref. The full source is preserved (with separators replaced) so molds
// with the same final segment under different foundries — including nested
// foundries that re-export shared molds — don't collide on disk.
//
//	"github.com/nimble-giant/agents@v1"  → "github.com_nimble-giant_agents_v1"
//	"nimble-giant/agents"                → "nimble-giant_agents"
//	"agents"                             → "agents"
//
// Anything that isn't [A-Za-z0-9._-] is collapsed to '_'.
func FluxFileSlug(ref string) string {
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
	for len(out) > 0 && out[len(out)-1] == '_' {
		out = out[:len(out)-1]
	}
	if len(out) == 0 {
		return "mold"
	}
	return string(out)
}

// PersistedFluxPaths returns the existing persisted flux files for the given
// mold ref, in load order (global, then project). Files that don't exist are
// omitted. Empty ref returns nil.
//
// Layering order matches Helm conventions: more specific (project) wins over
// less specific (global). Both sit between the mold's built-in defaults and
// any user-supplied -f files.
func PersistedFluxPaths(ref string) []string {
	if strings.TrimSpace(ref) == "" {
		return nil
	}
	slug := FluxFileSlug(ref)
	var paths []string
	if home, err := os.UserHomeDir(); err == nil {
		p := filepath.Join(home, ".ailloy", "flux", slug+".yaml")
		if persistedFluxFileExists(p) {
			paths = append(paths, p)
		}
	}
	p := filepath.Join(".ailloy", "flux", slug+".yaml")
	if persistedFluxFileExists(p) {
		paths = append(paths, p)
	}
	return paths
}

func persistedFluxFileExists(path string) bool {
	info, err := os.Stat(path) // #nosec G304 -- path built from sanitized slug under known prefixes
	if err != nil {
		return false
	}
	return !info.IsDir()
}
