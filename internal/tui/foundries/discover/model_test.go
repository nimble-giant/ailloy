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

func TestApplySessionOverrides_StoresEncoded(t *testing.T) {
	m := Model{}
	m = m.ApplySessionOverrides("official/x", map[string]any{
		"agents.targets":  []string{"opencode", "claude"},
		"agents.parallel": true,
	})
	got := m.pending["official/x"]
	if len(got) != 2 {
		t.Fatalf("len = %d want 2; got %v", len(got), got)
	}
	// Sorted: "agents.parallel" < "agents.targets".
	if got[0] != "agents.parallel=true" {
		t.Fatalf("got[0] = %q", got[0])
	}
	// Slices are emitted as YAML flow sequences so the --set parser produces
	// a real list, not a single space-separated string.
	if got[1] != "agents.targets=[opencode,claude]" {
		t.Fatalf("got[1] = %q want agents.targets=[opencode,claude]", got[1])
	}
}
