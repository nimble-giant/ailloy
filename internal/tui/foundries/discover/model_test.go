package discover

import (
	"testing"

	"github.com/nimble-giant/ailloy/internal/tui/foundries/data"
)

func TestCurrentMold_NoSelection(t *testing.T) {
	m := Model{}
	if _, _, ok := m.CurrentMold(); ok {
		t.Fatalf("expected ok=false on empty model")
	}
}

func TestCurrentMold_HighlightedEntry(t *testing.T) {
	m := Model{
		filtered: []data.CatalogEntry{
			{Name: "agents", Source: "official/agents"},
		},
		cursor: 0,
	}
	ref, scope, ok := m.CurrentMold()
	if !ok {
		t.Fatalf("expected ok=true with one filtered entry")
	}
	if ref != "official/agents" {
		t.Fatalf("ref = %q want %q", ref, "official/agents")
	}
	if scope != data.ScopeProject {
		t.Fatalf("scope = %v want ScopeProject (default)", scope)
	}
}
