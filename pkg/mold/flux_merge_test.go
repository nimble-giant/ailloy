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
