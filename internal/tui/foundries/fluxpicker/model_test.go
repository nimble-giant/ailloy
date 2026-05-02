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
