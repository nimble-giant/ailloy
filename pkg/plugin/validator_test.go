package plugin

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func setupValidPlugin(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// Create directory structure
	dirs := []string{
		filepath.Join(dir, ".claude-plugin"),
		filepath.Join(dir, "commands"),
		filepath.Join(dir, "hooks"),
		filepath.Join(dir, "scripts"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0750); err != nil {
			t.Fatalf("failed to create dir %s: %v", d, err)
		}
	}

	// Create manifest
	manifest := map[string]interface{}{
		"name":        "test-plugin",
		"version":     "1.0.0",
		"description": "A test plugin",
		"author": map[string]string{
			"name": "Test Author",
		},
	}
	manifestData, _ := json.MarshalIndent(manifest, "", "  ")
	if err := os.WriteFile(filepath.Join(dir, ".claude-plugin", "plugin.json"), manifestData, 0644); err != nil {
		t.Fatalf("failed to write manifest: %v", err)
	}

	// Create a command file
	cmdContent := `# test-command
description: A test command

## Instructions for Claude

When this command is invoked, you must:

1. Do the thing
`
	if err := os.WriteFile(filepath.Join(dir, "commands", "test-command.md"), []byte(cmdContent), 0644); err != nil {
		t.Fatalf("failed to write command: %v", err)
	}

	// Create README
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Test Plugin"), 0644); err != nil {
		t.Fatalf("failed to write README: %v", err)
	}

	// Create hooks
	hooks := map[string]interface{}{"hooks": []interface{}{}}
	hooksData, _ := json.MarshalIndent(hooks, "", "  ")
	if err := os.WriteFile(filepath.Join(dir, "hooks", "hooks.json"), hooksData, 0644); err != nil {
		t.Fatalf("failed to write hooks: %v", err)
	}

	// Create install script
	if err := os.WriteFile(filepath.Join(dir, "scripts", "install.sh"), []byte("#!/bin/bash\necho hello"), 0750); err != nil {
		t.Fatalf("failed to write install script: %v", err)
	}

	return dir
}

func TestNewValidator(t *testing.T) {
	v := NewValidator("/some/path")
	if v == nil {
		t.Fatal("expected non-nil validator")
	}
	if v.PluginPath != "/some/path" {
		t.Errorf("expected path '/some/path', got '%s'", v.PluginPath)
	}
}

func TestValidator_ValidPlugin(t *testing.T) {
	dir := setupValidPlugin(t)
	v := NewValidator(dir)

	result, err := v.Validate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.IsValid {
		t.Errorf("expected valid plugin, got errors: %v", result.Errors)
	}
	if !result.HasManifest {
		t.Error("expected HasManifest to be true")
	}
	if !result.HasCommands {
		t.Error("expected HasCommands to be true")
	}
	if !result.HasREADME {
		t.Error("expected HasREADME to be true")
	}
	if result.CommandCount != 1 {
		t.Errorf("expected 1 command, got %d", result.CommandCount)
	}
}

func TestValidator_NonExistentPlugin(t *testing.T) {
	v := NewValidator("/nonexistent/path")
	result, err := v.Validate()
	if err == nil {
		t.Error("expected error for non-existent plugin")
	}
	if result.IsValid {
		t.Error("expected IsValid to be false")
	}
}

func TestValidator_MissingManifest(t *testing.T) {
	dir := setupValidPlugin(t)
	// Remove manifest
	if err := os.Remove(filepath.Join(dir, ".claude-plugin", "plugin.json")); err != nil {
		t.Fatalf("failed to remove manifest: %v", err)
	}

	v := NewValidator(dir)
	result, err := v.Validate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsValid {
		t.Error("expected IsValid to be false with missing manifest")
	}
	if result.HasManifest {
		t.Error("expected HasManifest to be false")
	}
}

func TestValidator_InvalidManifestJSON(t *testing.T) {
	dir := setupValidPlugin(t)
	// Write invalid JSON
	if err := os.WriteFile(filepath.Join(dir, ".claude-plugin", "plugin.json"), []byte("{invalid}"), 0644); err != nil {
		t.Fatalf("failed to write invalid manifest: %v", err)
	}

	v := NewValidator(dir)
	result, err := v.Validate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsValid {
		t.Error("expected IsValid to be false with invalid JSON manifest")
	}
}

func TestValidator_ManifestMissingFields(t *testing.T) {
	dir := setupValidPlugin(t)
	// Write manifest missing required fields
	manifest := map[string]interface{}{
		"name": "test-plugin",
		// missing version and description
	}
	manifestData, _ := json.MarshalIndent(manifest, "", "  ")
	if err := os.WriteFile(filepath.Join(dir, ".claude-plugin", "plugin.json"), manifestData, 0644); err != nil {
		t.Fatalf("failed to write manifest: %v", err)
	}

	v := NewValidator(dir)
	result, err := v.Validate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsValid {
		t.Error("expected IsValid to be false with missing required fields")
	}

	// Should have errors for version and description
	foundVersion := false
	foundDesc := false
	for _, e := range result.Errors {
		if contains(e, "version") {
			foundVersion = true
		}
		if contains(e, "description") {
			foundDesc = true
		}
	}
	if !foundVersion {
		t.Error("expected error about missing version")
	}
	if !foundDesc {
		t.Error("expected error about missing description")
	}
}

