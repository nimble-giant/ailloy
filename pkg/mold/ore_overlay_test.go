package mold

import "testing"

func TestOreReservedKeys(t *testing.T) {
	if OreOutputKey != "output" {
		t.Errorf("OreOutputKey = %q, want %q", OreOutputKey, "output")
	}
	if OreBlanksDir != "blanks" {
		t.Errorf("OreBlanksDir = %q, want %q", OreBlanksDir, "blanks")
	}
}

func TestMergeFluxOutput_BaseWinsOnCollision(t *testing.T) {
	base := map[string]any{
		"AGENTS.md": "AGENTS.md",
	}
	overlays := []OreOutputOverlay{{
		Source: "ore:agent_targets",
		Entries: map[string]any{
			"AGENTS.md":     "should_be_shadowed",
			"blanks/agents": []any{map[string]any{"dest": ".claude/agents"}},
		},
	}}

	merged, report, err := MergeFluxOutput(base, overlays)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if merged["AGENTS.md"] != "AGENTS.md" {
		t.Errorf("base lost: got %v", merged["AGENTS.md"])
	}
	if _, ok := merged["blanks/agents"]; !ok {
		t.Errorf("net-new overlay entry missing")
	}
	if len(report.ShadowedOutput) != 1 || report.ShadowedOutput[0].Key != "AGENTS.md" {
		t.Errorf("expected one shadowed entry for AGENTS.md, got %+v", report.ShadowedOutput)
	}
	if report.OutputSources["blanks/agents"] != "ore:agent_targets" {
		t.Errorf("OutputSources missing entry: %+v", report.OutputSources)
	}
}

func TestMergeFluxOutput_OverlayOverlayConflictErrors(t *testing.T) {
	base := map[string]any{}
	overlays := []OreOutputOverlay{
		{Source: "ore:a", Entries: map[string]any{"foo": "a"}},
		{Source: "ore:b", Entries: map[string]any{"foo": "b"}},
	}
	_, _, err := MergeFluxOutput(base, overlays)
	if err == nil {
		t.Fatal("expected conflict error, got nil")
	}
}

func TestMergeFluxOutput_NilBase(t *testing.T) {
	overlays := []OreOutputOverlay{{
		Source:  "ore:agent_targets",
		Entries: map[string]any{"blanks/agents": ".claude/agents"},
	}}
	merged, _, err := MergeFluxOutput(nil, overlays)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if merged["blanks/agents"] != ".claude/agents" {
		t.Errorf("nil base path lost overlay entry: %+v", merged)
	}
}

func TestExtractOreOutput_RemovesKeyAndReturnsValue(t *testing.T) {
	defaults := map[string]any{
		"enabled": false,
		"output": map[string]any{
			"blanks/AGENTS.md": "AGENTS.md",
		},
	}
	got := ExtractOreOutput(defaults)
	if _, stillThere := defaults["output"]; stillThere {
		t.Errorf("ExtractOreOutput should remove the output key from input")
	}
	if got["blanks/AGENTS.md"] != "AGENTS.md" {
		t.Errorf("ExtractOreOutput returned wrong value: %+v", got)
	}
}

func TestExtractOreOutput_NoOutputKeyReturnsNil(t *testing.T) {
	defaults := map[string]any{"enabled": false}
	if got := ExtractOreOutput(defaults); got != nil {
		t.Errorf("expected nil, got %+v", got)
	}
}
