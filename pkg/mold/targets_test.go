package mold

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
	"testing/fstest"
)

// testRegistryFlux returns flux carrying a small harness registry under
// ore.agent_targets so tests don't depend on the embedded defaults.
func testRegistryFlux(targets ...any) map[string]any {
	return map[string]any{
		"ore": map[string]any{
			"agent_targets": map[string]any{
				"enabled": true,
				"targets": targets,
				"harnesses": map[string]any{
					"claude": map[string]any{
						"instructions": "CLAUDE.md",
						"agents":       ".claude/agents",
						"skills":       ".claude/skills",
					},
					"codex": map[string]any{
						"instructions": "AGENTS.md",
						"agents": map[string]any{
							"dest":   ".codex/agents",
							"ext":    ".toml",
							"format": "codex-toml",
						},
						"skills": ".agents/skills",
					},
					"opencode": map[string]any{
						"instructions": "AGENTS.md",
						"agents":       map[string]any{"dest": ".opencode/agents", "format": "opencode"},
					},
				},
			},
		},
	}
}

func TestExpandTargetOutputs_FansOutPerHarness(t *testing.T) {
	// Given a mold that asks for its agents/ to go to every target's agents surface
	output := map[string]any{"agents": map[string]any{"targets": "agents"}}
	flux := testRegistryFlux("claude", "codex")

	// When the output is expanded
	got, err := ExpandTargetOutputs(output, flux)
	if err != nil {
		t.Fatal(err)
	}

	// Then one destination per harness is produced, each with its own context
	want := []any{
		map[string]any{
			"dest": ".claude/agents",
			"set": map[string]any{
				"agent.current_target":  "claude",
				"agent.current_targets": []any{"claude"},
				"agent.format":          "markdown",
				"agent.targets":         []any{"claude", "codex"},
			},
		},
		map[string]any{
			"dest": ".codex/agents",
			"ext":  ".toml",
			"set": map[string]any{
				"agent.current_target":  "codex",
				"agent.current_targets": []any{"codex"},
				"agent.format":          "codex-toml",
				"agent.targets":         []any{"claude", "codex"},
			},
		},
	}
	if !reflect.DeepEqual(got.(map[string]any)["agents"], want) {
		t.Errorf("agents expanded to\n%#v\nwant\n%#v", got.(map[string]any)["agents"], want)
	}
}

func TestExpandTargetOutputs_SharedDestCollapses(t *testing.T) {
	// Given two harnesses that both read AGENTS.md
	output := map[string]any{"INSTRUCTIONS.md": map[string]any{"targets": "instructions"}}
	flux := testRegistryFlux("codex", "opencode")

	got, err := ExpandTargetOutputs(output, flux)
	if err != nil {
		t.Fatal(err)
	}

	// Then AGENTS.md is written once, listing both harnesses
	entries := got.(map[string]any)["INSTRUCTIONS.md"].([]any)
	if len(entries) != 1 {
		t.Fatalf("want 1 entry, got %d: %#v", len(entries), entries)
	}
	set := entries[0].(map[string]any)["set"].(map[string]any)
	if set["agent.current_target"] != "codex" {
		t.Errorf("current_target = %v, want codex (first listed)", set["agent.current_target"])
	}
	if !reflect.DeepEqual(set["agent.current_targets"], []any{"codex", "opencode"}) {
		t.Errorf("current_targets = %#v", set["agent.current_targets"])
	}
}

func TestExpandTargetOutputs_NoHarnessSupportsKindSkips(t *testing.T) {
	// Given only opencode is targeted and it has no skills surface
	output := map[string]any{"skills": map[string]any{"targets": "skills"}}
	got, err := ExpandTargetOutputs(output, testRegistryFlux("opencode"))
	if err != nil {
		t.Fatal(err)
	}
	// Then the source is explicitly skipped rather than dropped (which would
	// let root-file auto-discovery install it at its identity path)
	if !reflect.DeepEqual(got.(map[string]any)["skills"], map[string]any{"skip": true}) {
		t.Errorf("got %#v, want skip", got.(map[string]any)["skills"])
	}
}

func TestExpandTargetOutputs_UnknownTargetErrors(t *testing.T) {
	output := map[string]any{"agents": map[string]any{"targets": "agents"}}
	_, err := ExpandTargetOutputs(output, testRegistryFlux("nope"))
	if err == nil || !strings.Contains(err.Error(), `unknown agent target "nope"`) {
		t.Fatalf("want unknown target error, got %v", err)
	}
}

func TestExpandTargetOutputs_AgentTargetsOverridesOre(t *testing.T) {
	// Given the established agent.targets convention is set by the user
	flux := testRegistryFlux("claude")
	flux["agent"] = map[string]any{"targets": []any{"codex"}}
	output := map[string]any{"agents": map[string]any{"targets": "agents"}}

	got, err := ExpandTargetOutputs(output, flux)
	if err != nil {
		t.Fatal(err)
	}
	entries := got.(map[string]any)["agents"].([]any)
	if len(entries) != 1 || entries[0].(map[string]any)["dest"] != ".codex/agents" {
		t.Errorf("want only codex, got %#v", entries)
	}
}

