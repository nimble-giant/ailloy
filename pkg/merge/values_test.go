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
