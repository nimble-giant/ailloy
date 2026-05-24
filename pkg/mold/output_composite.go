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
// Results are merged via MergeFluxOutput (consumer-wins / ore-ore-conflict
// semantics), deterministically sorted by DestPath, deduplicated, and every
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

	// Build ore overlays for MergeFluxOutput: consumer-wins semantics and
	// ore-ore source-path conflict detection.
	oreOverlays := make([]OreOutputOverlay, 0, len(oreSources))
	for _, src := range oreSources {
		if len(src.Output) > 0 {
			oreOverlays = append(oreOverlays, OreOutputOverlay{
				Source:  "ore:" + src.Namespace,
				Entries: src.Output,
			})
		}
	}

	var resolved []ResolvedFile

	// Distinguish explicit output map (consumer-wins merge path) from nil
	// output (auto-discovery path).
	consumerBaseMap, hasExplicitOutput := moldOutput.(map[string]any)

	if hasExplicitOutput {
		// Consumer has an explicit output: map. Merge with ore overlays so
		// consumer keys suppress matching ore keys and ore-ore conflicts error.
		mergedMap, report, merr := MergeFluxOutput(consumerBaseMap, oreOverlays)
		if merr != nil {
			return nil, merr
		}

		// Partition merged map by origin for per-FS resolution.
		consumerSub := make(map[string]any)
		oreSubMaps := make(map[string]map[string]any)
		for k, v := range mergedMap {
			srcID := report.OutputSources[k]
			if srcID == "" {
				consumerSub[k] = v
			} else {
				ns := strings.TrimPrefix(srcID, "ore:")
				if oreSubMaps[ns] == nil {
					oreSubMaps[ns] = make(map[string]any)
				}
				oreSubMaps[ns][k] = v
			}
		}

		primary, rerr := ResolveFiles(consumerSub, moldFS, opts...)
		if rerr != nil {
			return nil, rerr
		}
		for _, r := range primary {
			if r.SrcFS == nil {
				r.SrcFS = moldFS
			}
			resolved = append(resolved, r)
		}

		for ns, subMap := range oreSubMaps {
			ore := oreByNS[ns]
			oreResolved, rerr := ResolveFiles(subMap, ore.FS, opts...)
			if rerr != nil {
				return nil, fmt.Errorf("resolving ore %q output: %w", ns, rerr)
			}
			for _, r := range oreResolved {
				r.SrcFS = ore.FS
				r.Origin = ns
				resolved = append(resolved, r)
			}
		}
	} else {
		// Consumer has no explicit output: use mold auto-discovery.
		primary, rerr := ResolveFiles(moldOutput, moldFS, opts...)
		if rerr != nil {
			return nil, rerr
		}
		for _, r := range primary {
			if r.SrcFS == nil {
				r.SrcFS = moldFS
			}
			resolved = append(resolved, r)
		}

		// Ore overlays still need ore-ore source-path conflict detection.
		if len(oreOverlays) > 0 {
			mergedOreMap, report, merr := MergeFluxOutput(nil, oreOverlays)
			if merr != nil {
				return nil, merr
			}
			oreSubMaps := make(map[string]map[string]any)
			for k, srcID := range report.OutputSources {
				ns := strings.TrimPrefix(srcID, "ore:")
				if oreSubMaps[ns] == nil {
					oreSubMaps[ns] = make(map[string]any)
				}
				oreSubMaps[ns][k] = mergedOreMap[k]
			}
			for ns, subMap := range oreSubMaps {
				ore := oreByNS[ns]
				oreResolved, rerr := ResolveFiles(subMap, ore.FS, opts...)
				if rerr != nil {
					return nil, fmt.Errorf("resolving ore %q output: %w", ns, rerr)
				}
				for _, r := range oreResolved {
					r.SrcFS = ore.FS
					r.Origin = ns
					resolved = append(resolved, r)
				}
			}
		}
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
		info, ferr := fs.Stat(src.FS, oreRelPath)
		if ferr != nil {
			return nil, fmt.Errorf("output entry for %q references %q which does not exist in ore %q: %w", fe.dest, oreRelPath, ns, ferr)
		}
		if info.IsDir() {
			return nil, fmt.Errorf("output entry for %q references %q in ore %q, but that is a directory — `from:` must point to a single file", fe.dest, oreRelPath, ns)
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

	sort.Slice(resolved, func(i, j int) bool {
		if resolved[i].DestPath != resolved[j].DestPath {
			return resolved[i].DestPath < resolved[j].DestPath
		}
		return resolved[i].SrcPath < resolved[j].SrcPath
	})

	// Deduplicate by DestPath: consumer-origin (Origin == "") beats ore-origin;
	// two ore-origin entries mapping to the same DestPath is a conflict error.
	resolved, err = deduplicateByDestPath(resolved)
	if err != nil {
		return nil, err
	}

	return resolved, nil
}

// deduplicateByDestPath removes duplicate DestPath entries from a sorted slice.
// Consumer-origin (Origin == "") beats ore-origin. Multiple ore-origin entries
// with the same DestPath return an error naming the conflicting sources.
func deduplicateByDestPath(resolved []ResolvedFile) ([]ResolvedFile, error) {
	if len(resolved) <= 1 {
		return resolved, nil
	}
	out := make([]ResolvedFile, 0, len(resolved))
	i := 0
	for i < len(resolved) {
		j := i + 1
		for j < len(resolved) && resolved[j].DestPath == resolved[i].DestPath {
			j++
		}
		group := resolved[i:j]
		if len(group) == 1 {
			out = append(out, group[0])
			i = j
			continue
		}
		// Multiple entries with the same DestPath: consumer-origin wins.
		var winner *ResolvedFile
		oreOrigins := make([]string, 0, len(group))
		for idx := range group {
			if group[idx].Origin == "" {
				winner = &group[idx]
				break
			}
			oreOrigins = append(oreOrigins, group[idx].Origin)
		}
		if winner != nil {
			out = append(out, *winner)
		} else {
			sort.Strings(oreOrigins)
			return nil, fmt.Errorf(
				"ore output dest-path conflict: %q mapped by multiple ore sources (%s); override in consumer mold output",
				group[0].DestPath, strings.Join(oreOrigins, ", "),
			)
		}
		i = j
	}
	return out, nil
}

// fromEntry holds a parsed `from: ore/<ns>/<path>` consumer output entry.
// Unlike OutputTarget.Process (*bool, nil-default), fromEntry.process is a
// resolved bool because it flows directly to ResolvedFile without going
// through OutputTarget.ShouldProcess().
type fromEntry struct {
	from     string
	dest     string
	process  bool
	set      map[string]any
	strategy string
}

// parseFromEntryFields extracts a `from: ore/<ns>/<path>` entry from a map.
// Returns the parsed fromEntry and true if `from:` is present and non-empty;
// returns the zero value and false otherwise. `defaultDest` is used when the
// map does not set `dest:` explicitly.
func parseFromEntryFields(m map[string]any, defaultDest string) (fromEntry, bool) {
	from, _ := m["from"].(string)
	if from == "" {
		return fromEntry{}, false
	}
	dest, _ := m["dest"].(string)
	if dest == "" {
		dest = defaultDest
	}
	process := true
	if p, ok := m["process"].(bool); ok {
		process = p
	}
	set, _ := m["set"].(map[string]any)
	strategy, _ := m["strategy"].(string)
	return fromEntry{
		from:     from,
		dest:     dest,
		process:  process,
		set:      set,
		strategy: strategy,
	}, true
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
			if fe, ok := parseFromEntryFields(v, key); ok {
				fromEntries = append(fromEntries, fe)
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
				if fe, ok := parseFromEntryFields(em, key); ok {
					fromEntries = append(fromEntries, fe)
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

	// Always return a map (possibly empty) when the consumer declared an
	// `output:` block. Returning nil here would cause ResolveFiles to fall
	// into identity/auto-discovery mode and walk the entire mold — wrong if
	// the consumer's only output was a set of `from:` selectors that all got
	// extracted.
	return fromEntries, remainder, nil
}

// parseOreFromSelector splits a "ore/<namespace>/<path>" selector into its
// namespace and path parts. Empty namespace or path is an error.
func parseOreFromSelector(s string) (namespace string, relpath string, err error) {
	const prefix = "ore/"
	rest, ok := strings.CutPrefix(s, prefix)
	if !ok {
		return "", "", fmt.Errorf("from selector %q must start with %q (form: ore/<namespace>/<path>)", s, prefix)
	}
	ns, p, ok := strings.Cut(rest, "/")
	if !ok {
		return "", "", fmt.Errorf("from selector %q must include a path after the namespace", s)
	}
	p = path.Clean(p)
	if ns == "" || p == "" || p == "." {
		return "", "", fmt.Errorf("from selector %q has empty namespace or path", s)
	}
	if strings.HasPrefix(p, "..") {
		return "", "", fmt.Errorf("from selector %q contains a path traversal (..)", s)
	}
	return ns, p, nil
}
