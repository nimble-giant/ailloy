package fluxpicker

import (
	"testing"

	"github.com/nimble-giant/ailloy/pkg/mold"
)

func TestCommitEditorValue_String(t *testing.T) {
	fv := mold.FluxVar{Name: "k", Type: "string"}
	m := commitEditorValue(Model{overrides: map[string]any{}}, fv, "hello")
	if got := m.Overrides()["k"]; got != "hello" {
		t.Fatalf("override = %v want hello", got)
	}
}

func TestCommitEditorValue_Int(t *testing.T) {
	fv := mold.FluxVar{Name: "k", Type: "int"}
	m := commitEditorValue(Model{overrides: map[string]any{}}, fv, "42")
	if got := m.Overrides()["k"]; got != 42 {
		t.Fatalf("override = %v want 42 (int)", got)
	}
}

func TestCommitEditorValue_IntInvalid(t *testing.T) {
	fv := mold.FluxVar{Name: "k", Type: "int"}
	m := Model{overrides: map[string]any{}}
	m2 := commitEditorValue(m, fv, "abc")
	if _, ok := m2.Overrides()["k"]; ok {
		t.Fatal("expected invalid int to NOT set the override")
	}
	if m2.err == nil {
		t.Fatal("expected validation error on m2.err")
	}
}

func TestCommitEditorValue_Bool(t *testing.T) {
	fv := mold.FluxVar{Name: "k", Type: "bool"}
	m := commitEditorValue(Model{overrides: map[string]any{}}, fv, "true")
	if got := m.Overrides()["k"]; got != true {
		t.Fatalf("override = %v want true", got)
	}
}

func TestCommitEditorValue_List(t *testing.T) {
	fv := mold.FluxVar{Name: "k", Type: "list"}
	m := commitEditorValue(Model{overrides: map[string]any{}}, fv, "a, b, c")
	got, ok := m.Overrides()["k"].([]string)
	if !ok {
		t.Fatalf("override type = %T want []string", m.Overrides()["k"])
	}
	if len(got) != 3 || got[0] != "a" || got[1] != "b" || got[2] != "c" {
		t.Fatalf("list = %v want [a b c]", got)
	}
}

func TestCommitEditorValue_EmptyClears(t *testing.T) {
	fv := mold.FluxVar{Name: "k", Type: "string"}
	m := Model{overrides: map[string]any{"k": "old"}}
	m2 := commitEditorValue(m, fv, "")
	if _, ok := m2.Overrides()["k"]; ok {
		t.Fatal("expected empty value to clear the override")
	}
}

func TestCommitEditorValue_SelectValidates(t *testing.T) {
	fv := mold.FluxVar{
		Name: "k",
		Type: "select",
		Options: []mold.SelectOption{
			{Value: "a", Label: "A"},
			{Value: "b", Label: "B"},
		},
	}
	m := commitEditorValue(Model{overrides: map[string]any{}}, fv, "a")
	if got := m.Overrides()["k"]; got != "a" {
		t.Fatalf("override = %v want a", got)
	}
	m2 := commitEditorValue(Model{overrides: map[string]any{}}, fv, "z")
	if _, ok := m2.Overrides()["k"]; ok {
		t.Fatal("expected invalid select to NOT set override")
	}
	if m2.err == nil {
		t.Fatal("expected error on invalid select option")
	}
}