func TestExpandTargetOutputs_EntryOverridesMerge(t *testing.T) {
	// Given an entry that adds its own strategy and set
	output := map[string]any{"rules.md": map[string]any{
		"targets":  "instructions",
		"strategy": "append",
		"set":      map[string]any{"section": "rules"},
	}}
	got, err := ExpandTargetOutputs(output, testRegistryFlux("claude"))
	if err != nil {
		t.Fatal(err)
	}
	e := got.(map[string]any)["rules.md"].([]any)[0].(map[string]any)
	if e["strategy"] != "append" {
		t.Errorf("strategy = %v", e["strategy"])
	}
	if e["set"].(map[string]any)["section"] != "rules" {
		t.Errorf("entry set not merged: %#v", e["set"])
	}
}

func TestExpandTargetOutputs_DisabledOreErrors(t *testing.T) {
	flux := testRegistryFlux("claude")
	flux["ore"].(map[string]any)["agent_targets"].(map[string]any)["enabled"] = false
	output := map[string]any{"agents": map[string]any{"targets": "agents"}}
	if _, err := ExpandTargetOutputs(output, flux); err == nil {
		t.Fatal("want error when agent_targets ore is disabled")
	}
}

func TestExpandTargetOutputs_PassesThroughOtherForms(t *testing.T) {
	output := map[string]any{"commands": ".claude/commands"}
	got, err := ExpandTargetOutputs(output, testRegistryFlux("claude"))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, output) {
		t.Errorf("got %#v", got)
	}
	if got, _ := ExpandTargetOutputs(".claude", nil); got != ".claude" {
		t.Errorf("string form changed: %#v", got)
	}
}

func TestExpandTargetOutputs_NilFluxUsesEveryBuiltinHarness(t *testing.T) {
	// Given no flux (validation/smelt paths), every built-in harness with the
	// surface is used so the source is still discovered
	output := map[string]any{"agents": map[string]any{"targets": "agents"}}
	got, err := ExpandTargetOutputs(output, nil)
	if err != nil {
		t.Fatal(err)
	}
	if n := len(got.(map[string]any)["agents"].([]any)); n < 2 {
		t.Errorf("want several built-in destinations, got %d", n)
	}
}

func TestResolveFiles_TargetsExtAndSkip(t *testing.T) {
	// Given a mold with agents/ and a root file that no harness supports
	fsys := fstest.MapFS{
		"agents/reviewer.md": &fstest.MapFile{Data: []byte("x")},
		"skills.md":          &fstest.MapFile{Data: []byte("x")},
	}
	output := map[string]any{
		"agents":    map[string]any{"targets": "agents"},
		"skills.md": map[string]any{"targets": "nothing"},
	}
	expanded, err := ExpandTargetOutputs(output, testRegistryFlux("claude", "codex"))
	if err != nil {
		t.Fatal(err)
	}

	// When resolved
	files, err := ResolveFiles(expanded, fsys)
	if err != nil {
		t.Fatal(err)
	}

	// Then the codex copy takes the .toml extension and skills.md is not installed
	var dests []string
	for _, f := range files {
		dests = append(dests, f.DestPath)
	}
	want := []string{".claude/agents/reviewer.md", ".codex/agents/reviewer.toml"}
	if !reflect.DeepEqual(dests, want) {
		t.Errorf("dests = %v, want %v", dests, want)
	}
}

func TestResolveFiles_ExpandsTargetsWithoutFlux(t *testing.T) {
	// Given a caller that passes raw output (temper, smelt), targets entries
	// still resolve against the built-in registry instead of erroring
	fsys := fstest.MapFS{"agents/a.md": &fstest.MapFile{Data: []byte("x")}}
	files, err := ResolveFiles(map[string]any{"agents": map[string]any{"targets": "agents"}}, fsys)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Fatal("want agents/a.md resolved")
	}
}

func TestBuiltinRegistry_Consistent(t *testing.T) {
	// Given every built-in harness enabled at once
	defaults, err := builtinTargetDefaults()
	if err != nil {
		t.Fatal(err)
	}
	registry := defaults["harnesses"].(map[string]any)
	if len(registry) < 50 {
		t.Errorf("registry has %d harnesses, want at least 50", len(registry))
	}

	for _, kind := range []string{"instructions", "agents", "skills", "commands", "rules", "mcp"} {
		// When each surface is expanded
		got, err := ExpandTargetOutputs(map[string]any{"src": map[string]any{"targets": kind}}, nil)
		if err != nil {
			t.Fatalf("%s: %v", kind, err)
		}

		// Then no destination is claimed with two different extensions or
		// formats, which would make harnesses overwrite each other
		seen := map[string]string{}
		for _, e := range got.(map[string]any)["src"].([]any) {
			m := e.(map[string]any)
			dest := m["dest"].(string)
			sig := fmt.Sprint(m["ext"], m["set"].(map[string]any)["agent.format"])
			if prev, ok := seen[dest]; ok && prev != sig {
				t.Errorf("%s: %s claimed as %s and %s", kind, dest, prev, sig)
			}
			seen[dest] = sig
		}
	}

	for id, h := range registry {
		if name, _ := h.(map[string]any)["name"].(string); name == "" {
			t.Errorf("harness %q has no name", id)
		}
	}
}
