package mold

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"

	"dario.cat/mergo"
)

// FetchSchemaFromSource resolves a mold by source string (local path or remote
// ref) and returns its flux schema and default values, without doing a full
// cast. For local paths it reads from the filesystem directly; for remote refs
// it delegates to ResolveSchemaFunc (set by callers that have access to the
// foundry resolver to avoid an import cycle).
//
// When neither flux.schema.yaml nor flux.yaml exist, returns an empty schema
// and empty defaults with no error.
func FetchSchemaFromSource(ctx context.Context, source string) ([]FluxVar, map[string]any, error) {
	if source == "" {
		return nil, nil, errors.New("FetchSchemaFromSource: empty source")
	}
	// Local path?
	if info, err := os.Stat(source); err == nil {
		if !info.IsDir() {
			return nil, nil, fmt.Errorf("FetchSchemaFromSource: %q is not a directory", source)
		}
		return loadSchemaFromFS(os.DirFS(source))
	} else if !errors.Is(err, fs.ErrNotExist) {
		return nil, nil, fmt.Errorf("FetchSchemaFromSource: stat %q: %w", source, err)
	}
	if ResolveSchemaFunc == nil {
		return nil, nil, fmt.Errorf("FetchSchemaFromSource: no remote resolver registered for source %q", source)
	}
	fsys, cleanup, err := ResolveSchemaFunc(ctx, source)
	if cleanup != nil {
		defer cleanup()
	}
	if err != nil {
		return nil, nil, err
	}
	return loadSchemaFromFS(fsys)
}

// ResolveSchemaFunc is set by callers that can resolve a remote mold to an
// fs.FS without creating an import cycle into pkg/mold. The returned cleanup
// function (if non-nil) is called after schema files are read.
var ResolveSchemaFunc func(ctx context.Context, source string) (fs.FS, func(), error)

func loadSchemaFromFS(fsys fs.FS) ([]FluxVar, map[string]any, error) {
	schema, err := LoadFluxSchema(fsys, "flux.schema.yaml")
	if err != nil {
		return nil, nil, fmt.Errorf("load schema: %w", err)
	}
	defaults, err := LoadFluxFile(fsys, "flux.yaml")
	if err != nil {
		return nil, nil, fmt.Errorf("load defaults: %w", err)
	}
	if defaults == nil {
		defaults = map[string]any{}
	}
	return schema, defaults, nil
}

// OreSearchPath describes one ore search location, in priority order.
// FS is the filesystem to read from; Root is the directory inside FS that
// contains <name>/ subdirectories (e.g. "ores" if FS is rooted at .ailloy/).
type OreSearchPath struct {
	Name string // "mold-local" / "project" / "global", for diagnostics
	FS   fs.FS
	Root string // typically "ores"
}

// LoadMoldFluxWithOres is the ore-aware schema/defaults loader. It loads the
// mold's own flux.schema.yaml + flux.yaml, then walks each search path in
// order, treating later paths as lower-priority (only loading ores not yet
// seen). It returns the merged schema, merged defaults, and an OreLoadReport
// covering shadowing and source attribution.
func LoadMoldFluxWithOres(moldFS fs.FS, paths []OreSearchPath) ([]FluxVar, map[string]any, OreLoadReport, error) {
	base, baseDefaults, err := loadSchemaFromFS(moldFS)
	if err != nil {
		return nil, nil, OreLoadReport{}, err
	}

	seen := map[string]struct{}{}
	var allOverlays []OverlaySchema
	allOreDefaults := map[string]any{}
	for _, p := range paths {
		overlays, defs, lerr := LoadOreOverlaysFromFS(p.FS, p.Root, seen)
		if lerr != nil {
			return nil, nil, OreLoadReport{}, fmt.Errorf("search path %s: %w", p.Name, lerr)
		}
		allOverlays = append(allOverlays, overlays...)
		MergeOreDefaults(allOreDefaults, defs)
	}

	mergedSchema, report, err := MergeFluxSchema(base, allOverlays)
	if err != nil {
		return nil, nil, report, err
	}

	// Layer ore defaults under mold defaults: ores first, then mold wins.
	merged := map[string]any{}
	if err := mergo.Merge(&merged, allOreDefaults); err != nil {
		return nil, nil, report, fmt.Errorf("merging ore defaults: %w", err)
	}
	if err := mergo.Merge(&merged, baseDefaults, mergo.WithOverride); err != nil {
		return nil, nil, report, fmt.Errorf("merging mold defaults over ore defaults: %w", err)
	}

	return mergedSchema, merged, report, nil
}

// MergeOreDefaults shallowly merges the "ore" namespace from src into dst.
// Each top-level key in src whose value is a map[string]any is merged into
// dst's same key (one level deep). Other top-level keys are added only if
// not already present in dst — this preserves higher-priority entries when
// later (lower-priority) search paths contribute the same key.
//
// Exported because EphemeralOreResolver in internal/commands needs to call
// it; see Phase 9.
func MergeOreDefaults(dst, src map[string]any) {
	for k, v := range src {
		if _, exists := dst[k]; !exists {
			dst[k] = v
			continue
		}
		dstMap, ok1 := dst[k].(map[string]any)
		srcMap, ok2 := v.(map[string]any)
		if ok1 && ok2 {
			for sk, sv := range srcMap {
				if _, present := dstMap[sk]; !present {
					dstMap[sk] = sv
				}
			}
		}
	}
}
