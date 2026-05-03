package merge

import (
	"strings"
	"testing"
)

func TestLoadYAML_PreservesOrder(t *testing.T) {
	in := []byte("b: 2\na: 1\nnested:\n  y: Y\n  x: X\n")
	got, err := loadYAML(in)
	if err != nil {
		t.Fatalf("loadYAML: %v", err)
	}
	wantKeys := []string{"b", "a", "nested"}
	if len(got.keys) != len(wantKeys) {
		t.Fatalf("keys: want %v, got %v", wantKeys, got.keys)
	}
	for i, k := range wantKeys {
		if got.keys[i] != k {
			t.Errorf("key[%d]: want %q, got %q", i, k, got.keys[i])
		}
	}
}

func TestLoadYAML_Sequence(t *testing.T) {
	in := []byte("items:\n  - a\n  - b\n  - c\n")
	got, err := loadYAML(in)
	if err != nil {
		t.Fatal(err)
	}
	items := got.fields["items"]
	if items == nil || items.kind != kindSeq {
		t.Fatalf("items: want kindSeq, got %+v", items)
	}
	if len(items.seq) != 3 {
		t.Fatalf("len: want 3, got %d", len(items.seq))
	}
}

func TestDumpYAML_RoundTrip(t *testing.T) {
	in := []byte("mcp:\n  outline:\n    url: https://outline\n")
	n, err := loadYAML(in)
	if err != nil {
		t.Fatal(err)
	}
	out, err := dumpYAML(n)
	if err != nil {
		t.Fatal(err)
	}
	got := string(out)
	if !strings.Contains(got, "mcp:") || !strings.Contains(got, "outline:") {
		t.Errorf("round-trip lost structure: %s", got)
	}
}

func TestYAMLMergeRoundTrip(t *testing.T) {
	base := []byte("mcp:\n  outline:\n    url: https://outline\n")
	overlay := []byte("mcp:\n  replicated-docs:\n    url: https://docs\n")
	bn, err := loadYAML(base)
	if err != nil {
		t.Fatal(err)
	}
	on, err := loadYAML(overlay)
	if err != nil {
		t.Fatal(err)
	}
	merged := mergeNodes(bn, on)
	out, err := dumpYAML(merged)
	if err != nil {
		t.Fatal(err)
	}
	got := string(out)
	if !strings.Contains(got, "outline") || !strings.Contains(got, "replicated-docs") {
		t.Errorf("merged YAML missing entries: %s", got)
	}
	// Order: outline before replicated-docs.
	if strings.Index(got, "outline") > strings.Index(got, "replicated-docs") {
		t.Errorf("expected outline before replicated-docs in:\n%s", got)
	}
}
