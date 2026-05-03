package merge

import (
	"strings"
	"testing"

	"github.com/goccy/go-yaml"
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

func TestLoadYAML_RejectsNonStringKey(t *testing.T) {
	// The goccy/go-yaml library normalizes scalar keys to strings during
	// UseOrderedMap decoding, so we exercise yamlToNode directly with a
	// constructed MapSlice carrying a non-string key. This guards the
	// defensive check that prevents 123 and "123" from silently colliding
	// on the same Go map key if a future decoder change preserves types.
	ms := yaml.MapSlice{
		{Key: 123, Value: "foo"},
		{Key: true, Value: "bar"},
	}
	_, err := yamlToNode(ms)
	if err == nil {
		t.Fatal("expected error for non-string YAML keys, got nil")
	}
	if !strings.Contains(err.Error(), "non-string YAML key") {
		t.Errorf("error message should mention non-string YAML key; got: %v", err)
	}
}

func TestYamlToNode_RejectsUnorderedMap(t *testing.T) {
	// Direct test on the helper, since UseOrderedMap should always produce
	// MapSlice in the production path. We pass a Go map[string]any here to
	// confirm the defensive branch errors instead of silently losing order.
	_, err := yamlToNode(map[string]any{"a": 1, "b": 2})
	if err == nil {
		t.Fatal("expected error for unordered map, got nil")
	}
	if !strings.Contains(err.Error(), "UseOrderedMap") {
		t.Errorf("error message should mention UseOrderedMap; got: %v", err)
	}
}

func TestLoadYAML_DuplicateKeysLastWins(t *testing.T) {
	// YAML keys 0 and 00 both normalize to string "0" via goccy/go-yaml
	// UseOrderedMap. Ensure the loader dedupes n.keys so dumpYAML produces
	// valid output rather than two "0:" lines (invalid YAML).
	in := []byte("0:\n00:\n")
	n, err := loadYAML(in)
	if err != nil {
		t.Fatal(err)
	}
	if len(n.keys) != 1 {
		t.Fatalf("expected single key entry, got %d: %v", len(n.keys), n.keys)
	}
	out, err := dumpYAML(n)
	if err != nil {
		t.Fatalf("dumpYAML: %v", err)
	}
	if _, err := loadYAML(out); err != nil {
		t.Errorf("dump produced invalid YAML with duplicates: %v\nout: %s", err, out)
	}
}

func TestDumpYAML_AmbiguousStringRoundTrips(t *testing.T) {
	// goccy/go-yaml's default marshaler leaves several string scalars bare
	// that should be quoted — document-separator tokens "---"/"...",
	// whitespace-only strings, strings containing newlines — so re-parsing
	// the dump yields a different value (usually nil or ""). dumpYAML
	// detects these and force-quotes them so round-trip is stable.
	for _, s := range []string{"---", "...", "\n", "\t", "\n\n", "a\n", "a\nb\n"} {
		n := &node{kind: kindScalar, scalar: s}
		out, err := dumpYAML(n)
		if err != nil {
			t.Fatalf("dumpYAML(%q): %v", s, err)
		}
		n2, err := loadYAML(out)
		if err != nil {
			t.Fatalf("loadYAML(%q dump): %v", s, err)
		}
		got, ok := n2.scalar.(string)
		if !ok || got != s {
			t.Errorf("round trip lost value: input=%q dump=%q parsed=%#v", s, out, n2.scalar)
		}
	}
}

func TestDumpYAML_MergeKeyRoundTrips(t *testing.T) {
	// goccy/go-yaml emits the YAML merge key "<<" bare against a null value
	// ("<<: null"), which then fails to re-parse. dumpYAML force-quotes such
	// keys via a custom map marshaler so round-trip is stable. We construct
	// the node directly because goccy/go-yaml refuses some "<<:" inputs.
	n := &node{
		kind:   kindMap,
		keys:   []string{"<<"},
		fields: map[string]*node{"<<": {kind: kindScalar, scalar: nil}},
	}
	out, err := dumpYAML(n)
	if err != nil {
		t.Fatalf("dumpYAML: %v", err)
	}
	n2, err := loadYAML(out)
	if err != nil {
		t.Fatalf("re-parse failed: %v\nout: %s", err, out)
	}
	if _, exists := n2.fields["<<"]; !exists {
		t.Errorf("<< key lost: out=%s n2.keys=%v", out, n2.keys)
	}
}
