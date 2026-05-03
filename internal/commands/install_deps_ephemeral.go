package commands

import (
	"fmt"

	"dario.cat/mergo"
	"github.com/nimble-giant/ailloy/pkg/foundry"
	"github.com/nimble-giant/ailloy/pkg/mold"
)

// EphemeralOreResolver resolves ore deps for a single command run without
// touching disk under .ailloy/. Used by forge and temper to preview a mold's
// merged schema/defaults without performing an install. Held overlays and
// defaults are suitable for passing into MergeFluxSchema and merging into a
// mold's own flux defaults.
type EphemeralOreResolver struct {
	overlays []mold.OverlaySchema
	defaults map[string]any
}

// Overlays returns the prefixed schema overlays produced from declared ore
// deps, one per dep. Order matches manifest.Dependencies (after sort by source
// inside MergeFluxSchema).
func (r *EphemeralOreResolver) Overlays() []mold.OverlaySchema {
	if r == nil {
		return nil
	}
	return r.overlays
}

// Defaults returns the ore-namespace defaults map produced from declared ore
// deps. Shape: {"ore": {"<name>": {<flux.yaml contents>}, ...}}.
func (r *EphemeralOreResolver) Defaults() map[string]any {
	if r == nil {
		return nil
	}
	return r.defaults
}

// ResolveDepsEphemeral walks manifest.Dependencies and resolves each ore
// without copying into .ailloy/. Ingot deps are validated for parseability
// but contribute nothing to the schema/defaults merge. Returns an empty
// resolver for nil/empty deps.
//
// allowLocalDeps mirrors the security rule from installDeclaredDeps: when
// the parent mold itself was resolved from a remote source, local-path deps
// are refused to prevent exfiltration via the forge/temper preview path.
func ResolveDepsEphemeral(manifest *mold.Mold, allowLocalDeps bool) (*EphemeralOreResolver, error) {
	r := &EphemeralOreResolver{defaults: map[string]any{}}
	if manifest == nil || len(manifest.Dependencies) == 0 {
		return r, nil
	}

	for _, d := range manifest.Dependencies {
		kind, kerr := d.Kind()
		if kerr != nil {
			return nil, fmt.Errorf("invalid dependency: %w", kerr)
		}

		ref := d.Source()
		if !allowLocalDeps && !foundry.IsRemoteReference(ref) {
			return nil, fmt.Errorf("dependency %q is a local path, but the mold was resolved from a remote source; refusing", ref)
		}

		// Resolve into the foundry cache (or stat a local dir). Never copies.
		fsys, _, _, _, _, err := resolveDepFS(ref, d.Version, false)
		if err != nil {
			return nil, fmt.Errorf("resolving %s %s: %w", kind, ref, err)
		}

		if kind == "ingot" {
			// Validate ingot manifest parses, but don't add anything to
			// overlays — ingots don't contribute to flux schema/defaults.
			if _, err := mold.LoadIngotFromFS(fsys, "ingot.yaml"); err != nil {
				return nil, fmt.Errorf("invalid ingot manifest at %s: %w", ref, err)
			}
			continue
		}

		// kind == "ore"
		ore, err := mold.LoadOreFromFS(fsys, "ore.yaml")
		if err != nil {
			return nil, fmt.Errorf("invalid ore manifest at %s: %w", ref, err)
		}
		if ore.Kind != "ore" {
			return nil, fmt.Errorf("manifest at %s has kind=%q, expected 'ore'", ref, ore.Kind)
		}

		name := ore.Name
		if d.As != "" {
			name = d.As
		}

		schema, err := mold.LoadFluxSchema(fsys, "flux.schema.yaml")
		if err != nil {
			return nil, fmt.Errorf("loading ore schema at %s: %w", ref, err)
		}
		prefix := "ore." + name + "."
		prefixed := make([]mold.FluxVar, 0, len(schema))
		for _, e := range schema {
			pe := e
			pe.Name = prefix + e.Name
			prefixed = append(prefixed, pe)
		}
		r.overlays = append(r.overlays, mold.OverlaySchema{Source: "ore:" + name, Entries: prefixed})

		oreDefaults, err := mold.LoadFluxFile(fsys, "flux.yaml")
		if err != nil {
			return nil, fmt.Errorf("loading ore defaults at %s: %w", ref, err)
		}
		if oreDefaults == nil {
			oreDefaults = map[string]any{}
		}
		nsMap, _ := r.defaults["ore"].(map[string]any)
		if nsMap == nil {
			nsMap = map[string]any{}
			r.defaults["ore"] = nsMap
		}
		nsMap[name] = oreDefaults
	}

	return r, nil
}

// MergeInto merges the resolver's overlays/defaults into the given base
// schema and defaults from a mold's own files, returning the merged result
// plus an OreLoadReport. Mold-wins precedence is preserved (base wins on
// collision). Safe to call on a nil receiver — returns the base values
// unchanged with an empty report.
func (r *EphemeralOreResolver) MergeInto(baseSchema []mold.FluxVar, baseDefaults map[string]any) ([]mold.FluxVar, map[string]any, mold.OreLoadReport, error) {
	if r == nil {
		return baseSchema, baseDefaults, mold.OreLoadReport{}, nil
	}
	merged, report, err := mold.MergeFluxSchema(baseSchema, r.overlays)
	if err != nil {
		return nil, nil, report, err
	}
	out := map[string]any{}
	// Mold-wins on collision via mergo.WithOverride; ores layered first.
	if err := mergo.Merge(&out, r.defaults); err != nil {
		return nil, nil, report, fmt.Errorf("merging ore defaults: %w", err)
	}
	if err := mergo.Merge(&out, baseDefaults, mergo.WithOverride); err != nil {
		return nil, nil, report, fmt.Errorf("merging mold defaults over ore defaults: %w", err)
	}
	return merged, out, report, nil
}
