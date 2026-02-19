package mold

import (
	"strings"
	"testing"
)

// --- ApplyFluxDefaults tests ---

func TestApplyFluxDefaults_AppliesMissingDefaults(t *testing.T) {
	schema := []FluxVar{
		{Name: "board", Type: "string", Default: "Engineering"},
		{Name: "org", Type: "string", Required: true},
	}
	flux := map[string]string{"org": "acme"}

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
	flux := map[string]string{"board": "Product"}

	result := ApplyFluxDefaults(schema, flux)

	if result["board"] != "Product" {
		t.Errorf("expected board=Product (not overridden), got %q", result["board"])
	}
}

func TestApplyFluxDefaults_DoesNotMutateInput(t *testing.T) {
	schema := []FluxVar{
		{Name: "board", Type: "string", Default: "Engineering"},
	}
	flux := map[string]string{}

	result := ApplyFluxDefaults(schema, flux)

	if _, exists := flux["board"]; exists {
		t.Error("input map should not be mutated")
	}
	if result["board"] != "Engineering" {
		t.Errorf("expected board=Engineering in result, got %q", result["board"])
	}
}

func TestApplyFluxDefaults_EmptySchema(t *testing.T) {
	flux := map[string]string{"key": "val"}
	result := ApplyFluxDefaults(nil, flux)

	if result["key"] != "val" {
		t.Errorf("expected key=val, got %q", result["key"])
	}
}

func TestApplyFluxDefaults_SkipsEmptyDefault(t *testing.T) {
	schema := []FluxVar{
		{Name: "org", Type: "string", Required: true},
	}
	flux := map[string]string{}

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
	err := ValidateFlux(schema, map[string]string{})
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
	err := ValidateFlux(schema, map[string]string{"org": ""})
	if err == nil {
		t.Fatal("expected error for empty required var")
	}
}

func TestValidateFlux_RequiredPresent(t *testing.T) {
	schema := []FluxVar{
		{Name: "org", Type: "string", Required: true},
	}
	if err := ValidateFlux(schema, map[string]string{"org": "acme"}); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateFlux_OptionalMissing(t *testing.T) {
	schema := []FluxVar{
		{Name: "board", Type: "string", Required: false},
	}
	if err := ValidateFlux(schema, map[string]string{}); err != nil {
		t.Errorf("unexpected error for missing optional var: %v", err)
	}
}

func TestValidateFlux_BoolValid(t *testing.T) {
	schema := []FluxVar{
		{Name: "enabled", Type: "bool", Required: true},
	}
	for _, val := range []string{"true", "false", "True", "FALSE"} {
		if err := ValidateFlux(schema, map[string]string{"enabled": val}); err != nil {
			t.Errorf("expected %q to be valid bool, got: %v", val, err)
		}
	}
}

func TestValidateFlux_BoolInvalid(t *testing.T) {
	schema := []FluxVar{
		{Name: "enabled", Type: "bool", Required: true},
	}
	err := ValidateFlux(schema, map[string]string{"enabled": "yes"})
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
		if err := ValidateFlux(schema, map[string]string{"count": val}); err != nil {
			t.Errorf("expected %q to be valid int, got: %v", val, err)
		}
	}
}

func TestValidateFlux_IntInvalid(t *testing.T) {
	schema := []FluxVar{
		{Name: "count", Type: "int", Required: true},
	}
	err := ValidateFlux(schema, map[string]string{"count": "abc"})
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
		if err := ValidateFlux(schema, map[string]string{"tags": val}); err != nil {
			t.Errorf("expected %q to be valid list, got: %v", val, err)
		}
	}
}

func TestValidateFlux_StringAlwaysValid(t *testing.T) {
	schema := []FluxVar{
		{Name: "name", Type: "string", Required: true},
	}
	for _, val := range []string{"hello", "123", "true", "a,b,c", "anything goes"} {
		if err := ValidateFlux(schema, map[string]string{"name": val}); err != nil {
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
	err := ValidateFlux(schema, map[string]string{})
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
	if err := ValidateFlux(nil, map[string]string{"extra": "val"}); err != nil {
		t.Errorf("expected no error for empty schema, got: %v", err)
	}
}

func TestValidateFlux_TypeAndRequiredCombined(t *testing.T) {
	schema := []FluxVar{
		{Name: "count", Type: "int", Required: true},
	}
	// Provided but wrong type
	err := ValidateFlux(schema, map[string]string{"count": "not-a-number"})
	if err == nil {
		t.Fatal("expected error for wrong type")
	}
	if !strings.Contains(err.Error(), "must be an int") {
		t.Errorf("expected type error, got: %v", err)
	}
}
