package fluxpicker

import (
	"testing"

	"github.com/nimble-giant/ailloy/internal/tui/foundries/data"
	"github.com/nimble-giant/ailloy/pkg/mold"
)

func TestOpenForFoundryStoresState(t *testing.T) {
	per := map[string][]mold.FluxVar{
		"alpha": {{Name: "agents.targets", Type: "list"}},
		"beta":  {{Name: "agents.targets", Type: "list"}},
	}
	refs := map[string]string{
		"alpha": "github.com/example/alpha",
		"beta":  "github.com/example/beta",
	}
	m := New().OpenForFoundry("nimble-mold", data.ScopeProject, per, refs)
	if !m.IsOpen() {
		t.Fatalf("picker should be open")
	}
	if got := m.FoundryName(); got != "nimble-mold" {
		t.Errorf("FoundryName = %q, want nimble-mold", got)
	}
	if !m.IsFoundryMode() {
		t.Errorf("IsFoundryMode = false, want true")
	}
	if got := m.PerMoldSchemas(); len(got) != 2 {
		t.Errorf("PerMoldSchemas len = %d, want 2", len(got))
	}
	if got := m.PerMoldSourceRefs(); len(got) != 2 {
		t.Errorf("PerMoldSourceRefs len = %d, want 2", len(got))
	}
	if got := m.SchemaConflicts(); len(got) != 0 {
		t.Errorf("expected no conflicts, got %v", got)
	}
}

func TestOpenForFoundryDetectsConflicts(t *testing.T) {
	per := map[string][]mold.FluxVar{
		"alpha": {{Name: "theme", Type: "string", Default: "dark"}},
		"beta":  {{Name: "theme", Type: "string", Default: "light"}},
	}
	m := New().OpenForFoundry("brand", data.ScopeProject, per, nil)
	conflicts := m.SchemaConflicts()
	if got := conflicts["theme"]; len(got) != 2 {
		t.Errorf("conflicts[theme] = %v, want both molds", got)
	}
}
