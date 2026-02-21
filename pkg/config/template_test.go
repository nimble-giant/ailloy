package config

import (
	"bytes"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --- preProcessTemplate tests ---

func TestPreProcessTemplate_BareVariable(t *testing.T) {
	input := "Hello {{name}}!"
	expected := "Hello {{.name}}!"
	result := preProcessTemplate(input)
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestPreProcessTemplate_DottedPath(t *testing.T) {
	input := "Field: {{ore.status.field_id}}"
	expected := "Field: {{.ore.status.field_id}}"
	result := preProcessTemplate(input)
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestPreProcessTemplate_AlreadyDotted(t *testing.T) {
	input := "Field: {{.ore.status.field_id}}"
	result := preProcessTemplate(input)
	if result != input {
		t.Errorf("expected no change, got %q", result)
	}
}

func TestPreProcessTemplate_GoTemplateKeywords(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"end", "{{end}}"},
		{"else", "{{else}}"},
		{"nil", "{{nil}}"},
		{"true", "{{true}}"},
		{"false", "{{false}}"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := preProcessTemplate(tt.input)
			if result != tt.input {
				t.Errorf("keyword %q should not be modified: got %q", tt.name, result)
			}
		})
	}
}

func TestPreProcessTemplate_GoTemplateDirectives(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"if", "{{if .ore.status.enabled}}content{{end}}"},
		{"range", "{{range $k, $v := .ore.status.options}}item{{end}}"},
		{"with", "{{with .ore}}data{{end}}"},
		{"trim markers", "{{- if .X -}}content{{- end -}}"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := preProcessTemplate(tt.input)
			if result != tt.input {
				t.Errorf("directive should not be modified: expected %q, got %q", tt.input, result)
			}
		})
	}
}

