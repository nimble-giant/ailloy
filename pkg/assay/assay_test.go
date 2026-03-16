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
