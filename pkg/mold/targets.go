package mold

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	agenttargets "github.com/nimble-giant/ailloy/ores/agent_targets"
)

// TargetsNamespace is the ore namespace that carries the harness registry
// `targets:` output entries expand against.
const TargetsNamespace = "agent_targets"

// defaultTargetFormat is the agent.format value for a harness surface that
// declares no format of its own.
const defaultTargetFormat = "markdown"

var (
	builtinOnce     sync.Once
	builtinDefaults map[string]any
	builtinErr      error
)

// builtinTargetDefaults returns the embedded agent_targets ore defaults.
func builtinTargetDefaults() (map[string]any, error) {
	builtinOnce.Do(func() {
		builtinDefaults, builtinErr = LoadFluxFile(agenttargets.FS, "flux.yaml")
	})
	return builtinDefaults, builtinErr
}

// OutputFor returns flux's output mapping with every `targets:` entry
// expanded against the harness registry. Callers that render should use it
// instead of flux["output"].
func OutputFor(flux map[string]any) (any, error) {
	return ExpandTargetOutputs(flux["output"], flux)
}

// ExpandTargetOutputs rewrites output entries of the form
//
//	agents: {targets: agents}
//
// into one destination per enabled harness that declares that surface in
// the agent_targets registry, each with agent.current_target, agent.format,
// agent.current_targets and agent.targets set for its render pass.
// Harnesses that share a destination are written once. When no enabled
// harness declares the surface the entry becomes {skip: true}.
//
// The registry and target list come from flux (ore.agent_targets.*, with a
// user-set agent.targets taking precedence) and fall back to the built-in
// ore. With nil flux every built-in harness is used, so callers that only
// need source paths (temper, smelt) still see every source.
//
// Non-map output values and entries without `targets` pass through unchanged.
func ExpandTargetOutputs(output any, flux map[string]any) (any, error) {
	m, ok := output.(map[string]any)
	if !ok || !hasTargetsEntry(m) {
		return output, nil
	}

	registry, targets, err := targetRegistry(flux)
	if err != nil {
		return nil, err
	}

	out := make(map[string]any, len(m))
	for src, val := range m {
		entry, ok := val.(map[string]any)
		if !ok || entry["targets"] == nil {
			out[src] = val
			continue
		}
		expanded, err := expandTargetEntry(entry, registry, targets)
		if err != nil {
			return nil, fmt.Errorf("output key %q: %w", src, err)
		}
		out[src] = expanded
	}
	return out, nil
}

func hasTargetsEntry(m map[string]any) bool {
	for _, v := range m {
		if e, ok := v.(map[string]any); ok && e["targets"] != nil {
			return true
		}
	}
	return false
}

// targetRegistry resolves the harness registry and the ordered target list.
func targetRegistry(flux map[string]any) (map[string]any, []string, error) {
	defaults, err := builtinTargetDefaults()
	if err != nil {
		return nil, nil, fmt.Errorf("loading built-in agent_targets ore: %w", err)
	}

	ore, _ := lookupMap(flux, "ore", TargetsNamespace)
	if ore == nil {
		ore = defaults
	}
	if enabled, ok := ore["enabled"].(bool); ok && !enabled {
		return nil, nil, fmt.Errorf("output uses targets: but the %s ore is disabled", TargetsNamespace)
	}

	registry, _ := ore["harnesses"].(map[string]any)
	if registry == nil {
		registry, _ = defaults["harnesses"].(map[string]any)
	}

	if flux == nil {
		all := make([]string, 0, len(registry))
		for id := range registry {
			all = append(all, id)
		}
		sort.Strings(all)
		return registry, all, nil
	}

	var raw any
	if agent, ok := flux["agent"].(map[string]any); ok && isNonEmptyList(agent["targets"]) {
		raw = agent["targets"]
	} else {
		raw = ore["targets"]
	}
	targets, err := stringList(raw)
	if err != nil {
		return nil, nil, fmt.Errorf("agent targets: %w", err)
	}
	return registry, targets, nil
}