func TestPreProcessTemplate_MixedContent(t *testing.T) {
	input := `Hello {{name}}! {{if .ore.status.enabled}}Status: {{default_status}}{{end}}`
	expected := `Hello {{.name}}! {{if .ore.status.enabled}}Status: {{.default_status}}{{end}}`
	result := preProcessTemplate(input)
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestPreProcessTemplate_WithWhitespace(t *testing.T) {
	input := "Hello {{ name }}!"
	expected := "Hello {{ .name }}!"
	result := preProcessTemplate(input)
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestPreProcessTemplate_MultipleVariables(t *testing.T) {
	input := "{{org}}/{{repo}} on {{board}}"
	expected := "{{.org}}/{{.repo}} on {{.board}}"
	result := preProcessTemplate(input)
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

// --- BuildTemplateData tests ---

func TestBuildTemplateData_FlatVariables(t *testing.T) {
	flux := map[string]any{
		"name":  "test",
		"board": "Engineering",
	}
	data := BuildTemplateData(flux, nil)

	if data["name"] != "test" {
		t.Errorf("expected name='test', got %v", data["name"])
	}
	if data["board"] != "Engineering" {
		t.Errorf("expected board='Engineering', got %v", data["board"])
	}
}

func TestBuildTemplateData_OrePresent(t *testing.T) {
	ore := &Ore{
		Status: OreConfig{
			Enabled: true,
			FieldID: "PVTSSF_abc",
			Options: map[string]OreOption{
				"ready": {Label: "Ready", ID: "opt_1"},
			},
		},
	}
	data := BuildTemplateData(nil, ore)

	oreData, ok := data["ore"].(map[string]any)
	if !ok {
		t.Fatal("expected ore to be a map")
	}
	statusData, ok := oreData["status"].(map[string]any)
	if !ok {
		t.Fatal("expected status to be a map")
	}
	if statusData["enabled"] != true {
		t.Error("expected status.enabled to be true")
	}
	if statusData["field_id"] != "PVTSSF_abc" {
		t.Errorf("expected field_id='PVTSSF_abc', got %v", statusData["field_id"])
	}
	opts, ok := statusData["options"].(map[string]any)
	if !ok {
		t.Fatal("expected options to be a map")
	}
	readyOpt, ok := opts["ready"].(map[string]any)
	if !ok {
		t.Fatal("expected ready option to be a map")
	}
	if readyOpt["label"] != "Ready" {
		t.Errorf("expected label='Ready', got %v", readyOpt["label"])
	}
	if readyOpt["id"] != "opt_1" {
		t.Errorf("expected id='opt_1', got %v", readyOpt["id"])
	}
}

func TestBuildTemplateData_NilOreGetsDefaults(t *testing.T) {
	data := BuildTemplateData(nil, nil)

	oreData, ok := data["ore"].(map[string]any)
	if !ok {
		t.Fatal("expected ore to be present even with nil input")
	}
	statusData, ok := oreData["status"].(map[string]any)
	if !ok {
		t.Fatal("expected status to be present in default ore")
	}
	if statusData["enabled"] != false {
		t.Error("expected default status to be disabled")
	}
}

// --- ProcessTemplate conditional rendering tests ---

func TestProcessTemplate_ConditionalEnabled(t *testing.T) {
	content := `{{if .ore.status.enabled}}Status is ON{{end}}`
	ore := &Ore{
		Status: OreConfig{Enabled: true},
	}

	result, err := ProcessTemplate(content, nil, ore)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "Status is ON" {
		t.Errorf("expected 'Status is ON', got %q", result)
	}
}

func TestProcessTemplate_ConditionalDisabled(t *testing.T) {
	content := `{{if .ore.status.enabled}}Status is ON{{end}}`
	ore := &Ore{
		Status: OreConfig{Enabled: false},
	}

	result, err := ProcessTemplate(content, nil, ore)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "" {
		t.Errorf("expected empty string when disabled, got %q", result)
	}
}

func TestProcessTemplate_ConditionalElse(t *testing.T) {
	content := `{{if .ore.status.enabled}}ON{{else}}OFF{{end}}`
	ore := &Ore{
		Status: OreConfig{Enabled: false},
	}

	result, err := ProcessTemplate(content, nil, ore)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "OFF" {
		t.Errorf("expected 'OFF', got %q", result)
	}
}

func TestProcessTemplate_NestedOreAccess(t *testing.T) {
	content := `Field: {{.ore.status.field_id}}`
	ore := &Ore{
		Status: OreConfig{
			Enabled: true,
			FieldID: "PVTSSF_abc123",
		},
	}

	result, err := ProcessTemplate(content, nil, ore)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "Field: PVTSSF_abc123"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestProcessTemplate_OptionAccess(t *testing.T) {
	content := `Ready: {{.ore.status.options.ready.label}} ({{.ore.status.options.ready.id}})`
	ore := &Ore{
		Status: OreConfig{
			Enabled: true,
			Options: map[string]OreOption{
				"ready": {Label: "Not Started", ID: "opt_abc"},
			},
		},
	}

	result, err := ProcessTemplate(content, nil, ore)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "Ready: Not Started (opt_abc)"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestProcessTemplate_RangeOverOptions(t *testing.T) {
	content := `{{range $key, $opt := .ore.priority.options}}{{$opt.label}} {{end}}`
	ore := &Ore{
		Priority: OreConfig{
			Enabled: true,
			Options: map[string]OreOption{
				"p0": {Label: "P0"},
				"p1": {Label: "P1"},
			},
		},
	}

	result, err := ProcessTemplate(content, nil, ore)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Map iteration order is non-deterministic, but both labels should appear
	if !strings.Contains(result, "P0") {
		t.Error("expected 'P0' in range output")
	}
	if !strings.Contains(result, "P1") {
		t.Error("expected 'P1' in range output")
	}
}

func TestProcessTemplate_OrConditional(t *testing.T) {
	content := `{{if or .ore.status.enabled .ore.priority.enabled}}HAS ORE{{end}}`
	ore := &Ore{
		Status:   OreConfig{Enabled: false},
		Priority: OreConfig{Enabled: true},
	}

	result, err := ProcessTemplate(content, nil, ore)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "HAS ORE" {
		t.Errorf("expected 'HAS ORE', got %q", result)
	}
}

// --- Bare variable usability tests ---

func TestProcessTemplate_BareVariables(t *testing.T) {
	content := "Board: {{default_board}}, Status: {{default_status}}"
	flux := map[string]any{
		"default_board":  "Engineering",
		"default_status": "Ready",
	}

	result, err := ProcessTemplate(content, flux, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "Board: Engineering, Status: Ready"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestProcessTemplate_MixedBareAndConditionals(t *testing.T) {
	content := `Board: {{default_board}}
{{if .ore.status.enabled}}Status Field: {{.ore.status.field_id}}{{end}}`

	flux := map[string]any{
		"default_board": "Engineering",
	}
	ore := &Ore{
		Status: OreConfig{
			Enabled: true,
			FieldID: "PVTSSF_xyz",
		},
	}

	result, err := ProcessTemplate(content, flux, ore)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "Board: Engineering") {
		t.Error("expected bare variable to be resolved")
	}
	if !strings.Contains(result, "Status Field: PVTSSF_xyz") {
		t.Error("expected ore field to be resolved")
	}
}

func TestProcessTemplate_BareDottedPath(t *testing.T) {
	content := "Field: {{ore.status.field_id}}"
	ore := &Ore{
		Status: OreConfig{
			Enabled: true,
			FieldID: "PVTSSF_test",
		},
	}

	result, err := ProcessTemplate(content, nil, ore)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "Field: PVTSSF_test"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

// --- Unresolved variable warning tests ---

func TestWarnUnresolvedVars_LogsWarning(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	log.SetFlags(0) // Remove timestamp for easier testing
	defer func() {
		log.SetOutput(nil)
		log.SetFlags(log.LstdFlags)
	}()

	content := "Value: {{missing_var}}"
	_, err := ProcessTemplate(content, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	logOutput := buf.String()
	if !strings.Contains(logOutput, "unresolved template variable") {
		t.Errorf("expected warning about unresolved variable, got log: %q", logOutput)
	}
	if !strings.Contains(logOutput, "missing_var") {
		t.Errorf("expected variable name in warning, got log: %q", logOutput)
	}
}

func TestWarnUnresolvedVars_NoWarningForResolved(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	log.SetFlags(0)
	defer func() {
		log.SetOutput(nil)
		log.SetFlags(log.LstdFlags)
	}()

	content := "Board: {{default_board}}"
	flux := map[string]any{"default_board": "Engineering"}
	_, err := ProcessTemplate(content, flux, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	logOutput := buf.String()
	if strings.Contains(logOutput, "unresolved") {
		t.Errorf("expected no warning for resolved variable, got log: %q", logOutput)
	}
}

// --- resolveDataPath tests ---

func TestResolveDataPath_TopLevel(t *testing.T) {
	data := map[string]any{"name": "test"}
	if !resolveDataPath(data, "name") {
		t.Error("expected 'name' to resolve")
	}
}

func TestResolveDataPath_Nested(t *testing.T) {
	data := map[string]any{
		"ore": map[string]any{
			"status": map[string]any{
				"field_id": "abc",
			},
		},
	}
	if !resolveDataPath(data, "ore.status.field_id") {
		t.Error("expected 'ore.status.field_id' to resolve")
	}
}

func TestResolveDataPath_Missing(t *testing.T) {
	data := map[string]any{"name": "test"}
	if resolveDataPath(data, "missing") {
		t.Error("expected 'missing' to not resolve")
	}
}

func TestResolveDataPath_PartiallyMissing(t *testing.T) {
	data := map[string]any{
		"ore": map[string]any{
			"status": map[string]any{},
		},
	}
	if resolveDataPath(data, "ore.status.nonexistent") {
		t.Error("expected partial path to not resolve")
	}
}

// --- Ingot template function tests ---

func TestProcessTemplate_IngotFunction(t *testing.T) {
	dir := t.TempDir()
	ingotDir := filepath.Join(dir, "ingots")
	if err := os.MkdirAll(ingotDir, 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ingotDir, "footer.md"), []byte("-- end --"), 0644); err != nil {
		t.Fatal(err)
	}

	r := NewIngotResolver([]string{dir}, nil, nil)
	content := `Hello {{ingot "footer"}}`
	result, err := ProcessTemplate(content, nil, nil, WithIngotResolver(r))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "Hello -- end --" {
		t.Errorf("expected 'Hello -- end --', got %q", result)
	}
}

func TestProcessTemplate_IngotWithFlux(t *testing.T) {
	dir := t.TempDir()
	ingotDir := filepath.Join(dir, "ingots")
	if err := os.MkdirAll(ingotDir, 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ingotDir, "banner.md"), []byte("Welcome to {{organization}}"), 0644); err != nil {
		t.Fatal(err)
	}

	flux := map[string]any{"organization": "Acme"}
	r := NewIngotResolver([]string{dir}, flux, nil)
	content := `{{ingot "banner"}} Corp`
	result, err := ProcessTemplate(content, flux, nil, WithIngotResolver(r))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "Welcome to Acme Corp" {
		t.Errorf("expected 'Welcome to Acme Corp', got %q", result)
	}
}

func TestProcessTemplate_WithoutIngotResolver(t *testing.T) {
	// Without a resolver, {{ingot "x"}} should cause a parse error since the function isn't registered
	content := `{{ingot "test"}}`
	_, err := ProcessTemplate(content, nil, nil)
	if err == nil {
		t.Error("expected error when ingot function is not registered")
	}
}

// --- Error handling tests ---

func TestProcessTemplate_InvalidTemplateSyntax(t *testing.T) {
	content := "{{if}}missing condition{{end}}"
	_, err := ProcessTemplate(content, nil, nil)
	if err == nil {
		t.Error("expected error for invalid template syntax")
	}
}

func TestProcessTemplate_EmptyContentReturnsEmpty(t *testing.T) {
	result, err := ProcessTemplate("", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestProcessTemplate_NilEverything(t *testing.T) {
	result, err := ProcessTemplate("plain text", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "plain text" {
		t.Errorf("expected 'plain text', got %q", result)
	}
}
