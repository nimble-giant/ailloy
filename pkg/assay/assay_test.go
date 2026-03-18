package assay

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nimble-giant/ailloy/pkg/mold"
)

func TestAssay_NoFiles(t *testing.T) {
	dir := t.TempDir()
	result, err := Assay(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.FilesScanned != 0 {
		t.Errorf("expected 0 files scanned, got %d", result.FilesScanned)
	}
	if len(result.Diagnostics) != 0 {
		t.Errorf("expected 0 diagnostics, got %d", len(result.Diagnostics))
	}
}

func TestAssay_BasicClaude(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "CLAUDE.md", "# Instructions\n\nDo things correctly.\n")

	// Need .git dir for project root detection
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0755); err != nil {
		t.Fatal(err)
	}

	result, err := Assay(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.FilesScanned == 0 {
		t.Error("expected files to be scanned")
	}
}

func TestAssay_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "CLAUDE.md", "")
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0755); err != nil {
		t.Fatal(err)
	}

	result, err := Assay(dir, nil)
	if err != nil {
		t.Fatal(err)
	}

	hasEmptyWarning := false
	for _, d := range result.Diagnostics {
		if d.Rule == "empty-file" {
			hasEmptyWarning = true
		}
	}
	if !hasEmptyWarning {
		t.Error("expected empty-file warning")
	}
}

func TestAssay_LongFile(t *testing.T) {
	dir := t.TempDir()
	content := "# Instructions\n"
	for i := 0; i < 200; i++ {
		content += "- Rule number something\n"
	}
	writeFile(t, dir, "CLAUDE.md", content)
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0755); err != nil {
		t.Fatal(err)
	}

	result, err := Assay(dir, nil)
	if err != nil {
		t.Fatal(err)
	}

	hasLineCount := false
	for _, d := range result.Diagnostics {
		if d.Rule == "line-count" {
			hasLineCount = true
		}
	}
	if !hasLineCount {
		t.Error("expected line-count warning for file with 200+ lines")
	}
}

func TestAssay_DisabledRule(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "CLAUDE.md", "")
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0755); err != nil {
		t.Fatal(err)
	}

	disabled := false
	cfg := &Config{
		Rules: map[string]RuleConfig{
			"empty-file": {Enabled: &disabled},
		},
	}

	result, err := Assay(dir, cfg)
	if err != nil {
		t.Fatal(err)
	}

	for _, d := range result.Diagnostics {
		if d.Rule == "empty-file" {
			t.Error("expected empty-file rule to be disabled")
		}
	}
}

func TestAssay_PluginManifestInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0755); err != nil {
		t.Fatal(err)
	}
	// Plugin with broken manifest JSON
	writeFile(t, dir, filepath.Join("my-plugin", ".claude-plugin", "plugin.json"), `{invalid`)
	writeFile(t, dir, filepath.Join("my-plugin", "commands", "cmd.md"), "# Cmd\n")

	result, err := Assay(dir, nil)
	if err != nil {
		t.Fatal(err)
	}

	var found bool
	for _, d := range result.Diagnostics {
		if d.Rule == "plugin-manifest" {
			found = true
		}
	}
	if !found {
		t.Error("expected plugin-manifest diagnostic for invalid JSON")
	}
}

func TestAssay_PluginManifestMissingFields(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0755); err != nil {
		t.Fatal(err)
	}
	// Plugin manifest missing required fields
	writeFile(t, dir, filepath.Join("my-plugin", ".claude-plugin", "plugin.json"), `{"name":"only-name"}`)
	writeFile(t, dir, filepath.Join("my-plugin", "commands", "cmd.md"), "# Cmd\n")

	result, err := Assay(dir, nil)
	if err != nil {
		t.Fatal(err)
	}

	pluginDiags := 0
	for _, d := range result.Diagnostics {
		if d.Rule == "plugin-manifest" {
			pluginDiags++
		}
	}
	// Expects errors for missing "version" and "description"
	if pluginDiags < 2 {
		t.Errorf("expected at least 2 plugin-manifest diagnostics, got %d", pluginDiags)
	}
}

func TestAssay_MarketplaceMultiplePlugins(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0755); err != nil {
		t.Fatal(err)
	}
	manifest := `{"name":"p","version":"1.0.0","description":"A plugin"}`
	writeFile(t, dir, filepath.Join("plugins", "alpha", ".claude-plugin", "plugin.json"), manifest)
	writeFile(t, dir, filepath.Join("plugins", "alpha", "commands", "cmd.md"), "# Cmd\n")
	writeFile(t, dir, filepath.Join("plugins", "beta", ".claude-plugin", "plugin.json"), manifest)
	writeFile(t, dir, filepath.Join("plugins", "beta", "commands", "cmd.md"), "# Cmd\n")

	result, err := Assay(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	// Should have scanned at least 4 files (2 manifests + 2 commands)
	if result.FilesScanned < 4 {
		t.Errorf("expected at least 4 files scanned for marketplace, got %d", result.FilesScanned)
	}
}

func TestAssayResult_HasFailures(t *testing.T) {
	result := &AssayResult{
		Diagnostics: []mold.Diagnostic{
			{Severity: mold.SeverityWarning, Message: "warn"},
			{Severity: mold.SeveritySuggestion, Message: "suggest"},
		},
	}

	if result.HasFailures(mold.SeverityError) {
		t.Error("should not have failures at error level")
	}
	if !result.HasFailures(mold.SeverityWarning) {
		t.Error("should have failures at warning level")
	}
	if !result.HasFailures(mold.SeveritySuggestion) {
		t.Error("should have failures at suggestion level")
	}
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
