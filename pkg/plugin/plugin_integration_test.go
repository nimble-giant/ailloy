package plugin

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Integration tests for the full Claude plugin lifecycle

func TestIntegration_PluginGenerateValidateLifecycle(t *testing.T) {
	dir := t.TempDir()
	outputDir := filepath.Join(dir, "lifecycle-plugin")

	// Step 1: Generate a complete plugin
	g := NewGenerator(outputDir, testMoldReader())
	g.Config = &Config{
		Name:        "lifecycle-test",
		Version:     "1.0.0",
		Description: "Testing the full plugin lifecycle",
		Author: Author{
			Name:  "Test Team",
			Email: "test@example.com",
			URL:   "https://example.com",
		},
	}

	err := g.Generate()
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	// Step 2: Validate the generated plugin
	v := NewValidator(outputDir)
	result, err := v.Validate()
	if err != nil {
		t.Fatalf("validation failed: %v", err)
	}

	if !result.IsValid {
		t.Errorf("generated plugin should be valid, errors: %v", result.Errors)
	}
	if !result.HasManifest {
		t.Error("expected manifest")
	}
	if !result.HasCommands {
		t.Error("expected commands")
	}
	if !result.HasREADME {
		t.Error("expected README")
	}
	if result.CommandCount == 0 {
		t.Error("expected at least one command")
	}
	if len(result.Errors) > 0 {
		t.Errorf("unexpected errors: %v", result.Errors)
	}
}

func TestIntegration_PluginGenerateUpdateValidateLifecycle(t *testing.T) {
	dir := t.TempDir()
	outputDir := filepath.Join(dir, "update-lifecycle")

	// Step 1: Generate initial plugin
	g := NewGenerator(outputDir, testMoldReader())
	g.Config = &Config{
		Name:        "update-lifecycle",
		Version:     "1.0.0",
		Description: "Testing generate->update->validate lifecycle",
		Author:      Author{Name: "Test"},
	}

	if err := g.Generate(); err != nil {
		t.Fatalf("initial generation failed: %v", err)
	}

	// Step 2: Add a custom command
	customCmdPath := filepath.Join(outputDir, "commands", "my-custom-cmd.md")
	customContent := `# my-custom-cmd
description: A team-specific custom command

## Instructions for Claude

When this command is invoked, you must do custom things.
`
	if err := os.WriteFile(customCmdPath, []byte(customContent), 0644); err != nil {
		t.Fatalf("failed to write custom command: %v", err)
	}

	// Step 3: Update the plugin
	u := NewUpdater(outputDir, testMoldReader())

	// Backup first
	if err := u.Backup(); err != nil {
		t.Fatalf("backup failed: %v", err)
	}

	// Update
	if err := u.Update(); err != nil {
		t.Fatalf("update failed: %v", err)
	}

	// Step 4: Verify custom command preserved
	if _, err := os.Stat(customCmdPath); err != nil {
		t.Error("custom command should be preserved after update")
	}

	// Step 5: Validate updated plugin
	v := NewValidator(outputDir)
	result, err := v.Validate()
	if err != nil {
		t.Fatalf("validation failed: %v", err)
	}

	if !result.IsValid {
		t.Errorf("updated plugin should be valid, errors: %v", result.Errors)
	}

	// Should have standard commands + custom command
	if result.CommandCount <= 1 {
		t.Errorf("expected more than 1 command (standard + custom), got %d", result.CommandCount)
	}
}

func TestIntegration_PluginBackupRestoreCycle(t *testing.T) {
	dir := t.TempDir()
	outputDir := filepath.Join(dir, "backup-restore")

	// Generate plugin
	g := NewGenerator(outputDir, testMoldReader())
	g.Config = &Config{
		Name:        "backup-test",
		Version:     "1.0.0",
		Description: "Backup restore testing",
		Author:      Author{Name: "Test"},
	}
	if err := g.Generate(); err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	// Count original commands
	origEntries, _ := os.ReadDir(filepath.Join(outputDir, "commands"))
	origCount := len(origEntries)

	// Create backup
	u := NewUpdater(outputDir, testMoldReader())
	if err := u.Backup(); err != nil {
		t.Fatalf("backup failed: %v", err)
	}

	// Damage the plugin: delete all commands
	if err := os.RemoveAll(filepath.Join(outputDir, "commands")); err != nil {
		t.Fatalf("failed to remove commands dir: %v", err)
	}

	// Verify damage
	if _, err := os.Stat(filepath.Join(outputDir, "commands")); !os.IsNotExist(err) {
		t.Error("expected commands directory to be removed")
	}

	// Restore from backup
	if err := u.Restore(); err != nil {
		t.Fatalf("restore failed: %v", err)
	}

	// Verify restore
	restoredEntries, err := os.ReadDir(filepath.Join(outputDir, "commands"))
	if err != nil {
		t.Fatalf("failed to read restored commands: %v", err)
	}
	if len(restoredEntries) != origCount {
		t.Errorf("expected %d commands after restore, got %d", origCount, len(restoredEntries))
	}

	// Validate restored plugin
	v := NewValidator(outputDir)
	result, err := v.Validate()
	if err != nil {
		t.Fatalf("validation after restore failed: %v", err)
	}
	if !result.IsValid {
		t.Errorf("restored plugin should be valid, errors: %v", result.Errors)
	}
}

