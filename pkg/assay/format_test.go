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

	if !strings.Contains(output, "ERROR:") {
		t.Error("expected ERROR: in output")
	}
	if !strings.Contains(output, "WARNING:") {
		t.Error("expected WARNING: in output")
	}
	if !strings.Contains(output, "SUGGESTION:") {
		t.Error("expected SUGGESTION: in output")
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
			f := NewFormatter(tt.format)
			if f == nil {
				t.Fatal("expected non-nil formatter")
			}
		})
	}
}
