package merge

import (
	"encoding/json"
	"testing"
)

func mkScalar(v any) *node { return &node{kind: kindScalar, scalar: v} }

func mkMap(entries ...any) *node {
	if len(entries)%2 != 0 {
		panic("mkMap: odd number of entries")
	}
	n := &node{kind: kindMap, fields: map[string]*node{}}
	for i := 0; i < len(entries); i += 2 {
		k := entries[i].(string)
		v := entries[i+1].(*node)
		n.keys = append(n.keys, k)
		n.fields[k] = v
	}
	return n
}

func mkSeq(items ...*node) *node {
	return &node{kind: kindSeq, seq: items}
}

func TestMergeNodes_ScalarOverwrite(t *testing.T) {
	got := mergeNodes(mkScalar("old"), mkScalar("new"))
	if got.kind != kindScalar || got.scalar != "new" {
		t.Fatalf("scalar overwrite: got %+v", got)
	}
}

func TestMergeNodes_TypeMismatchOverlayWins(t *testing.T) {
	got := mergeNodes(mkScalar("x"), mkMap("a", mkScalar(1)))
	if got.kind != kindMap {
		t.Fatalf("type mismatch: overlay map should win, got kind=%v", got.kind)
	}
}

func TestMergeNodes_MapMergePreservesOrder(t *testing.T) {
	base := mkMap(
		"a", mkScalar(1),
		"b", mkScalar(2),
		"c", mkScalar(3),
	)
	overlay := mkMap(
		"b", mkScalar(20), // override existing
		"d", mkScalar(4), // append new
	)
	got := mergeNodes(base, overlay)

	wantKeys := []string{"a", "b", "c", "d"}
	if len(got.keys) != len(wantKeys) {
		t.Fatalf("keys: want %v, got %v", wantKeys, got.keys)
	}
	for i, k := range wantKeys {
		if got.keys[i] != k {
			t.Fatalf("key[%d]: want %q, got %q", i, k, got.keys[i])
		}
	}
	if got.fields["b"].scalar != 20 {
		t.Fatalf("b override: want 20, got %v", got.fields["b"].scalar)
	}
	if got.fields["d"].scalar != 4 {
		t.Fatalf("d new: want 4, got %v", got.fields["d"].scalar)
	}
}

func TestMergeNodes_NestedMapMerge(t *testing.T) {
	base := mkMap(
		"mcp", mkMap(
			"outline", mkMap("url", mkScalar("https://outline")),
		),
	)
	overlay := mkMap(
		"mcp", mkMap(
			"replicated-docs", mkMap("url", mkScalar("https://docs")),
		),
	)
	got := mergeNodes(base, overlay)

	mcp := got.fields["mcp"]
	if mcp == nil || mcp.kind != kindMap {
		t.Fatalf("mcp not a map: %+v", mcp)
	}
	if mcp.fields["outline"] == nil {
		t.Error("outline missing after merge")
	}
	if mcp.fields["replicated-docs"] == nil {
		t.Error("replicated-docs missing after merge")
	}
}

func TestMergeNodes_SeqConcatDedupe(t *testing.T) {
	base := mkSeq(mkScalar("a"), mkScalar("b"))
	overlay := mkSeq(mkScalar("b"), mkScalar("c"))
	got := mergeNodes(base, overlay)

	if got.kind != kindSeq {
		t.Fatalf("expected seq, got %v", got.kind)
	}
	if len(got.seq) != 3 {
		t.Fatalf("expected 3 entries (a,b,c), got %d: %+v", len(got.seq), got.seq)
	}
	wantOrder := []string{"a", "b", "c"}
	for i, w := range wantOrder {
		if got.seq[i].scalar != w {
			t.Errorf("seq[%d]: want %q, got %v", i, w, got.seq[i].scalar)
		}
	}
}

func TestMergeNodes_SeqDedupeDeepEqual(t *testing.T) {
	base := mkSeq(mkMap("name", mkScalar("outline")))
	overlay := mkSeq(
		mkMap("name", mkScalar("outline")),         // dup
		mkMap("name", mkScalar("replicated-docs")), // new
	)
	got := mergeNodes(base, overlay)

	if len(got.seq) != 2 {
		t.Fatalf("expected 2 entries (outline + replicated-docs), got %d", len(got.seq))
	}
}

func TestNodeEqual(t *testing.T) {
	if !nodeEqual(mkScalar(1), mkScalar(1)) {
		t.Error("scalar 1 should equal scalar 1")
	}
	if nodeEqual(mkScalar(1), mkScalar(2)) {
		t.Error("scalar 1 should NOT equal scalar 2")
	}

	a := mkMap("x", mkScalar(1), "y", mkScalar(2))
	b := mkMap("y", mkScalar(2), "x", mkScalar(1)) // different key order
	if !nodeEqual(a, b) {
		t.Error("maps with same key/values but different insertion order should be equal")
	}

	if nodeEqual(mkSeq(mkScalar(1), mkScalar(2)), mkSeq(mkScalar(2), mkScalar(1))) {
		t.Error("seq equality is order-sensitive")
	}
}

func TestNodeEqual_JSONNumberStrings(t *testing.T) {
	// json.Number compares by string identity inside a scalar.
	a := mkScalar(json.Number("5"))
	b := mkScalar(json.Number("5"))
	if !nodeEqual(a, b) {
		t.Error("identical json.Number scalars should be equal")
	}
	c := mkScalar(json.Number("5.0"))
	if nodeEqual(a, c) {
		t.Error("json.Number(\"5\") and json.Number(\"5.0\") differ as strings")
	}
}
