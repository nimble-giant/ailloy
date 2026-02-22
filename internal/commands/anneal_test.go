package commands

import (
	"os"
	"testing"

	"github.com/nimble-giant/ailloy/pkg/mold"
)

func TestInferSchemaFromFlux_FlatMap(t *testing.T) {
	flux := map[string]any{
		"org": "acme",
	}

	schema := inferSchemaFromFlux(flux)

	if len(schema) != 1 {
		t.Fatalf("expected 1 var, got %d", len(schema))
	}
	if schema[0].Name != "org" {
		t.Errorf("expected name 'org', got %q", schema[0].Name)
	}
	if schema[0].Type != "string" {
		t.Errorf("expected type 'string', got %q", schema[0].Type)
	}
}

func TestInferSchemaFromFlux_NestedMap(t *testing.T) {
	flux := map[string]any{
		"project": map[string]any{
			"board": "Engineering",
			"org":   "acme",
		},
	}

	schema := inferSchemaFromFlux(flux)

	if len(schema) != 2 {
		t.Fatalf("expected 2 vars, got %d", len(schema))
	}
	// Sorted by name
	if schema[0].Name != "project.board" {
		t.Errorf("expected first var 'project.board', got %q", schema[0].Name)
	}
	if schema[1].Name != "project.org" {
		t.Errorf("expected second var 'project.org', got %q", schema[1].Name)
	}
}

func TestInferSchemaFromFlux_BoolValue(t *testing.T) {
	flux := map[string]any{
		"enabled": true,
	}

	schema := inferSchemaFromFlux(flux)

	if len(schema) != 1 {
		t.Fatalf("expected 1 var, got %d", len(schema))
	}
	if schema[0].Type != "bool" {
		t.Errorf("expected type 'bool', got %q", schema[0].Type)
	}
}

func TestInferSchemaFromFlux_IntValue(t *testing.T) {
	flux := map[string]any{
		"count": 42,
	}

	schema := inferSchemaFromFlux(flux)

	if len(schema) != 1 {
		t.Fatalf("expected 1 var, got %d", len(schema))
	}
	if schema[0].Type != "int" {
		t.Errorf("expected type 'int', got %q", schema[0].Type)
	}
}

func TestInferSchemaFromFlux_EmptyMap(t *testing.T) {
	schema := inferSchemaFromFlux(map[string]any{})
	if len(schema) != 0 {
		t.Errorf("expected 0 vars, got %d", len(schema))
	}
}

func TestResolveAnnealSchema_WithFluxSchema(t *testing.T) {
	// This is an integration-style test using the remote nimble-mold.
	// We chdir to a tmpDir so the foundry lock file doesn't pollute the repo.
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	reader := testMoldReader(t)

	schema, fluxDefaults, err := resolveAnnealSchema(reader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(schema) == 0 {
		t.Fatal("expected non-empty schema from nimble-mold")
	}

	// Check that we got a schema with expected variables
	var hasOrg, hasProjectID bool
	for _, fv := range schema {
		switch fv.Name {
		case "project.organization":
			hasOrg = true
			if !fv.Required {
				t.Error("expected project.organization to be required")
			}
		case "project.id":
			hasProjectID = true
			if fv.Discover == nil {
				t.Error("expected project.id to have discover spec")
			}
		}
	}
	if !hasOrg {
		t.Error("expected project.organization in schema")
	}
	if !hasProjectID {
		t.Error("expected project.id in schema")
	}

	// Flux defaults should also be loaded
	if len(fluxDefaults) == 0 {
		t.Error("expected non-empty flux defaults")
	}
}

func TestValidateFlux_SelectType(t *testing.T) {
	schema := []mold.FluxVar{
		{Name: "provider", Type: "select", Options: []mold.SelectOption{
			{Label: "GitHub", Value: "github"},
			{Label: "GitLab", Value: "gitlab"},
		}},
	}

	flux := map[string]any{"provider": "github"}
	err := mold.ValidateFlux(schema, flux)
	if err != nil {
		t.Errorf("expected no error for valid select value, got: %v", err)
	}
}
