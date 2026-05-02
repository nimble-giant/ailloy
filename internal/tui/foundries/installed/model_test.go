package installed

import (
	"testing"

	"github.com/nimble-giant/ailloy/internal/tui/foundries/data"
	"github.com/nimble-giant/ailloy/pkg/foundry"
)

func TestCurrentMold_NoSelection(t *testing.T) {
	m := Model{}
	if _, _, ok := m.CurrentMold(); ok {
		t.Fatalf("expected ok=false on empty model")
	}
}

func TestCurrentMold_HighlightedItem_PreservesScope(t *testing.T) {
	m := Model{
		items: []data.InventoryItem{
			{Scope: data.ScopeGlobal, Entry: foundry.InstalledEntry{Source: "official/agents"}},
		},
		cursor: 0,
	}
	ref, scope, ok := m.CurrentMold()
	if !ok {
		t.Fatalf("expected ok=true")
	}
	if ref != "official/agents" {
		t.Fatalf("ref = %q want %q", ref, "official/agents")
	}
	if scope != data.ScopeGlobal {
		t.Fatalf("scope = %v want ScopeGlobal", scope)
	}
}
