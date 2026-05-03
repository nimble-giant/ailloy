package fluxpicker

import (
	"testing"

	"github.com/nimble-giant/ailloy/pkg/mold"
)

func TestFilterKeys_EmptyQueryReturnsAll(t *testing.T) {
	schema := []mold.FluxVar{
		{Name: "agents.targets"},
		{Name: "agents.parallel"},
		{Name: "runtime.profile"},
	}
	out := filterKeys(schema, "")
	if len(out) != 3 {
		t.Fatalf("len = %d want 3", len(out))
	}
	if out[0].Name != "agents.parallel" {
		t.Fatalf("expected sorted by name; got first = %q", out[0].Name)
	}
}

func TestFilterKeys_FuzzyRanking(t *testing.T) {
	schema := []mold.FluxVar{
		{Name: "agents.targets"},
		{Name: "agents.parallel"},
		{Name: "runtime.profile"},
	}
	out := filterKeys(schema, "agents.tar")
	if len(out) == 0 {
		t.Fatal("expected at least one match")
	}
	if out[0].Name != "agents.targets" {
		t.Fatalf("top match = %q want agents.targets", out[0].Name)
	}
}

func TestFilterKeys_NoMatches(t *testing.T) {
	schema := []mold.FluxVar{
		{Name: "agents.targets"},
	}
	out := filterKeys(schema, "zzz")
	if len(out) != 0 {
		t.Fatalf("expected zero matches, got %d", len(out))
	}
}

func TestFilterKeys_NilSchema(t *testing.T) {
	if got := filterKeys(nil, ""); len(got) != 0 {
		t.Fatalf("nil schema empty query: len = %d want 0", len(got))
	}
	if got := filterKeys(nil, "x"); len(got) != 0 {
		t.Fatalf("nil schema with query: len = %d want 0", len(got))
	}
}
