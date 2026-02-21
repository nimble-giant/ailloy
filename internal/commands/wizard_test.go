package commands

import (
	"testing"
)

func TestParseCustomVars_ValidInput(t *testing.T) {
	input := "key1=value1\nkey2=value2\nkey3=value with spaces"
	got := parseCustomVars(input)

	if len(got) != 3 {
		t.Fatalf("expected 3 vars, got %d", len(got))
	}
	if got["key1"] != "value1" {
		t.Errorf("key1: expected value1, got %s", got["key1"])
	}
	if got["key2"] != "value2" {
		t.Errorf("key2: expected value2, got %s", got["key2"])
	}
	if got["key3"] != "value with spaces" {
		t.Errorf("key3: expected 'value with spaces', got %s", got["key3"])
	}
}

func TestParseCustomVars_EmptyInput(t *testing.T) {
	got := parseCustomVars("")
	if len(got) != 0 {
		t.Errorf("expected 0 vars, got %d", len(got))
	}
}

func TestParseCustomVars_SkipsBlankLines(t *testing.T) {
	input := "key1=val1\n\n\nkey2=val2\n"
	got := parseCustomVars(input)
	if len(got) != 2 {
		t.Fatalf("expected 2 vars, got %d", len(got))
	}
}

func TestParseCustomVars_SkipsInvalidLines(t *testing.T) {
	input := "key1=val1\ninvalidline\nkey2=val2"
	got := parseCustomVars(input)
	if len(got) != 2 {
		t.Fatalf("expected 2 vars (invalid skipped), got %d", len(got))
	}
}

func TestParseCustomVars_TrimSpaces(t *testing.T) {
	input := "  key1  =  value1  "
	got := parseCustomVars(input)
	if got["key1"] != "value1" {
		t.Errorf("expected trimmed key/value, got key=%q val=%q", "key1", got["key1"])
	}
}

func TestParseCustomVars_ValueWithEquals(t *testing.T) {
	input := "key1=val=ue"
	got := parseCustomVars(input)
	if got["key1"] != "val=ue" {
		t.Errorf("expected 'val=ue', got %q", got["key1"])
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		slice    []string
		item     string
		expected bool
	}{
		{[]string{"a", "b", "c"}, "b", true},
		{[]string{"a", "b", "c"}, "d", false},
		{[]string{}, "a", false},
		{nil, "a", false},
	}

	for _, tt := range tests {
		got := contains(tt.slice, tt.item)
		if got != tt.expected {
			t.Errorf("contains(%v, %q) = %v, want %v", tt.slice, tt.item, got, tt.expected)
		}
	}
}

func TestAilloyTheme_NotNil(t *testing.T) {
	theme := ailloyTheme()
	if theme == nil {
		t.Fatal("expected non-nil theme")
	}
}

func TestGetFluxString(t *testing.T) {
	flux := map[string]any{
		"project": map[string]any{
			"board": "Engineering",
		},
	}

	val, ok := getFluxString(flux, "project.board")
	if !ok || val != "Engineering" {
		t.Errorf("expected Engineering, got %q", val)
	}

	_, ok = getFluxString(flux, "missing.path")
	if ok {
		t.Error("expected missing path to return false")
	}
}

func TestGetFluxBool(t *testing.T) {
	flux := map[string]any{
		"ore": map[string]any{
			"status": map[string]any{
				"enabled": true,
			},
		},
	}

	val, ok := getFluxBool(flux, "ore.status.enabled")
	if !ok || !val {
		t.Error("expected true")
	}

	_, ok = getFluxBool(flux, "missing.path")
	if ok {
		t.Error("expected missing path to return false")
	}
}
