package mold

import (
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"
)

// OreSource identifies one resolved ore dependency: its namespace, the fs
// to read its blank/source files from, and any output mapping it declared
// via its flux.yaml `output:` block.
type OreSource struct {
	Namespace string         // the effective namespace (e.g. "agent_targets")
	FS        fs.FS          // the ore's filesystem root
	Output    map[string]any // ore-supplied output entries (may be nil)
}

// ResolveFilesWithOreSources is the composite-fs aware sibling of
// ResolveFiles. It walks the consumer mold's `output` mapping (resolving
// `from: ore/<ns>/...` selectors against the matching OreSource.FS) and
// also walks each ore source's own Output mapping against its own FS.
// Results are merged, deterministically sorted by DestPath, and every
// ResolvedFile carries its SrcFS and Origin so callers can read the right
// bytes downstream.
//
// `output` may be nil (mold-only auto-discovery, identical to ResolveFiles).
// `oreSources` may be nil/empty — in that case this function reduces to a
// thin wrapper around ResolveFiles.
func ResolveFilesWithOreSources(output any, moldFS fs.FS, oreSources []OreSource, opts ...ResolveOption) ([]ResolvedFile, error) {
	oreByNS := make(map[string]OreSource, len(oreSources))
	for _, s := range oreSources {
		oreByNS[s.Namespace] = s
	}

	// Extract `from:` entries from the consumer output map so they can be
	// resolved against ore filesystems rather than moldFS. ResolveFiles only
	// knows how to walk moldFS; `from:` entries that point at ore/<ns>/...
	// would fail stat checks if passed through unchanged.
	fromEntries, moldOutput, err := splitFromEntries(output)
	if err != nil {
		return nil, err
	}

	// Resolve consumer mold entries (without `from:` entries) against moldFS.
	primary, err := ResolveFiles(moldOutput, moldFS, opts...)
	if err != nil {
		return nil, err
	}

	resolved := make([]ResolvedFile, 0, len(primary)+len(fromEntries))

	// Annotate primary entries with their moldFS source.
	for _, r := range primary {
		if r.SrcFS == nil {
			r.SrcFS = moldFS
		}
		resolved = append(resolved, r)
	}

	// Resolve `from: ore/<ns>/<path>` consumer entries against ore filesystems.
	for _, fe := range fromEntries {
		ns, oreRelPath, ferr := parseOreFromSelector(fe.from)
		if ferr != nil {
			return nil, ferr
		}
		src, ok := oreByNS[ns]
		if !ok {
			return nil, fmt.Errorf("output entry for %q references ore %q but no such ore dependency is declared", fe.dest, ns)
		}
		if _, ferr := fs.Stat(src.FS, oreRelPath); ferr != nil {
			return nil, fmt.Errorf("output entry for %q references %q which does not exist in ore %q: %w", fe.dest, oreRelPath, ns, ferr)
		}
		resolved = append(resolved, ResolvedFile{
			SrcPath:  oreRelPath,
			DestPath: fe.dest,
			Process:  fe.process,
			Set:      fe.set,
			Strategy: fe.strategy,
			SrcFS:    src.FS,
			Origin:   ns,
		})
	}

	// Resolve each ore's own output mapping against its own FS.
	for _, src := range oreSources {
		if len(src.Output) == 0 {
			continue
		}
		oreResolved, err := ResolveFiles(src.Output, src.FS, opts...)
		if err != nil {
			return nil, fmt.Errorf("resolving ore %q output: %w", src.Namespace, err)
		}
		for _, r := range oreResolved {
			r.SrcFS = src.FS
			r.Origin = src.Namespace
			resolved = append(resolved, r)
		}
	}

	sort.Slice(resolved, func(i, j int) bool {
		if resolved[i].DestPath != resolved[j].DestPath {
			return resolved[i].DestPath < resolved[j].DestPath
		}
		return resolved[i].SrcPath < resolved[j].SrcPath
	})
	return resolved, nil
}