func TestIntegration_BlankTransformationConsistency(t *testing.T) {
	// Test that all embedded blanks can be loaded and transformed successfully
	g := NewGenerator(t.TempDir(), testMoldReader())
	if err := g.loadBlanks(); err != nil {
		t.Fatalf("failed to load blanks: %v", err)
	}

	tr := NewTransformer()

	for _, tmpl := range g.commands {
		t.Run(tmpl.Name, func(t *testing.T) {
			output, err := tr.Transform(tmpl)
			if err != nil {
				t.Fatalf("transform failed for %s: %v", tmpl.Name, err)
			}

			content := string(output)

			// Every transformed command must have these sections
			if !strings.Contains(content, "# "+tmpl.Name) {
				t.Errorf("missing command header for %s", tmpl.Name)
			}
			if !strings.Contains(content, "description:") {
				t.Errorf("missing description for %s", tmpl.Name)
			}
			if !strings.Contains(content, "## Instructions for Claude") {
				t.Errorf("missing instructions section for %s", tmpl.Name)
			}
		})
	}
}

func TestIntegration_ManifestSchemaConsistency(t *testing.T) {
	dir := t.TempDir()
	outputDir := filepath.Join(dir, "schema-test")

	g := NewGenerator(outputDir, testMoldReader())
	g.Config = &Config{
		Name:        "schema-test",
		Version:     "2.5.1",
		Description: "Schema consistency testing",
		Author: Author{
			Name:  "Schema Tester",
			Email: "schema@test.com",
			URL:   "https://schema.test",
		},
	}

	if err := g.createStructure(); err != nil {
		t.Fatalf("failed to create structure: %v", err)
	}
	if err := g.generateManifest(); err != nil {
		t.Fatalf("failed to generate manifest: %v", err)
	}

	// Read and parse manifest
	manifestPath := filepath.Join(outputDir, ".claude-plugin", "plugin.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("failed to read manifest: %v", err)
	}

	var manifest map[string]interface{}
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("invalid JSON manifest: %v", err)
	}

	// Verify all required fields per Claude Code plugin schema
	requiredFields := []string{"name", "version", "description", "author"}
	for _, field := range requiredFields {
		if _, ok := manifest[field]; !ok {
			t.Errorf("manifest missing required field: %s", field)
		}
	}

	// Verify field values
	if manifest["name"] != "schema-test" {
		t.Errorf("unexpected name: %v", manifest["name"])
	}
	if manifest["version"] != "2.5.1" {
		t.Errorf("unexpected version: %v", manifest["version"])
	}

	// Verify author is an object with name
	author, ok := manifest["author"].(map[string]interface{})
	if !ok {
		t.Fatal("author should be an object")
	}
	if author["name"] != "Schema Tester" {
		t.Errorf("unexpected author name: %v", author["name"])
	}
}

func TestIntegration_PluginDirectoryPermissions(t *testing.T) {
	dir := t.TempDir()
	outputDir := filepath.Join(dir, "perm-test")

	g := NewGenerator(outputDir, testMoldReader())
	g.Config = &Config{
		Name:        "perm-test",
		Version:     "1.0.0",
		Description: "Permission testing",
		Author:      Author{Name: "Test"},
	}

	if err := g.Generate(); err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	// Verify directory permissions
	dirsToCheck := []string{
		outputDir,
		filepath.Join(outputDir, ".claude-plugin"),
		filepath.Join(outputDir, "commands"),
		filepath.Join(outputDir, "hooks"),
		filepath.Join(outputDir, "scripts"),
	}

	for _, d := range dirsToCheck {
		info, err := os.Stat(d)
		if err != nil {
			t.Errorf("dir %s not found: %v", d, err)
			continue
		}
		perm := info.Mode().Perm()
		if perm&0700 != 0700 {
			t.Errorf("dir %s should be owner-rwx, got %o", d, perm)
		}
	}

	// Verify install script is executable
	scriptPath := filepath.Join(outputDir, "scripts", "install.sh")
	info, err := os.Stat(scriptPath)
	if err != nil {
		t.Fatalf("install script not found: %v", err)
	}
	if info.Mode()&0100 == 0 {
		t.Error("install script should be executable")
	}
}

func TestIntegration_MultipleGenerationsIdempotent(t *testing.T) {
	dir := t.TempDir()
	outputDir := filepath.Join(dir, "idempotent")

	cfg := &Config{
		Name:        "idempotent-test",
		Version:     "1.0.0",
		Description: "Testing idempotent generation",
		Author:      Author{Name: "Test"},
	}

	// Generate twice
	for i := 0; i < 2; i++ {
		g := NewGenerator(outputDir, testMoldReader())
		g.Config = cfg
		if err := g.Generate(); err != nil {
			t.Fatalf("generation %d failed: %v", i+1, err)
		}
	}

	// Validate after second generation
	v := NewValidator(outputDir)
	result, err := v.Validate()
	if err != nil {
		t.Fatalf("validation failed: %v", err)
	}
	if !result.IsValid {
		t.Errorf("plugin should be valid after re-generation, errors: %v", result.Errors)
	}
}
