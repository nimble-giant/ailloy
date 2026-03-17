package assay

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/nimble-giant/ailloy/pkg/mold"
	"github.com/nimble-giant/ailloy/pkg/styles"
)

// Formatter controls how assay results are rendered.
type Formatter interface {
	Format(result *AssayResult) string
}

// NewFormatter returns a formatter for the given format name.
func NewFormatter(format string) Formatter {
	switch format {
	case "json":
		return &JSONFormatter{}
	case "markdown":
		return &MarkdownFormatter{}
	default:
		return &ConsoleFormatter{}
	}
}

// ConsoleFormatter renders diagnostics with styled terminal output.
type ConsoleFormatter struct{}

func (f *ConsoleFormatter) Format(result *AssayResult) string {
	var b strings.Builder

	for _, d := range result.Diagnostics {
		loc := ""
		if d.File != "" {
			loc = styles.SubtleStyle.Render(d.File + ": ") //nolint:goconst
		}

		switch d.Severity {
		case mold.SeverityError:
			b.WriteString(styles.ErrorStyle.Render("ERROR: ") + loc + d.Message)
		case mold.SeverityWarning:
			b.WriteString(styles.WarningStyle.Render("WARNING: ") + loc + d.Message)
		case mold.SeveritySuggestion:
			b.WriteString(styles.InfoStyle.Render("SUGGESTION: ") + loc + d.Message)
		}
		b.WriteString("\n")
	}

	return b.String()
}

// JSONFormatter renders diagnostics as JSON.
type JSONFormatter struct{}

type jsonDiagnostic struct {
	Severity string `json:"severity"`
	Message  string `json:"message"`
	File     string `json:"file,omitempty"`
	Rule     string `json:"rule,omitempty"`
}

type jsonOutput struct {
	FilesScanned int              `json:"files_scanned"`
	Diagnostics  []jsonDiagnostic `json:"diagnostics"`
}

func (f *JSONFormatter) Format(result *AssayResult) string {
	out := jsonOutput{
		FilesScanned: result.FilesScanned,
		Diagnostics:  make([]jsonDiagnostic, 0, len(result.Diagnostics)),
	}
	for _, d := range result.Diagnostics {
		out.Diagnostics = append(out.Diagnostics, jsonDiagnostic{
			Severity: d.Severity.String(),
			Message:  d.Message,
			File:     d.File,
			Rule:     d.Rule,
		})
	}
	data, _ := json.MarshalIndent(out, "", "  ")
	return string(data) + "\n"
}

// MarkdownFormatter renders diagnostics as markdown.
type MarkdownFormatter struct{}

func (f *MarkdownFormatter) Format(result *AssayResult) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("# Assay Results\n\n**Files scanned:** %d\n\n", result.FilesScanned))

	groups := map[mold.DiagSeverity][]mold.Diagnostic{
		mold.SeverityError:      result.Errors(),
		mold.SeverityWarning:    result.Warnings(),
		mold.SeveritySuggestion: result.Suggestions(),
	}

	labels := []struct {
		sev   mold.DiagSeverity
		emoji string
		title string
	}{
		{mold.SeverityError, ":x:", "Errors"},
		{mold.SeverityWarning, ":warning:", "Warnings"},
		{mold.SeveritySuggestion, ":bulb:", "Suggestions"},
	}

	for _, l := range labels {
		diags := groups[l.sev]
		if len(diags) == 0 {
			continue
		}
		b.WriteString(fmt.Sprintf("## %s %s (%d)\n\n", l.emoji, l.title, len(diags)))
		for _, d := range diags {
			file := ""
			if d.File != "" {
				file = fmt.Sprintf("`%s`: ", d.File)
			}
			b.WriteString(fmt.Sprintf("- %s%s\n", file, d.Message))
		}
		b.WriteString("\n")
	}

	return b.String()
}
