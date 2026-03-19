package assay

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/nimble-giant/ailloy/pkg/mold"
)

func TestConsoleFormatter(t *testing.T) {
	result := &AssayResult{
		FilesScanned: 2,
		Diagnostics: []mold.Diagnostic{
			{Severity: mold.SeverityError, Message: "broken import", File: "CLAUDE.md", Rule: "import-validation"},
			{Severity: mold.SeverityWarning, Message: "too long", File: "AGENTS.md", Rule: "line-count"},
			{Severity: mold.SeveritySuggestion, Message: "add AGENTS.md", Rule: "agents-md-presence"},
		},
	}

	f := &ConsoleFormatter{}
	output := f.Format(result)

	if !strings.Contains(output, "ERROR") {
		t.Error("expected ERROR in output")
	}
	if !strings.Contains(output, "WARN") {
		t.Error("expected WARN in output")
	}
	if !strings.Contains(output, "HINT") {
		t.Error("expected HINT in output")
	}
}

func TestJSONFormatter(t *testing.T) {
	result := &AssayResult{
		FilesScanned: 1,
		Diagnostics: []mold.Diagnostic{
			{Severity: mold.SeverityError, Message: "broken", File: "test.md", Rule: "test-rule"},
		},
	}

	f := &JSONFormatter{}
	output := f.Format(result)

	var parsed jsonOutput
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}
	if parsed.FilesScanned != 1 {
		t.Errorf("expected 1 file scanned, got %d", parsed.FilesScanned)
	}
	if len(parsed.Diagnostics) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", len(parsed.Diagnostics))
	}
	if parsed.Diagnostics[0].Severity != "error" {
		t.Errorf("expected severity 'error', got %q", parsed.Diagnostics[0].Severity)
	}
}

func TestMarkdownFormatter(t *testing.T) {
	result := &AssayResult{
		FilesScanned: 2,
		Diagnostics: []mold.Diagnostic{
			{Severity: mold.SeverityError, Message: "broken", File: "test.md"},
			{Severity: mold.SeverityWarning, Message: "warn"},
		},
	}

	f := &MarkdownFormatter{}
	output := f.Format(result)

	if !strings.Contains(output, "# Assay Results") {
		t.Error("expected markdown header")
	}
	if !strings.Contains(output, "Errors") {
		t.Error("expected Errors section")
	}
	if !strings.Contains(output, "Warnings") {
		t.Error("expected Warnings section")
	}
}

func TestConsoleFormatter_TipDeduplication(t *testing.T) {
	result := &AssayResult{
		FilesScanned: 3,
		Diagnostics: []mold.Diagnostic{
			{Severity: mold.SeverityWarning, Message: "file a too long", File: "a.md", Rule: "line-count", Tip: "split into smaller files"},
			{Severity: mold.SeverityWarning, Message: "file b too long", File: "b.md", Rule: "line-count", Tip: "split into smaller files"},
			{Severity: mold.SeverityWarning, Message: "file c too long", File: "c.md", Rule: "line-count", Tip: "split into smaller files"},
		},
	}

	f := &ConsoleFormatter{}
	output := f.Format(result)

	// Tip should appear exactly once (after the last finding that has it)
	count := strings.Count(output, "split into smaller files")
	if count != 1 {
		t.Errorf("expected tip to appear once (deduplicated), got %d occurrences", count)
	}
}

func TestConsoleFormatter_TipAfterLastFinding(t *testing.T) {
	result := &AssayResult{
		FilesScanned: 3,
		Diagnostics: []mold.Diagnostic{
			{Severity: mold.SeverityWarning, Message: "unknown fields: created", File: "a.md", Rule: "command-frontmatter", Tip: "allow-fields created"},
			{Severity: mold.SeverityWarning, Message: "unknown fields: created", File: "b.md", Rule: "command-frontmatter", Tip: "allow-fields created"},
			{Severity: mold.SeverityWarning, Message: "unknown fields: version", File: "c.md", Rule: "command-frontmatter", Tip: "allow-fields version"},
		},
	}

	f := &ConsoleFormatter{}
	output := f.Format(result)

	// "allow-fields created" shown once, after b.md (last finding with that tip)
	if c := strings.Count(output, "allow-fields created"); c != 1 {
		t.Errorf("expected deduplicated tip once, got %d", c)
	}
	// "allow-fields version" shown once, after c.md
	if c := strings.Count(output, "allow-fields version"); c != 1 {
		t.Errorf("expected unique tip once, got %d", c)
	}

	// Tip for "created" should appear after b.md but before c.md
	bIdx := strings.Index(output, "b.md")
	createdIdx := strings.Index(output, "allow-fields created")
	cIdx := strings.Index(output, "c.md")
	if createdIdx < bIdx || createdIdx > cIdx {
		t.Error("expected 'allow-fields created' tip to appear between b.md and c.md findings")
	}
}

func TestConsoleFormatter_DistinctTipsBothShown(t *testing.T) {
	result := &AssayResult{
		FilesScanned: 2,
		Diagnostics: []mold.Diagnostic{
			{Severity: mold.SeverityError, Message: "name too long", File: "a.yml", Rule: "name-format", Tip: "shorten the name"},
			{Severity: mold.SeverityError, Message: "name has hyphens", File: "b.yml", Rule: "name-format", Tip: "fix hyphen usage"},
		},
	}

	f := &ConsoleFormatter{}
	output := f.Format(result)

	if !strings.Contains(output, "shorten the name") {
		t.Error("expected first tip in output")
	}
	if !strings.Contains(output, "fix hyphen usage") {
		t.Error("expected second tip in output")
	}
}

func TestConsoleFormatter_FileHyperlinks(t *testing.T) {
	result := &AssayResult{
		FilesScanned: 1,
		Diagnostics: []mold.Diagnostic{
			{Severity: mold.SeverityWarning, Message: "too long", File: "skills/my-skill/SKILL.md", Rule: "line-count"},
		},
	}

	// With WorkDir set, output should contain OSC 8 hyperlink escape sequences
	f := &ConsoleFormatter{WorkDir: "/home/user/project"}
	output := f.Format(result)
	if !strings.Contains(output, "\033]8;;file:///home/user/project/skills/my-skill/SKILL.md\033\\") {
		t.Error("expected OSC 8 hyperlink with file:// URL in output")
	}
	if !strings.Contains(output, "\033]8;;\033\\") {
		t.Error("expected OSC 8 closing sequence in output")
	}

	// Without WorkDir, no hyperlinks — just plain text
	f2 := &ConsoleFormatter{}
	output2 := f2.Format(result)
	if strings.Contains(output2, "\033]8;;") {
		t.Error("expected no hyperlink sequences when WorkDir is empty")
	}
	if !strings.Contains(output2, "skills/my-skill/SKILL.md") {
		t.Error("expected plain file path in output")
	}
}

func TestNewFormatter(t *testing.T) {
	tests := []struct {
		format string
		typ    string
	}{
		{"console", "*assay.ConsoleFormatter"},
		{"json", "*assay.JSONFormatter"},
		{"markdown", "*assay.MarkdownFormatter"},
		{"unknown", "*assay.ConsoleFormatter"},
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			f := NewFormatter(tt.format, "")
			if f == nil {
				t.Fatal("expected non-nil formatter")
			}
		})
	}
}
