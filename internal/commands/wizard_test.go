package commands

import (
	"strings"
	"testing"

	"github.com/nimble-giant/ailloy/pkg/mold"
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

func TestBuildSummaryPreview_WithValues(t *testing.T) {
	got := buildSummaryPreview(
		"Engineering", "acme",
		[]string{"status", "priority"},
		"PVTSSF_1", "", "",
		"",
	)

	if !strings.Contains(got, "project.board: Engineering") {
		t.Error("expected project.board in preview")
	}
	if !strings.Contains(got, "project.organization: acme") {
		t.Error("expected project.organization in preview")
	}
	if !strings.Contains(got, "ore.status.enabled: enabled") {
		t.Error("expected status ore enabled")
	}
	if !strings.Contains(got, "ore.priority.enabled: enabled") {
		t.Error("expected priority ore enabled")
	}
	if !strings.Contains(got, "ore.iteration.enabled: disabled") {
		t.Error("expected iteration ore disabled")
	}
	if !strings.Contains(got, "ore.status.field_id: PVTSSF_1") {
		t.Error("expected status field ID")
	}
}

func TestBuildSummaryPreview_Empty(t *testing.T) {
	got := buildSummaryPreview(
		"", "",
		nil,
		"", "", "",
		"",
	)

	if !strings.Contains(got, "Flux values to write") {
		t.Error("expected header")
	}
	// All ore should be disabled
	if !strings.Contains(got, "ore.status.enabled: disabled") {
		t.Error("expected status disabled")
	}
}

func TestBuildSummaryPreview_CustomVars(t *testing.T) {
	got := buildSummaryPreview(
		"", "",
		nil,
		"", "", "",
		"reviewer=alice\nchannel=#eng",
	)

	if !strings.Contains(got, "Custom variables:") {
		t.Error("expected custom variables section")
	}
	if !strings.Contains(got, "reviewer: alice") {
		t.Error("expected reviewer custom var")
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

func TestParseBoardNumber(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"5:PVT_abc123", 5},
		{"12:PVT_def456", 12},
		{"invalid", 0},
		{"", 0},
		{":PVT_abc", 0},
	}

	for _, tt := range tests {
		got := parseBoardNumber(tt.input)
		if got != tt.expected {
			t.Errorf("parseBoardNumber(%q) = %d, want %d", tt.input, got, tt.expected)
		}
	}
}

func TestParseBoardID(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"5:PVT_abc123", "PVT_abc123"},
		{"12:PVT_def456", "PVT_def456"},
		{"invalid", ""},
		{"", ""},
	}

	for _, tt := range tests {
		got := parseBoardID(tt.input)
		if got != tt.expected {
			t.Errorf("parseBoardID(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestAilloyTheme_NotNil(t *testing.T) {
	theme := ailloyTheme()
	if theme == nil {
		t.Fatal("expected non-nil theme")
	}
}

func TestApplyWizardResults_BasicVars(t *testing.T) {
	flux := make(map[string]any)

	applyWizardResults(flux,
		"Engineering", "acme",
		[]string{"status", "priority"},
		"", "", "",
		"custom_key=custom_val",
		false, "", nil,
	)

	boardVal, ok := mold.GetNestedAny(flux, "project.board")
	if !ok || boardVal != "Engineering" {
		t.Error("expected project.board to be set")
	}
	orgVal, ok := mold.GetNestedAny(flux, "project.organization")
	if !ok || orgVal != "acme" {
		t.Error("expected project.organization to be set")
	}
	statusEnabled, ok := mold.GetNestedAny(flux, "ore.status.enabled")
	if !ok || statusEnabled != true {
		t.Error("expected status ore enabled")
	}
	priorityEnabled, ok := mold.GetNestedAny(flux, "ore.priority.enabled")
	if !ok || priorityEnabled != true {
		t.Error("expected priority ore enabled")
	}
	iterEnabled, ok := mold.GetNestedAny(flux, "ore.iteration.enabled")
	if !ok || iterEnabled != false {
		t.Error("expected iteration ore disabled")
	}
	customVal, ok := mold.GetNestedAny(flux, "custom_key")
	if !ok || customVal != "custom_val" {
		t.Error("expected custom_key to be set")
	}
}

func TestApplyWizardResults_ProjectID(t *testing.T) {
	flux := make(map[string]any)

	applyWizardResults(flux,
		"", "",
		nil,
		"", "", "",
		"",
		true, "5:PVT_abc123", nil,
	)

	projectID, ok := mold.GetNestedAny(flux, "project.id")
	if !ok || projectID != "PVT_abc123" {
		t.Errorf("expected project.id PVT_abc123, got %v", projectID)
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