// fromEntry holds a parsed `from: ore/<ns>/<path>` consumer output entry.
type fromEntry struct {
	from     string
	dest     string
	process  bool
	set      map[string]any
	strategy string
}

// splitFromEntries separates consumer output entries that carry a `from:` key
// from those that do not. It returns:
//   - fromEntries: the parsed `from:` entries
//   - remainder: the original output value with `from:` keys stripped (suitable for ResolveFiles)
//
// If output is not a map[string]any, fromEntries will be empty and the
// original output is returned unchanged.
func splitFromEntries(output any) ([]fromEntry, any, error) {
	m, ok := output.(map[string]any)
	if !ok {
		return nil, output, nil
	}

	var fromEntries []fromEntry
	remainder := make(map[string]any, len(m))

	for key, val := range m {
		switch v := val.(type) {
		case map[string]any:
			from, hasFrom := v["from"].(string)
			dest, _ := v["dest"].(string)
			if hasFrom && from != "" {
				process := true
				if p, ok := v["process"].(bool); ok {
					process = p
				}
				var set map[string]any
				if s, ok := v["set"].(map[string]any); ok {
					set = s
				}
				var strategy string
				if s, ok := v["strategy"].(string); ok {
					strategy = s
				}
				if dest == "" {
					dest = key
				}
				fromEntries = append(fromEntries, fromEntry{
					from:     from,
					dest:     dest,
					process:  process,
					set:      set,
					strategy: strategy,
				})
				// Do not include in remainder — ResolveFiles cannot handle these.
				continue
			}
			remainder[key] = val
		case []any:
			// A list may contain a mix of from: entries and regular entries.
			// Split them out.
			var regularList []any
			for _, entry := range v {
				em, ok := entry.(map[string]any)
				if !ok {
					regularList = append(regularList, entry)
					continue
				}
				from, hasFrom := em["from"].(string)
				dest, _ := em["dest"].(string)
				if hasFrom && from != "" {
					process := true
					if p, ok := em["process"].(bool); ok {
						process = p
					}
					var set map[string]any
					if s, ok := em["set"].(map[string]any); ok {
						set = s
					}
					var strategy string
					if s, ok := em["strategy"].(string); ok {
						strategy = s
					}
					if dest == "" {
						dest = key
					}
					fromEntries = append(fromEntries, fromEntry{
						from:     from,
						dest:     dest,
						process:  process,
						set:      set,
						strategy: strategy,
					})
				} else {
					regularList = append(regularList, entry)
				}
			}
			if len(regularList) > 0 {
				remainder[key] = regularList
			}
		default:
			remainder[key] = val
		}
	}

	// Sort fromEntries by dest for determinism.
	sort.Slice(fromEntries, func(i, j int) bool {
		return fromEntries[i].dest < fromEntries[j].dest
	})

	var remainderOut any
	if len(remainder) > 0 {
		remainderOut = remainder
	}
	return fromEntries, remainderOut, nil
}

// parseOreFromSelector splits a "ore/<namespace>/<path>" selector into its
// namespace and path parts. Empty namespace or path is an error.
func parseOreFromSelector(s string) (namespace string, relpath string, err error) {
	const prefix = "ore/"
	if !strings.HasPrefix(s, prefix) {
		return "", "", fmt.Errorf("from selector %q must start with %q (form: ore/<namespace>/<path>)", s, prefix)
	}
	rest := s[len(prefix):]
	slash := strings.IndexByte(rest, '/')
	if slash < 0 {
		return "", "", fmt.Errorf("from selector %q must include a path after the namespace", s)
	}
	ns := rest[:slash]
	p := path.Clean(rest[slash+1:])
	if ns == "" || p == "" || p == "." {
		return "", "", fmt.Errorf("from selector %q has empty namespace or path", s)
	}
	return ns, p, nil
}
