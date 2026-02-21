package commands

import (
	"strings"
	"testing"

	"github.com/nimble-giant/ailloy/pkg/config"
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

func TestBuildCustomVarsText_ExcludesManagedVars(t *testing.T) {
	cfg := &config.Config{
		Templates: config.TemplateConfig{
			Flux: map[string]any{
				"default_board":      "Engineering",
				"organization":       "acme",
				"project_id":         "PVT_123",
				"custom_var":         "custom_value",
				"another_custom_var": "another_value",
			},
		},
	}

	got := buildCustomVarsText(cfg)
	if strings.Contains(got, "default_board") {
		t.Error("should exclude managed var default_board")
	}
	if strings.Contains(got, "organization") {
		t.Error("should exclude managed var organization")
	}
	if strings.Contains(got, "project_id") {
		t.Error("should exclude managed var project_id")
	}
	if !strings.Contains(got, "custom_var=custom_value") {
		t.Error("should include custom_var")
	}
	if !strings.Contains(got, "another_custom_var=another_value") {
		t.Error("should include another_custom_var")
	}
}

func TestBuildSummaryDiff_NewValues(t *testing.T) {
	origVars := map[string]string{}
	origOre := oreSnapshot{}

	got := buildSummaryDiff(
		origVars, origOre,
		"Engineering", "acme",
		[]string{"status", "priority"},
		"PVTSSF_1", "", "",
		"",
	)

	if !strings.Contains(got, "+ default_board: Engineering") {
		t.Error("expected new default_board in diff")
	}
	if !strings.Contains(got, "+ organization: acme") {
		t.Error("expected new organization in diff")
	}
	if !strings.Contains(got, "+ Status ore: enabled") {
		t.Error("expected status ore enabled")
	}
	if !strings.Contains(got, "+ Priority ore: enabled") {
		t.Error("expected priority ore enabled")
	}
	if !strings.Contains(got, "Iteration ore: disabled (unchanged)") {
		t.Error("expected iteration ore unchanged")
	}
}

func TestBuildSummaryDiff_ChangedValues(t *testing.T) {
	origVars := map[string]string{
		"default_board": "OldBoard",
		"organization":  "oldorg",
	}
	origOre := oreSnapshot{
		StatusEnabled: true,
	}

	got := buildSummaryDiff(
		origVars, origOre,
		"NewBoard", "neworg",
		[]string{"status"},
		"", "", "",
		"",
	)

	if !strings.Contains(got, "~ default_board: OldBoard -> NewBoard") {
		t.Error("expected changed default_board in diff")
	}
	if !strings.Contains(got, "~ organization: oldorg -> neworg") {
		t.Error("expected changed organization in diff")
	}
	if !strings.Contains(got, "Status ore: enabled (unchanged)") {
		t.Error("expected status unchanged")
	}
}

func TestBuildSummaryDiff_UnchangedValues(t *testing.T) {
	origVars := map[string]string{
		"default_board": "Board",
		"organization":  "org",
	}
	origOre := oreSnapshot{
		StatusEnabled: true,
	}

	got := buildSummaryDiff(
		origVars, origOre,
		"Board", "org",
		[]string{"status"},
		"", "", "",
		"",
	)

	if !strings.Contains(got, "default_board: Board (unchanged)") {
		t.Error("expected unchanged default_board")
	}
}

func TestBuildSummaryDiff_CustomVars(t *testing.T) {
	got := buildSummaryDiff(
		map[string]string{}, oreSnapshot{},
		"", "",
		nil,
		"", "", "",
		"reviewer=alice\nchannel=#eng",
	)

	if !strings.Contains(got, "Custom variables:") {
		t.Error("expected custom variables section")
	}
	if !strings.Contains(got, "reviewer = alice") {
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

func TestSnapshotVars(t *testing.T) {
	cfg := &config.Config{
		Templates: config.TemplateConfig{
			Flux: map[string]any{"a": "1", "b": "2"},
		},
	}

	snap := snapshotVars(cfg)

	// Modify original
	cfg.Templates.Flux["a"] = "changed"

	// Snapshot should be unaffected
	if snap["a"] != "1" {
		t.Error("snapshot should be independent of original")
	}
}

func TestSnapshotOre(t *testing.T) {
	cfg := &config.Config{
		Ore: config.Ore{
			Status:   config.OreConfig{Enabled: true, FieldID: "f1"},
			Priority: config.OreConfig{Enabled: false},
		},
	}

	snap := snapshotOre(cfg)

	if !snap.StatusEnabled {
		t.Error("expected status enabled")
	}
	if snap.PriorityEnabled {
		t.Error("expected priority disabled")
	}
	if snap.StatusFieldID != "f1" {
		t.Errorf("expected f1, got %s", snap.StatusFieldID)
	}
}

func TestAilloyTheme_NotNil(t *testing.T) {
	theme := ailloyTheme()
	if theme == nil {
		t.Fatal("expected non-nil theme")
	}
}

func TestApplyWizardResults_BasicVars(t *testing.T) {
	cfg := &config.Config{
		Templates: config.TemplateConfig{
			Flux: make(map[string]any),
		},
		Ore: config.DefaultOre(),
	}

	applyWizardResults(cfg,
		"Engineering", "acme",
		[]string{"status", "priority"},
		"", "", "",
		"custom_key=custom_val",
		false, "", nil,
	)

	if cfg.Templates.Flux["default_board"] != "Engineering" {
		t.Error("expected default_board to be set")
	}
	if cfg.Templates.Flux["organization"] != "acme" {
		t.Error("expected organization to be set")
	}
	if !cfg.Ore.Status.Enabled {
		t.Error("expected status ore enabled")
	}
	if !cfg.Ore.Priority.Enabled {
		t.Error("expected priority ore enabled")
	}
	if cfg.Ore.Iteration.Enabled {
		t.Error("expected iteration ore disabled")
	}
	if cfg.Templates.Flux["custom_key"] != "custom_val" {
		t.Error("expected custom_key to be set")
	}
}

func TestApplyWizardResults_ProjectID(t *testing.T) {
	cfg := &config.Config{
		Templates: config.TemplateConfig{
			Flux: make(map[string]any),
		},
		Ore: config.DefaultOre(),
	}

	applyWizardResults(cfg,
		"", "",
		nil,
		"", "", "",
		"",
		true, "5:PVT_abc123", nil,
	)

	if cfg.Templates.Flux["project_id"] != "PVT_abc123" {
		t.Errorf("expected project_id PVT_abc123, got %q", cfg.Templates.Flux["project_id"])
	}
}
