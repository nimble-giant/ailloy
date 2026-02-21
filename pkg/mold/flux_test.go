package mold

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

// --- ApplyFluxDefaults tests ---

func TestApplyFluxDefaults_AppliesMissingDefaults(t *testing.T) {
	schema := []FluxVar{
		{Name: "board", Type: "string", Default: "Engineering"},
		{Name: "org", Type: "string", Required: true},
	}
	flux := map[string]any{"org": "acme"}

	result := ApplyFluxDefaults(schema, flux)

	if result["board"] != "Engineering" {
		t.Errorf("expected board=Engineering, got %q", result["board"])
	}
	if result["org"] != "acme" {
		t.Errorf("expected org=acme, got %q", result["org"])
	}
}

func TestApplyFluxDefaults_DoesNotOverrideExisting(t *testing.T) {
	schema := []FluxVar{
		{Name: "board", Type: "string", Default: "Engineering"},
	}
	flux := map[string]any{"board": "Product"}

	result := ApplyFluxDefaults(schema, flux)

	if result["board"] != "Product" {
		t.Errorf("expected board=Product (not overridden), got %q", result["board"])
	}
}

func TestApplyFluxDefaults_DoesNotMutateInput(t *testing.T) {
	schema := []FluxVar{
		{Name: "board", Type: "string", Default: "Engineering"},
	}
	flux := map[string]any{}

	result := ApplyFluxDefaults(schema, flux)

	if _, exists := flux["board"]; exists {
		t.Error("input map should not be mutated")
	}
	if result["board"] != "Engineering" {
		t.Errorf("expected board=Engineering in result, got %q", result["board"])
	}
}

func TestApplyFluxDefaults_EmptySchema(t *testing.T) {
	flux := map[string]any{"key": "val"}
	result := ApplyFluxDefaults(nil, flux)

	if result["key"] != "val" {
		t.Errorf("expected key=val, got %q", result["key"])
	}
}

func TestApplyFluxDefaults_SkipsEmptyDefault(t *testing.T) {
	schema := []FluxVar{
		{Name: "org", Type: "string", Required: true},
	}
	flux := map[string]any{}

	result := ApplyFluxDefaults(schema, flux)

	if _, exists := result["org"]; exists {
		t.Error("should not set value for var with no default")
	}
}

// --- ValidateFlux tests ---

func TestValidateFlux_RequiredMissing(t *testing.T) {
	schema := []FluxVar{
		{Name: "org", Type: "string", Required: true},
	}
	err := ValidateFlux(schema, map[string]any{})
	if err == nil {
		t.Fatal("expected error for missing required var")
	}
	if !strings.Contains(err.Error(), "org") {
		t.Errorf("expected error to mention 'org', got: %v", err)
	}
}

func TestValidateFlux_RequiredEmpty(t *testing.T) {
	schema := []FluxVar{
		{Name: "org", Type: "string", Required: true},
	}
	err := ValidateFlux(schema, map[string]any{"org": ""})
	if err == nil {
		t.Fatal("expected error for empty required var")
	}
}

