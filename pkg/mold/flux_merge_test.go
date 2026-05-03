package mold

import (
	"strings"
	"testing"
)

func TestMergeFluxSchema_NoOverlays_ReturnsBase(t *testing.T) {
	base := []FluxVar{
		{Name: "project.organization", Type: "string"},
		{Name: "project.number", Type: "int"},
	}
	out, _, err := MergeFluxSchema(base, nil)
	if err != nil {
		t.Fatalf("MergeFluxSchema: %v", err)
	}
	if len(out) != 2 || out[0].Name != "project.organization" || out[1].Name != "project.number" {
		t.Errorf("unexpected merge: %+v", out)
	}
}

func TestMergeFluxSchema_OverlayAppendsNetNew(t *testing.T) {
	base := []FluxVar{{Name: "project.organization", Type: "string"}}
	overlays := []OverlaySchema{
		{Source: "ore:status", Entries: []FluxVar{
			{Name: "ore.status.enabled", Type: "bool", Default: "false"},
			{Name: "ore.status.field_id", Type: "string"},
		}},
	}
	out, report, err := MergeFluxSchema(base, overlays)
	if err != nil {
		t.Fatalf("MergeFluxSchema: %v", err)
	}
	if len(out) != 3 {
		t.Fatalf("len = %d, want 3: %+v", len(out), out)
	}
	if out[0].Name != "project.organization" || out[1].Name != "ore.status.enabled" || out[2].Name != "ore.status.field_id" {
		t.Errorf("unexpected order: %+v", out)
	}
	if len(report.Shadowed) != 0 {
		t.Errorf("Shadowed should be empty, got %+v", report.Shadowed)
	}
}

func TestMergeFluxSchema_BaseShadowsOverlay(t *testing.T) {
	base := []FluxVar{
		{Name: "ore.status.field_id", Type: "string", Description: "mold-local override"},
	}
	overlays := []OverlaySchema{
		{Source: "ore:status", Entries: []FluxVar{
			{Name: "ore.status.field_id", Type: "string", Description: "from package"},
			{Name: "ore.status.enabled", Type: "bool"},
		}},
	}
	out, report, err := MergeFluxSchema(base, overlays)
	if err != nil {
		t.Fatalf("MergeFluxSchema: %v", err)
	}
	// base entry wins; overlay's enabled is appended.
	if len(out) != 2 {
		t.Fatalf("len = %d, want 2: %+v", len(out), out)
	}
	if out[0].Description != "mold-local override" {
		t.Errorf("base should win on field_id: %+v", out[0])
	}
	if len(report.Shadowed) != 1 || report.Shadowed[0].Name != "ore.status.field_id" {
		t.Errorf("Shadowed should record field_id: %+v", report.Shadowed)
	}
}

func TestMergeFluxSchema_OverlayConflict_Errors(t *testing.T) {
	base := []FluxVar{}
	overlays := []OverlaySchema{
		{Source: "ore:status@a", Entries: []FluxVar{{Name: "ore.status.enabled", Type: "bool"}}},
		{Source: "ore:status@b", Entries: []FluxVar{{Name: "ore.status.enabled", Type: "bool"}}},
	}
	_, _, err := MergeFluxSchema(base, overlays)
	if err == nil {
		t.Fatal("expected conflict error, got nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, "ore.status.enabled") || !strings.Contains(msg, "ore:status@a") || !strings.Contains(msg, "ore:status@b") {
		t.Errorf("error should name key + both sources: %s", msg)
	}
}

func TestValidateOrphanDefaults_EmptyInputs_NoOrphans(t *testing.T) {
	if got := ValidateOrphanDefaults(nil, nil); len(got) != 0 {
		t.Errorf("got %v", got)
	}
}

func TestValidateOrphanDefaults_AllKnown_NoOrphans(t *testing.T) {
	schema := []FluxVar{{Name: "project.organization"}, {Name: "ore.status.enabled"}}
	defaults := map[string]any{
		"project": map[string]any{"organization": "nimble-giant"},
		"ore":     map[string]any{"status": map[string]any{"enabled": false}},
	}
	if got := ValidateOrphanDefaults(schema, defaults); len(got) != 0 {
		t.Errorf("got %v", got)
	}
}

func TestValidateOrphanDefaults_ExtraLeaf_Reported(t *testing.T) {
	schema := []FluxVar{{Name: "project.organization"}}
	defaults := map[string]any{
		"project": map[string]any{"organization": "ng", "number": 5},
		"orphan":  "value",
	}
	got := ValidateOrphanDefaults(schema, defaults)
	want := []string{"orphan", "project.number"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("got %v; want %v", got, want)
	}
}

func TestValidateOrphanDefaults_NilValueIsLeaf(t *testing.T) {
	schema := []FluxVar{}
	defaults := map[string]any{"some_key": nil}
	got := ValidateOrphanDefaults(schema, defaults)
	if len(got) != 1 || got[0] != "some_key" {
		t.Errorf("nil value should be reported as leaf orphan; got %v", got)
	}
}

func TestValidateOrphanDefaults_EmptyMapEmitsNothing(t *testing.T) {
	schema := []FluxVar{}
	defaults := map[string]any{"empty_branch": map[string]any{}}
	got := ValidateOrphanDefaults(schema, defaults)
	if len(got) != 0 {
		t.Errorf("empty map should emit no leaves; got %v", got)
	}
}