func TestValidator_NoCommands(t *testing.T) {
	dir := setupValidPlugin(t)
	// Remove all command files
	if err := os.RemoveAll(filepath.Join(dir, "commands")); err != nil {
		t.Fatalf("failed to remove commands dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "commands"), 0750); err != nil {
		t.Fatalf("failed to recreate commands dir: %v", err)
	}

	v := NewValidator(dir)
	result, err := v.Validate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsValid {
		t.Error("expected IsValid to be false with no commands")
	}
	if result.HasCommands {
		t.Error("expected HasCommands to be false")
	}
}

func TestValidator_MissingREADME(t *testing.T) {
	dir := setupValidPlugin(t)
	if err := os.Remove(filepath.Join(dir, "README.md")); err != nil {
		t.Fatalf("failed to remove README: %v", err)
	}

	v := NewValidator(dir)
	result, err := v.Validate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.HasREADME {
		t.Error("expected HasREADME to be false")
	}
	// Missing README should be a warning, not an error
	foundWarning := false
	for _, w := range result.Warnings {
		if contains(w, "README") {
			foundWarning = true
		}
	}
	if !foundWarning {
		t.Error("expected warning about missing README")
	}
}

func TestValidator_MissingInstallScript(t *testing.T) {
	dir := setupValidPlugin(t)
	if err := os.Remove(filepath.Join(dir, "scripts", "install.sh")); err != nil {
		t.Fatalf("failed to remove install script: %v", err)
	}

	v := NewValidator(dir)
	result, err := v.Validate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	foundWarning := false
	for _, w := range result.Warnings {
		if contains(w, "install.sh") {
			foundWarning = true
		}
	}
	if !foundWarning {
		t.Error("expected warning about missing install script")
	}
}

func TestValidator_InvalidHooksJSON(t *testing.T) {
	dir := setupValidPlugin(t)
	if err := os.WriteFile(filepath.Join(dir, "hooks", "hooks.json"), []byte("{invalid"), 0644); err != nil {
		t.Fatalf("failed to write invalid hooks: %v", err)
	}

	v := NewValidator(dir)
	result, err := v.Validate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	foundWarning := false
	for _, w := range result.Warnings {
		if contains(w, "hooks.json") {
			foundWarning = true
		}
	}
	if !foundWarning {
		t.Error("expected warning about invalid hooks.json")
	}
}

func TestValidator_CommandMissingHeader(t *testing.T) {
	dir := setupValidPlugin(t)
	// Write a command without proper header
	if err := os.WriteFile(filepath.Join(dir, "commands", "bad-cmd.md"), []byte("No header here"), 0644); err != nil {
		t.Fatalf("failed to write command: %v", err)
	}

	v := NewValidator(dir)
	result, err := v.Validate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	foundWarning := false
	for _, w := range result.Warnings {
		if contains(w, "missing proper header") {
			foundWarning = true
		}
	}
	if !foundWarning {
		t.Error("expected warning about missing command header")
	}
}

func TestIsValidVersion(t *testing.T) {
	tests := []struct {
		version string
		valid   bool
	}{
		{"1.0.0", true},
		{"0.1.0", true},
		{"10.20.30", true},
		{"", false},
		{"abc", false},
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			result := isValidVersion(tt.version)
			if result != tt.valid {
				t.Errorf("isValidVersion(%q) = %v, want %v", tt.version, result, tt.valid)
			}
		})
	}
}

func TestHasCommandHeader(t *testing.T) {
	tests := []struct {
		content  string
		expected bool
	}{
		{"# Header", true},
		// hasCommandHeader checks content[0] == '#', so "##" also matches
		{"## Sub-header", true},
		{"No header", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.content, func(t *testing.T) {
			result := hasCommandHeader(tt.content)
			if result != tt.expected {
				t.Errorf("hasCommandHeader(%q) = %v, want %v", tt.content, result, tt.expected)
			}
		})
	}
}

func TestHasDescription(t *testing.T) {
	tests := []struct {
		content  string
		expected bool
	}{
		{"description: A command", true},
		{"Some text\ndescription: here\nmore text", true},
		{"no description here", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.content, func(t *testing.T) {
			result := hasDescription(tt.content)
			if result != tt.expected {
				t.Errorf("hasDescription(%q) = %v, want %v", tt.content, result, tt.expected)
			}
		})
	}
}

func TestHasInstructions(t *testing.T) {
	tests := []struct {
		content  string
		expected bool
	}{
		{"## Instructions for Claude", true},
		{"When this command is invoked", true},
		{"no instructions", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.content, func(t *testing.T) {
			result := hasInstructions(tt.content)
			if result != tt.expected {
				t.Errorf("hasInstructions(%q) = %v, want %v", tt.content, result, tt.expected)
			}
		})
	}
}

func TestContainsPattern(t *testing.T) {
	tests := []struct {
		content  string
		pattern  string
		expected bool
	}{
		{"hello world", "world", true},
		{"hello world", "xyz", false},
		{"short", "very long pattern that exceeds content", false},
		{"", "a", false},
	}

	for _, tt := range tests {
		t.Run(tt.content+"/"+tt.pattern, func(t *testing.T) {
			result := containsPattern(tt.content, tt.pattern)
			if result != tt.expected {
				t.Errorf("containsPattern(%q, %q) = %v, want %v", tt.content, tt.pattern, result, tt.expected)
			}
		})
	}
}

func TestFindSubstring(t *testing.T) {
	tests := []struct {
		s      string
		substr string
		want   int
	}{
		{"hello world", "world", 6},
		{"hello world", "hello", 0},
		{"hello world", "xyz", -1},
		{"aaa", "a", 0},
		{"", "a", -1},
	}

	for _, tt := range tests {
		t.Run(tt.s+"/"+tt.substr, func(t *testing.T) {
			got := findSubstring(tt.s, tt.substr)
			if got != tt.want {
				t.Errorf("findSubstring(%q, %q) = %d, want %d", tt.s, tt.substr, got, tt.want)
			}
		})
	}
}

func contains(s, substr string) bool {
	return findSubstring(s, substr) != -1
}