func TestValidateFlux_RequiredPresent(t *testing.T) {
	schema := []FluxVar{
		{Name: "org", Type: "string", Required: true},
	}
	if err := ValidateFlux(schema, map[string]any{"org": "acme"}); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateFlux_OptionalMissing(t *testing.T) {
	schema := []FluxVar{
		{Name: "board", Type: "string", Required: false},
	}
	if err := ValidateFlux(schema, map[string]any{}); err != nil {
		t.Errorf("unexpected error for missing optional var: %v", err)
	}
}

func TestValidateFlux_BoolValid(t *testing.T) {
	schema := []FluxVar{
		{Name: "enabled", Type: "bool", Required: true},
	}
	for _, val := range []string{"true", "false", "True", "FALSE"} {
		if err := ValidateFlux(schema, map[string]any{"enabled": val}); err != nil {
			t.Errorf("expected %q to be valid bool, got: %v", val, err)
		}
	}
}

func TestValidateFlux_BoolInvalid(t *testing.T) {
	schema := []FluxVar{
		{Name: "enabled", Type: "bool", Required: true},
	}
	err := ValidateFlux(schema, map[string]any{"enabled": "yes"})
	if err == nil {
		t.Fatal("expected error for invalid bool")
	}
	if !strings.Contains(err.Error(), "must be a bool") {
		t.Errorf("expected bool error, got: %v", err)
	}
}

func TestValidateFlux_IntValid(t *testing.T) {
	schema := []FluxVar{
		{Name: "count", Type: "int", Required: true},
	}
	for _, val := range []string{"0", "42", "-1", "100"} {
		if err := ValidateFlux(schema, map[string]any{"count": val}); err != nil {
			t.Errorf("expected %q to be valid int, got: %v", val, err)
		}
	}
}

func TestValidateFlux_IntInvalid(t *testing.T) {
	schema := []FluxVar{
		{Name: "count", Type: "int", Required: true},
	}
	err := ValidateFlux(schema, map[string]any{"count": "abc"})
	if err == nil {
		t.Fatal("expected error for invalid int")
	}
	if !strings.Contains(err.Error(), "must be an int") {
		t.Errorf("expected int error, got: %v", err)
	}
}

func TestValidateFlux_ListValid(t *testing.T) {
	schema := []FluxVar{
		{Name: "tags", Type: "list", Required: true},
	}
	for _, val := range []string{"a,b,c", "single", "one,two"} {
		if err := ValidateFlux(schema, map[string]any{"tags": val}); err != nil {
			t.Errorf("expected %q to be valid list, got: %v", val, err)
		}
	}
}

func TestValidateFlux_StringAlwaysValid(t *testing.T) {
	schema := []FluxVar{
		{Name: "name", Type: "string", Required: true},
	}
	for _, val := range []string{"hello", "123", "true", "a,b,c", "anything goes"} {
		if err := ValidateFlux(schema, map[string]any{"name": val}); err != nil {
			t.Errorf("expected %q to be valid string, got: %v", val, err)
		}
	}
}

func TestValidateFlux_MultipleErrors(t *testing.T) {
	schema := []FluxVar{
		{Name: "org", Type: "string", Required: true},
		{Name: "enabled", Type: "bool", Required: true},
		{Name: "count", Type: "int", Required: true},
	}
	err := ValidateFlux(schema, map[string]any{})
	if err == nil {
		t.Fatal("expected multiple errors")
	}
	errMsg := err.Error()
	if !strings.Contains(errMsg, "org") {
		t.Error("expected error to mention 'org'")
	}
	if !strings.Contains(errMsg, "enabled") {
		t.Error("expected error to mention 'enabled'")
	}
	if !strings.Contains(errMsg, "count") {
		t.Error("expected error to mention 'count'")
	}
}

func TestValidateFlux_EmptySchema(t *testing.T) {
	if err := ValidateFlux(nil, map[string]any{"extra": "val"}); err != nil {
		t.Errorf("expected no error for empty schema, got: %v", err)
	}
}

func TestValidateFlux_TypeAndRequiredCombined(t *testing.T) {
	schema := []FluxVar{
		{Name: "count", Type: "int", Required: true},
	}
	// Provided but wrong type
	err := ValidateFlux(schema, map[string]any{"count": "not-a-number"})
	if err == nil {
		t.Fatal("expected error for wrong type")
	}
	if !strings.Contains(err.Error(), "must be an int") {
		t.Errorf("expected type error, got: %v", err)
	}
}

// --- LoadFluxFile tests ---

func TestLoadFluxFile_Valid(t *testing.T) {
	fsys := fstest.MapFS{
		"flux.yaml": &fstest.MapFile{Data: []byte("org: acme\nboard: Engineering\n")},
	}
	vals, err := LoadFluxFile(fsys, "flux.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if vals["org"] != "acme" {
		t.Errorf("expected org=acme, got %q", vals["org"])
	}
	if vals["board"] != "Engineering" {
		t.Errorf("expected board=Engineering, got %q", vals["board"])
	}
}

func TestLoadFluxFile_Missing(t *testing.T) {
	fsys := fstest.MapFS{}
	vals, err := LoadFluxFile(fsys, "flux.yaml")
	if err != nil {
		t.Fatalf("unexpected error for missing file: %v", err)
	}
	if len(vals) != 0 {
		t.Errorf("expected empty map, got %v", vals)
	}
}

func TestLoadFluxFile_Empty(t *testing.T) {
	fsys := fstest.MapFS{
		"flux.yaml": &fstest.MapFile{Data: []byte("")},
	}
	vals, err := LoadFluxFile(fsys, "flux.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(vals) != 0 {
		t.Errorf("expected empty map, got %v", vals)
	}
}

func TestLoadFluxFile_InvalidYAML(t *testing.T) {
	fsys := fstest.MapFS{
		"flux.yaml": &fstest.MapFile{Data: []byte("{{{bad yaml")},
	}
	_, err := LoadFluxFile(fsys, "flux.yaml")
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

// --- LoadFluxSchema tests ---

func TestLoadFluxSchema_Valid(t *testing.T) {
	fsys := fstest.MapFS{
		"flux.schema.yaml": &fstest.MapFile{Data: []byte(`
- name: org
  type: string
  required: true
- name: board
  type: string
`)},
	}
	schema, err := LoadFluxSchema(fsys, "flux.schema.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(schema) != 2 {
		t.Fatalf("expected 2 schema entries, got %d", len(schema))
	}
	if schema[0].Name != "org" || !schema[0].Required {
		t.Errorf("expected org required, got %+v", schema[0])
	}
}

func TestLoadFluxSchema_Missing(t *testing.T) {
	fsys := fstest.MapFS{}
	schema, err := LoadFluxSchema(fsys, "flux.schema.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if schema != nil {
		t.Errorf("expected nil schema, got %v", schema)
	}
}

func TestLoadFluxSchema_InvalidYAML(t *testing.T) {
	fsys := fstest.MapFS{
		"flux.schema.yaml": &fstest.MapFile{Data: []byte("{{{bad")},
	}
	_, err := LoadFluxSchema(fsys, "flux.schema.yaml")
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

// --- ApplyFluxFileDefaults tests ---

func TestApplyFluxFileDefaults_FillsGaps(t *testing.T) {
	defaults := map[string]any{"board": "Engineering", "cli": "gh"}
	flux := map[string]any{"board": "Product"}

	result := ApplyFluxFileDefaults(defaults, flux)

	if result["board"] != "Product" {
		t.Errorf("expected board=Product (not overridden), got %q", result["board"])
	}
	if result["cli"] != "gh" {
		t.Errorf("expected cli=gh (filled from defaults), got %q", result["cli"])
	}
}

func TestApplyFluxFileDefaults_DoesNotMutateInput(t *testing.T) {
	defaults := map[string]any{"board": "Engineering"}
	flux := map[string]any{}

	result := ApplyFluxFileDefaults(defaults, flux)

	if _, exists := flux["board"]; exists {
		t.Error("input map should not be mutated")
	}
	if result["board"] != "Engineering" {
		t.Errorf("expected board=Engineering in result, got %q", result["board"])
	}
}

func TestApplyFluxFileDefaults_EmptyDefaults(t *testing.T) {
	flux := map[string]any{"key": "val"}
	result := ApplyFluxFileDefaults(nil, flux)
	if result["key"] != "val" {
		t.Errorf("expected key=val, got %q", result["key"])
	}
}

// --- LoadFluxFile multiline tests ---

func TestLoadFluxFile_MultilineValues(t *testing.T) {
	fsys := fstest.MapFS{
		"flux.yaml": &fstest.MapFile{Data: []byte("simple: hello\nmultiline: |-\n  line one\n  line two\n  line three\n")},
	}
	vals, err := LoadFluxFile(fsys, "flux.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if vals["simple"] != "hello" {
		t.Errorf("expected simple=hello, got %q", vals["simple"])
	}
	multi, _ := vals["multiline"].(string)
	if !strings.Contains(multi, "line one") {
		t.Errorf("expected multiline to contain 'line one', got %q", multi)
	}
	if !strings.Contains(multi, "line three") {
		t.Errorf("expected multiline to contain 'line three', got %q", multi)
	}
}

// --- Full precedence chain tests ---

func TestPrecedenceChain_SchemaDefaultsLessThanFluxFile(t *testing.T) {
	// Schema provides defaults for "board" and "cli"
	schema := []FluxVar{
		{Name: "board", Type: "string", Default: "Schema-Board"},
		{Name: "cli", Type: "string", Default: "schema-cli"},
	}

	// flux.yaml provides defaults for "board" and "org"
	fluxFileDefaults := map[string]any{
		"board": "FluxFile-Board",
		"org":   "flux-org",
	}

	// Start with empty user flux
	flux := make(map[string]any)

	// Step 1: Apply schema defaults (lowest priority)
	flux = ApplyFluxDefaults(schema, flux)

	// Step 2: Apply flux.yaml defaults (higher priority, fills remaining gaps)
	flux = ApplyFluxFileDefaults(fluxFileDefaults, flux)

	// Schema default for "cli" should survive (not in flux.yaml)
	if flux["cli"] != "schema-cli" {
		t.Errorf("expected cli=schema-cli (from schema), got %q", flux["cli"])
	}
	// Schema set "board" first, flux.yaml should NOT override it
	// (ApplyFluxFileDefaults only fills gaps)
	if flux["board"] != "Schema-Board" {
		t.Errorf("expected board=Schema-Board (schema applied first), got %q", flux["board"])
	}
	// flux.yaml adds "org"
	if flux["org"] != "flux-org" {
		t.Errorf("expected org=flux-org (from flux.yaml), got %q", flux["org"])
	}
}

func TestPrecedenceChain_UserOverridesEverything(t *testing.T) {
	schema := []FluxVar{
		{Name: "board", Type: "string", Default: "Schema-Board"},
		{Name: "cli", Type: "string", Default: "schema-cli"},
	}
	fluxFileDefaults := map[string]any{
		"board": "FluxFile-Board",
		"cli":   "flux-cli",
	}

	// User has already set "board" via config
	userFlux := map[string]any{"board": "User-Board"}

	// Step 1: Apply schema defaults
	result := ApplyFluxDefaults(schema, userFlux)
	// Step 2: Apply flux.yaml defaults
	result = ApplyFluxFileDefaults(fluxFileDefaults, result)

	// User's "board" wins over both schema and flux.yaml
	if result["board"] != "User-Board" {
		t.Errorf("expected board=User-Board (user override), got %q", result["board"])
	}
	// Schema default for "cli" fills gap, flux.yaml can't override it
	if result["cli"] != "schema-cli" {
		t.Errorf("expected cli=schema-cli (schema applied first), got %q", result["cli"])
	}
}

// --- Schema validation with file-loaded flux ---

func TestValidateFlux_WithFileLoadedSchema(t *testing.T) {
	// Simulate loading schema from flux.schema.yaml
	schemaFS := fstest.MapFS{
		"flux.schema.yaml": &fstest.MapFile{Data: []byte(`
- name: org
  type: string
  required: true
- name: count
  type: int
`)},
	}
	schema, err := LoadFluxSchema(schemaFS, "flux.schema.yaml")
	if err != nil {
		t.Fatalf("unexpected error loading schema: %v", err)
	}

	// Simulate loading flux values from flux.yaml
	fluxFS := fstest.MapFS{
		"flux.yaml": &fstest.MapFile{Data: []byte("org: acme\ncount: 42\n")},
	}
	flux, err := LoadFluxFile(fluxFS, "flux.yaml")
	if err != nil {
		t.Fatalf("unexpected error loading flux: %v", err)
	}

	// Validate — should pass
	if err := ValidateFlux(schema, flux); err != nil {
		t.Errorf("expected valid flux to pass validation, got: %v", err)
	}
}

func TestValidateFlux_FileLoadedSchemaRejectsInvalid(t *testing.T) {
	schemaFS := fstest.MapFS{
		"flux.schema.yaml": &fstest.MapFile{Data: []byte(`
- name: org
  type: string
  required: true
- name: count
  type: int
`)},
	}
	schema, err := LoadFluxSchema(schemaFS, "flux.schema.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Missing required "org" and invalid type for "count"
	flux := map[string]any{"count": "not-a-number"}

	err = ValidateFlux(schema, flux)
	if err == nil {
		t.Fatal("expected validation error")
	}
	errMsg := err.Error()
	if !strings.Contains(errMsg, "org") {
		t.Error("expected error to mention missing 'org'")
	}
	if !strings.Contains(errMsg, "must be an int") {
		t.Error("expected error to mention invalid int for 'count'")
	}
}

func Test_InlineFluxStillWorks(t *testing.T) {
	// Old-style mold with flux: section in mold.yaml
	schema := []FluxVar{
		{Name: "org", Type: "string", Required: true, Default: "default-org"},
		{Name: "board", Type: "string", Default: "Engineering"},
	}

	// No flux.yaml or flux.schema.yaml
	emptyFS := fstest.MapFS{}
	fluxFile, err := LoadFluxFile(emptyFS, "flux.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	fluxSchema, err := LoadFluxSchema(emptyFS, "flux.schema.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// flux.yaml returns empty map, schema returns nil
	if len(fluxFile) != 0 {
		t.Errorf("expected empty flux file, got %v", fluxFile)
	}
	if fluxSchema != nil {
		t.Error("expected nil schema")
	}

	// Apply inline schema defaults
	flux := ApplyFluxDefaults(schema, map[string]any{})

	// Validate with inline schema (since flux.schema.yaml is nil)
	if err := ValidateFlux(schema, flux); err != nil {
		t.Errorf("expected validation to pass with schema defaults, got: %v", err)
	}

	if flux["org"] != "default-org" {
		t.Errorf("expected org=default-org, got %q", flux["org"])
	}
	if flux["board"] != "Engineering" {
		t.Errorf("expected board=Engineering, got %q", flux["board"])
	}
}

// --- LayerFluxFiles tests ---

func TestLayerFluxFiles_SingleFile(t *testing.T) {
	dir := t.TempDir()
	f1 := filepath.Join(dir, "values.yaml")
	if err := os.WriteFile(f1, []byte("org: acme\nboard: Engineering\n"), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := LayerFluxFiles([]string{f1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["org"] != "acme" {
		t.Errorf("expected org=acme, got %v", result["org"])
	}
	if result["board"] != "Engineering" {
		t.Errorf("expected board=Engineering, got %v", result["board"])
	}
}

func TestLayerFluxFiles_OverrideLaterFile(t *testing.T) {
	dir := t.TempDir()
	f1 := filepath.Join(dir, "base.yaml")
	f2 := filepath.Join(dir, "override.yaml")
	if err := os.WriteFile(f1, []byte("org: acme\nboard: Engineering\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(f2, []byte("board: Product\n"), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := LayerFluxFiles([]string{f1, f2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["org"] != "acme" {
		t.Errorf("expected org=acme, got %v", result["org"])
	}
	if result["board"] != "Product" {
		t.Errorf("expected board=Product (override), got %v", result["board"])
	}
}

func TestLayerFluxFiles_NestedOverride(t *testing.T) {
	dir := t.TempDir()
	f1 := filepath.Join(dir, "base.yaml")
	f2 := filepath.Join(dir, "ore.yaml")
	if err := os.WriteFile(f1, []byte("ore:\n  status:\n    enabled: false\n    field_id: \"\"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(f2, []byte("ore:\n  status:\n    enabled: true\n    field_id: \"PVTSSF_abc\"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := LayerFluxFiles([]string{f1, f2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ore, ok := result["ore"].(map[string]any)
	if !ok {
		t.Fatal("expected ore to be a map")
	}
	status, ok := ore["status"].(map[string]any)
	if !ok {
		t.Fatal("expected status to be a map")
	}
	if status["enabled"] != true {
		t.Errorf("expected enabled=true, got %v", status["enabled"])
	}
	if status["field_id"] != "PVTSSF_abc" {
		t.Errorf("expected field_id=PVTSSF_abc, got %v", status["field_id"])
	}
}

func TestLayerFluxFiles_EmptyPaths(t *testing.T) {
	result, err := LayerFluxFiles(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty map, got %v", result)
	}
}

func TestLayerFluxFiles_MissingFileError(t *testing.T) {
	_, err := LayerFluxFiles([]string{"/nonexistent/file.yaml"})
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

// --- ApplySetOverrides tests ---

func TestApplySetOverrides_Simple(t *testing.T) {
	flux := map[string]any{"board": "Engineering"}
	err := ApplySetOverrides(flux, []string{"board=Product"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if flux["board"] != "Product" {
		t.Errorf("expected board=Product, got %v", flux["board"])
	}
}

func TestApplySetOverrides_DottedPath(t *testing.T) {
	flux := map[string]any{}
	err := ApplySetOverrides(flux, []string{"scm.provider=GitLab"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	scm, ok := flux["scm"].(map[string]any)
	if !ok {
		t.Fatal("expected scm to be a map")
	}
	if scm["provider"] != "GitLab" {
		t.Errorf("expected provider=GitLab, got %v", scm["provider"])
	}
}

func TestApplySetOverrides_MultipleFlags(t *testing.T) {
	flux := map[string]any{}
	err := ApplySetOverrides(flux, []string{"org=acme", "board=Product", "scm.provider=GitLab"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if flux["org"] != "acme" {
		t.Errorf("expected org=acme, got %v", flux["org"])
	}
	if flux["board"] != "Product" {
		t.Errorf("expected board=Product, got %v", flux["board"])
	}
}

func TestApplySetOverrides_InvalidFormat(t *testing.T) {
	flux := map[string]any{}
	err := ApplySetOverrides(flux, []string{"no-equals-sign"})
	if err == nil {
		t.Fatal("expected error for invalid format")
	}
	if !strings.Contains(err.Error(), "invalid --set format") {
		t.Errorf("expected format error, got: %v", err)
	}
}

func TestApplySetOverrides_EmptyKey(t *testing.T) {
	flux := map[string]any{}
	err := ApplySetOverrides(flux, []string{"=value"})
	if err == nil {
		t.Fatal("expected error for empty key")
	}
	if !strings.Contains(err.Error(), "key cannot be empty") {
		t.Errorf("expected empty key error, got: %v", err)
	}
}

func TestApplySetOverrides_ValueWithEquals(t *testing.T) {
	flux := map[string]any{}
	err := ApplySetOverrides(flux, []string{"key=a=b=c"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if flux["key"] != "a=b=c" {
		t.Errorf("expected key=a=b=c, got %v", flux["key"])
	}
}

// --- GetNestedAny tests ---

func TestGetNestedAny_String(t *testing.T) {
	m := map[string]any{"org": "acme"}
	val, ok := GetNestedAny(m, "org")
	if !ok {
		t.Fatal("expected key to be found")
	}
	if val != "acme" {
		t.Errorf("expected acme, got %v", val)
	}
}

func TestGetNestedAny_Bool(t *testing.T) {
	m := map[string]any{
		"ore": map[string]any{
			"status": map[string]any{"enabled": true},
		},
	}
	val, ok := GetNestedAny(m, "ore.status.enabled")
	if !ok {
		t.Fatal("expected key to be found")
	}
	if val != true {
		t.Errorf("expected true, got %v", val)
	}
}

func TestGetNestedAny_Map(t *testing.T) {
	m := map[string]any{
		"ore": map[string]any{
			"status": map[string]any{
				"options": map[string]any{
					"ready": map[string]any{"label": "Ready"},
				},
			},
		},
	}
	val, ok := GetNestedAny(m, "ore.status.options")
	if !ok {
		t.Fatal("expected key to be found")
	}
	opts, ok := val.(map[string]any)
	if !ok {
		t.Fatal("expected map")
	}
	ready, ok := opts["ready"].(map[string]any)
	if !ok {
		t.Fatal("expected ready map")
	}
	if ready["label"] != "Ready" {
		t.Errorf("expected Ready, got %v", ready["label"])
	}
}

func TestGetNestedAny_Missing(t *testing.T) {
	m := map[string]any{"org": "acme"}
	_, ok := GetNestedAny(m, "missing")
	if ok {
		t.Error("expected key to not be found")
	}
}

func TestGetNestedAny_PartialPath(t *testing.T) {
	m := map[string]any{
		"ore": map[string]any{},
	}
	_, ok := GetNestedAny(m, "ore.status.enabled")
	if ok {
		t.Error("expected partial path to not be found")
	}
}

// --- SetNestedAny tests ---

func TestSetNestedAny_TopLevel(t *testing.T) {
	m := map[string]any{}
	SetNestedAny(m, "org", "acme")
	if m["org"] != "acme" {
		t.Errorf("expected org=acme, got %v", m["org"])
	}
}

func TestSetNestedAny_Nested(t *testing.T) {
	m := map[string]any{}
	SetNestedAny(m, "ore.status.enabled", true)
	ore, ok := m["ore"].(map[string]any)
	if !ok {
		t.Fatal("expected ore to be a map")
	}
	status, ok := ore["status"].(map[string]any)
	if !ok {
		t.Fatal("expected status to be a map")
	}
	if status["enabled"] != true {
		t.Errorf("expected enabled=true, got %v", status["enabled"])
	}
}

func TestSetNestedAny_OverwriteExisting(t *testing.T) {
	m := map[string]any{
		"ore": map[string]any{
			"status": map[string]any{"enabled": false},
		},
	}
	SetNestedAny(m, "ore.status.enabled", true)
	ore := m["ore"].(map[string]any)
	status := ore["status"].(map[string]any)
	if status["enabled"] != true {
		t.Errorf("expected enabled=true after overwrite, got %v", status["enabled"])
	}
}

func TestSetNestedAny_MapValue(t *testing.T) {
	m := map[string]any{}
	opts := map[string]any{
		"ready": map[string]any{"label": "Ready", "id": "opt_1"},
	}
	SetNestedAny(m, "ore.status.options", opts)
	ore := m["ore"].(map[string]any)
	status := ore["status"].(map[string]any)
	statusOpts := status["options"].(map[string]any)
	ready := statusOpts["ready"].(map[string]any)
	if ready["label"] != "Ready" {
		t.Errorf("expected Ready, got %v", ready["label"])
	}
}
