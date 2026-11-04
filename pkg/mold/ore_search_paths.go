package mold

import (
	"github.com/nimble-giant/ailloy/pkg/projectstate"
	"io/fs"
	"os"
)

// BuildDefaultOreSearchPaths returns the canonical ore-search-path order used
// by the cast pipeline (and any other consumer that wants to mirror it):
//
//  1. mold-local — ores under the mold's own filesystem at "ores/".
//     Highest priority; keys here shadow project/global ores of the same
//     namespace.
//  2. project — ores installed for the current project under
//     "<project state dir>/ores" (~/.ailloy/projects/<project>/ores).
//     Honored even when the caller is doing a global cast, because users
//     may have project-installed ores they want to layer in.
//  3. global — ores installed into the user's home directory under
//     ".ailloy/ores". Lowest priority; only contributes namespaces not
//     already provided by mold-local or project. Skipped if the home dir
//     cannot be determined.
//
// Lower-priority entries only contribute ore namespaces not already seen,
// mirroring how flux defaults are layered.
//
// The global flag is currently unused — it does not change the search-path
// order. It is kept on the signature so callers can express intent and so a
// future change (e.g. global cast skips project ores for strict isolation)
// can land without re-threading every call site.
func BuildDefaultOreSearchPaths(moldFS fs.FS, global bool) []OreSearchPath {
	paths := []OreSearchPath{
		{Name: "mold-local", FS: moldFS, Root: "ores"},
	}
	if stateDir := projectstate.Dir(); stateDir != "" {
		paths = append(paths, OreSearchPath{
			Name: "project",
			FS:   os.DirFS(stateDir),
			Root: "ores",
		})
	}
	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths, OreSearchPath{
			Name: "global",
			FS:   os.DirFS(home),
			Root: ".ailloy/ores",
		})
	}
	_ = global // currently only affects install-dir, not search-path order
	return paths
}
