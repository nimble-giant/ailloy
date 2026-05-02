package fluxpicker

import (
	"testing"

	"github.com/nimble-giant/ailloy/internal/tui/foundries/data"
	"github.com/nimble-giant/ailloy/pkg/mold"
)

func TestNew_DefaultsAndOpenForMold(t *testing.T) {
	m := New()
	if m.IsOpen() {
		t.Fatal("expected closed picker on construction")
	}

	schema := []mold.FluxVar{{Name: "k", Type: "string"}}
	m = m.OpenFor("official/demo", data.ScopeProject, schema, map[string]any{"k": "v"})
	if !m.IsOpen() {
		t.Fatal("expected open after OpenFor")
	}
	if got := m.MoldRef(); got != "official/demo" {
		t.Fatalf("MoldRef = %q want official/demo", got)
	}
}

func TestClose_ClearsOpenFlag(t *testing.T) {
	m := New().OpenFor("a", data.ScopeProject, nil, nil)
	m = m.Close()
	if m.IsOpen() {
		t.Fatal("expected closed after Close")
	}
}

func TestSetOverride_StoresValue(t *testing.T) {
	m := New().OpenFor("a", data.ScopeProject, nil, nil)
	m = m.SetOverride("agents.targets", []string{"opencode"})
	if got := m.Overrides()["agents.targets"]; got == nil {
		t.Fatalf("expected override to be set, overrides = %+v", m.Overrides())
	}
}

func TestClearOverride_RemovesValue(t *testing.T) {
	m := New().OpenFor("a", data.ScopeProject, nil, nil)
	m = m.SetOverride("k", "v").ClearOverride("k")
	if _, ok := m.Overrides()["k"]; ok {
		t.Fatal("expected override to be cleared")
	}
}

func TestResetOverrides_EmptiesMap(t *testing.T) {
	m := New().OpenFor("a", data.ScopeProject, nil, nil).
		SetOverride("k1", 1).
		SetOverride("k2", 2).
		ResetOverrides()
	if len(m.Overrides()) != 0 {
		t.Fatalf("expected empty overrides, got %+v", m.Overrides())
	}
}

func TestBadgeStateFor(t *testing.T) {
	schema := []mold.FluxVar{{Name: "k", Default: "d"}}
	defaults := map[string]any{"k": "d"}
	m := New().OpenFor("a", data.ScopeProject, schema, defaults)
	if got := m.BadgeStateFor("k"); got != BadgeDefault {
		t.Fatalf("expected BadgeDefault, got %v", got)
	}
	m = m.SetOverride("k", "x")
	if got := m.BadgeStateFor("k"); got != BadgeSet {
		t.Fatalf("expected BadgeSet, got %v", got)
	}
	if got := m.BadgeStateFor("missing"); got != BadgeUnset {
		t.Fatalf("expected BadgeUnset, got %v", got)
	}
}

func TestBadgeStateFor_DottedDefault(t *testing.T) {
	schema := []mold.FluxVar{{Name: "agents.parallel"}}
	defaults := map[string]any{
		"agents": map[string]any{"parallel": true},
	}
	m := New().OpenFor("a", data.ScopeProject, schema, defaults)
	if got := m.BadgeStateFor("agents.parallel"); got != BadgeDefault {
		t.Fatalf("expected BadgeDefault for nested default, got %v", got)
	}
}