// targetGroup is one destination shared by one or more harnesses.
type targetGroup struct {
	dest, ext, format, strategy string
	ids                         []any
}

func expandTargetEntry(entry, registry map[string]any, targets []string) (any, error) {
	kind, ok := entry["targets"].(string)
	if !ok || kind == "" {
		return nil, fmt.Errorf("targets must be a surface name string (e.g. agents, skills)")
	}

	var groups []*targetGroup
	for _, id := range targets {
		h, ok := registry[id].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("unknown agent target %q (known: %s)", id, strings.Join(sortedKeys(registry), ", "))
		}
		surface, err := parseSurface(h[kind])
		if err != nil {
			return nil, fmt.Errorf("agent target %q surface %q: %w", id, kind, err)
		}
		if surface == nil {
			continue
		}
		var g *targetGroup
		for _, existing := range groups {
			if existing.dest == surface.dest && existing.ext == surface.ext &&
				existing.format == surface.format && existing.strategy == surface.strategy {
				g = existing
				break
			}
		}
		if g == nil {
			g = surface
			groups = append(groups, g)
		}
		g.ids = append(g.ids, id)
	}

	if len(groups) == 0 {
		return map[string]any{"skip": true}, nil
	}

	allTargets := make([]any, len(targets))
	for i, t := range targets {
		allTargets[i] = t
	}

	list := make([]any, 0, len(groups))
	for _, g := range groups {
		set := map[string]any{
			"agent.current_target":  g.ids[0],
			"agent.current_targets": g.ids,
			"agent.format":          g.format,
			"agent.targets":         allTargets,
		}
		if es, ok := entry["set"].(map[string]any); ok {
			for k, v := range es {
				set[k] = v
			}
		}
		t := map[string]any{"dest": g.dest, "set": set}
		if g.ext != "" {
			t["ext"] = g.ext
		}
		if g.strategy != "" {
			t["strategy"] = g.strategy
		}
		for _, k := range []string{"strategy", "process"} {
			if v, ok := entry[k]; ok {
				t[k] = v
			}
		}
		list = append(list, t)
	}
	return list, nil
}

// parseSurface reads a harness surface: a destination string, or a map with
// dest and optional ext, format and strategy. nil means the harness has no
// such surface.
func parseSurface(v any) (*targetGroup, error) {
	switch s := v.(type) {
	case nil:
		return nil, nil
	case string:
		return &targetGroup{dest: s, format: defaultTargetFormat}, nil
	case map[string]any:
		g := &targetGroup{format: defaultTargetFormat}
		for key, dst := range map[string]*string{"dest": &g.dest, "ext": &g.ext, "format": &g.format, "strategy": &g.strategy} {
			if raw, ok := s[key]; ok {
				str, ok := raw.(string)
				if !ok {
					return nil, fmt.Errorf("%s must be a string", key)
				}
				*dst = str
			}
		}
		if g.dest == "" {
			return nil, fmt.Errorf("dest is required")
		}
		return g, nil
	default:
		return nil, fmt.Errorf("must be a string or map, got %T", v)
	}
}

func lookupMap(m map[string]any, keys ...string) (map[string]any, bool) {
	cur := m
	for _, k := range keys {
		next, ok := cur[k].(map[string]any)
		if !ok {
			return nil, false
		}
		cur = next
	}
	return cur, true
}

func isNonEmptyList(v any) bool {
	switch l := v.(type) {
	case []any:
		return len(l) > 0
	case []string:
		return len(l) > 0
	}
	return false
}

func stringList(v any) ([]string, error) {
	switch l := v.(type) {
	case nil:
		return nil, nil
	case []string:
		return l, nil
	case []any:
		out := make([]string, 0, len(l))
		for _, e := range l {
			s, ok := e.(string)
			if !ok {
				return nil, fmt.Errorf("target ids must be strings, got %T", e)
			}
			out = append(out, s)
		}
		return out, nil
	default:
		return nil, fmt.Errorf("must be a list, got %T", v)
	}
}

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
