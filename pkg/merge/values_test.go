package merge

import (
	"reflect"
	"testing"
)

func TestMergeYAMLValues_OverlayWinsOnScalar(t *testing.T) {
	base := map[string]any{"a": "base", "b": "base"}
	overlay := map[string]any{"a": "overlay"}
	out := MergeYAMLValues(base, overlay)
	want := map[string]any{"a": "overlay", "b": "base"}
	if !reflect.DeepEqual(out, want) {
		t.Errorf("got %v; want %v", out, want)
	}
}

func TestMergeYAMLValues_NestedMapsMerge(t *testing.T) {
	base := map[string]any{"ore": map[string]any{"status": map[string]any{"enabled": false, "field_id": ""}}}
	overlay := map[string]any{"ore": map[string]any{"status": map[string]any{"field_id": "PVTSSF_x"}}}
	out := MergeYAMLValues(base, overlay)
	got := out["ore"].(map[string]any)["status"].(map[string]any)
	if got["enabled"] != false {
		t.Errorf("enabled should remain false: %v", got)
	}
	if got["field_id"] != "PVTSSF_x" {
		t.Errorf("field_id should be overlaid: %v", got)
	}
}

func TestMergeYAMLValues_MultipleOverlaysLeftToRight(t *testing.T) {
	base := map[string]any{"a": 1}
	o1 := map[string]any{"a": 2, "b": 2}
	o2 := map[string]any{"a": 3}
	out := MergeYAMLValues(base, o1, o2)
	if out["a"] != 3 || out["b"] != 2 {
		t.Errorf("got %v", out)
	}
}

func TestMergeYAMLValues_NonInterfaceSliceDoesNotPanic(t *testing.T) {
	// Programmatically-constructed maps may use typed slices ([]string) rather
	// than YAML-unmarshalled []any. valueEqual must not panic comparing them.
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panic: %v", r)
		}
	}()
	base := map[string]any{"items": []string{"a", "b"}}
	overlay := map[string]any{"items": []string{"a", "b"}}
	out := MergeYAMLValues(base, overlay)
	// kind mismatch ([]string vs []string both match overlayIsSeq=false),
	// so overlay wins via fall-through. Just verify no panic and we got something.
	if out["items"] == nil {
		t.Errorf("missing items: %v", out)
	}
}

func TestMergeYAMLValues_SequenceConcatWithDedup(t *testing.T) {
	base := map[string]any{"items": []any{"a", "b"}}
	overlay := map[string]any{"items": []any{"b", "c"}}
	out := MergeYAMLValues(base, overlay)
	got, ok := out["items"].([]any)
	if !ok {
		t.Fatalf("items is not []any: %T", out["items"])
	}
	want := []any{"a", "b", "c"}
	if len(got) != len(want) {
		t.Fatalf("len=%d, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("got[%d]=%v; want %v", i, got[i], want[i])
		}
	}
}
