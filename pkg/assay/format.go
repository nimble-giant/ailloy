package assay

import (
	"encoding/json"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/nimble-giant/ailloy/pkg/mold"
	"github.com/nimble-giant/ailloy/pkg/styles"
)

// Formatter controls how assay results are rendered.
type Formatter interface {
	Format(result *AssayResult) string
}

// NewFormatter returns a formatter for the given format name.
// workDir is used to resolve relative file paths for clickable terminal hyperlinks.
func NewFormatter(format string, workDir string) Formatter {
	switch format {
	case "json":
		return &JSONFormatter{}
	case "markdown":
		return &MarkdownFormatter{}
	default:
		return &ConsoleFormatter{WorkDir: workDir}
	}
}

// contextGroupStat holds rollup token stats for a plugin or project group.
type contextGroupStat struct {
	Name            string
	EstimatedTokens int
	FileCount       int
}

// contextSummary computes rollup totals from per-file stats.
// Returns per-plugin groups (if any plugins exist) and the overall total.
func contextSummary(stats []FileContextStat) (groups []contextGroupStat, total int) {
	pluginTokens := map[string]int{} // pluginDir -> tokens
	pluginFiles := map[string]int{}  // pluginDir -> file count
	var pluginOrder []string
	projectTokens := 0
	projectFiles := 0

	for _, cs := range stats {
		total += cs.EstimatedTokens
		if cs.PluginDir != "" {
			if _, seen := pluginTokens[cs.PluginDir]; !seen {
				pluginOrder = append(pluginOrder, cs.PluginDir)
			}
			pluginTokens[cs.PluginDir] += cs.EstimatedTokens
			pluginFiles[cs.PluginDir]++
		} else {
			projectTokens += cs.EstimatedTokens
			projectFiles++
		}
	}

	if projectFiles > 0 {
		groups = append(groups, contextGroupStat{
			Name:            "project",
			EstimatedTokens: projectTokens,
			FileCount:       projectFiles,
		})
	}
	for _, pd := range pluginOrder {
		groups = append(groups, contextGroupStat{
			Name:            pd,
			EstimatedTokens: pluginTokens[pd],
			FileCount:       pluginFiles[pd],
		})
	}
	return groups, total
}

// ConsoleFormatter renders diagnostics grouped by rule with educational rationale headers.
type ConsoleFormatter struct {
	// WorkDir is used to resolve relative file paths into clickable file:// hyperlinks.
	WorkDir string
}

var (
	tipStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Italic(true)
	rationaleStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#aaaaaa")).Italic(true)
	ruleHeaderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#9b72cf")).Bold(true)
	separatorStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#444444"))
)

// diagGroup preserves insertion order while grouping diagnostics by rule.
type diagGroup struct {
	rule  string
	diags []mold.Diagnostic
}

func (f *ConsoleFormatter) Format(result *AssayResult) string {
	// Group diagnostics by rule, preserving first-seen order.
	var groups []diagGroup
	index := map[string]int{}
	for _, d := range result.Diagnostics {
		key := d.Rule
		if key == "" {
			key = "_"
		}
		if i, ok := index[key]; ok {
			groups[i].diags = append(groups[i].diags, d)
		} else {
			index[key] = len(groups)
			groups = append(groups, diagGroup{rule: key, diags: []mold.Diagnostic{d}})
		}
	}

	var b strings.Builder
	for _, g := range groups {
		// ── rule-name ──────────────────────────────────────────
		ruleName := g.rule
		if ruleName == "_" {
			ruleName = "general"
		}
		header := ruleHeaderStyle.Render(ruleName)
		sep := separatorStyle.Render(strings.Repeat("─", max(0, 72-len(ruleName)-4)))
		b.WriteString("── " + header + " " + sep + "\n")

		// Rationale (if defined)
		if r := RuleRationale(g.rule); r != "" {
			b.WriteString(rationaleStyle.Render("   "+r) + "\n")
		}
		b.WriteString("\n")

		// Track tip positions: first and last occurrence index per tip.
		tipFirstIdx := map[string]int{}
		tipLastIdx := map[string]int{}
		for i, d := range g.diags {
			if d.Tip != "" {
				if _, seen := tipFirstIdx[d.Tip]; !seen {
					tipFirstIdx[d.Tip] = i
				}
				tipLastIdx[d.Tip] = i
			}
		}

		// Diagnostics
		for i, d := range g.diags {
			var badge string
			switch d.Severity {
			case mold.SeverityError:
				badge = styles.ErrorStyle.Render("  ERROR  ")
			case mold.SeverityWarning:
				badge = styles.WarningStyle.Render("  WARN   ")
			case mold.SeveritySuggestion:
				badge = styles.InfoStyle.Render("  HINT   ")
			}

			if d.File != "" {
				fileText := styles.SubtleStyle.Render(d.File)
				b.WriteString("   " + badge + "  " + f.fileHyperlink(d.File, fileText) + "\n")
			} else {
				b.WriteString("   " + badge + "\n")
			}
			b.WriteString("            " + d.Message + "\n")

			// Show tip after the last finding that has it.
			// If the tip spans multiple findings, add a blank line before it
			// so it visually separates from the last finding and reads as
			// applying to the whole group.
			if d.Tip != "" && tipLastIdx[d.Tip] == i {
				isGrouped := tipFirstIdx[d.Tip] != tipLastIdx[d.Tip]
				if isGrouped {
					b.WriteString("\n")
				}
				b.WriteString("            " + tipStyle.Render("💡 "+d.Tip) + "\n")
			}
			b.WriteString("\n")
		}

		// Context usage summary: appended after context-usage diagnostics
		if g.rule == "context-usage" && len(result.ContextStats) > 0 {
			f.writeContextSummary(&b, result)
		}
	}

	// If there are context stats but no context-usage diagnostics were generated
	// (all files within thresholds), render the summary section standalone.
	if len(result.ContextStats) > 0 && !result.HasContextFindings() {
		ruleName := "context-usage"
		header := ruleHeaderStyle.Render(ruleName)
		sep := separatorStyle.Render(strings.Repeat("─", max(0, 72-len(ruleName)-4)))
		b.WriteString("── " + header + " " + sep + "\n")
		if r := RuleRationale(ruleName); r != "" {
			b.WriteString(rationaleStyle.Render("   "+r) + "\n")
		}
		b.WriteString("\n")
		f.writeContextSummary(&b, result)
	}

	return b.String()
}

// writeContextSummary appends the context usage rollup summary after context-usage diagnostics.
// With --verbose, it also includes per-file detail for files that didn't trigger diagnostics.
func (f *ConsoleFormatter) writeContextSummary(b *strings.Builder, result *AssayResult) {
	summary, total := contextSummary(result.ContextStats)

	// Per-file detail: only with --verbose
	if result.Verbose {
		for _, cs := range result.ContextStats {
			pct := float64(cs.EstimatedTokens) * 100 / float64(result.ContextWindow)
			var label string
			if cs.ImportCount == 0 {
				label = fmt.Sprintf("~%d tokens (%.1f%%)", cs.EstimatedTokens, pct)
			} else {
				label = fmt.Sprintf("~%d tokens (%.1f%%, %d @import(s))", cs.EstimatedTokens, pct, cs.ImportCount)
			}
			fileText := styles.SubtleStyle.Render(cs.File)
			b.WriteString("   " + f.fileHyperlink(cs.File, fileText) + "  " + label + "\n")
		}
		b.WriteString("\n")
	}

	// Rollup summary: show per-group breakdown when plugins exist
	if len(summary) > 1 || (len(summary) == 1 && summary[0].Name != "project") {
		for _, g := range summary {
			name := g.Name
			if name == "project" {
				name = "project root"
			}
			pct := float64(g.EstimatedTokens) * 100 / float64(result.ContextWindow)
			fmt.Fprintf(b, "   %s  ~%d tokens (%.1f%%, %d file(s))\n",
				styles.SubtleStyle.Render(name), g.EstimatedTokens, pct, g.FileCount)
		}
	}
	totalPct := float64(total) * 100 / float64(result.ContextWindow)
	fmt.Fprintf(b, "   %s  ~%d tokens (%.1f%% of %dK context window)\n\n",
		ruleHeaderStyle.Render("total context"), total, totalPct, result.ContextWindow/1000)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// fileHyperlink wraps displayText in an OSC 8 terminal hyperlink pointing to a file:// URL.
// If WorkDir is empty or the path can't be resolved, returns displayText unchanged.
func (f *ConsoleFormatter) fileHyperlink(path, displayText string) string {
	if f.WorkDir == "" {
		return displayText
	}
	absPath := path
	if !filepath.IsAbs(path) {
		absPath = filepath.Join(f.WorkDir, path)
	}
	fileURL := "file://" + url.PathEscape(absPath)
	// Restore path separators that PathEscape encodes
	fileURL = strings.ReplaceAll(fileURL, "%2F", "/")
	return fmt.Sprintf("\033]8;;%s\033\\%s\033]8;;\033\\", fileURL, displayText)
}

// JSONFormatter renders diagnostics as JSON.
type JSONFormatter struct{}

type jsonDiagnostic struct {
	Severity string `json:"severity"`
	Message  string `json:"message"`
	Tip      string `json:"tip,omitempty"`
	File     string `json:"file,omitempty"`
	Rule     string `json:"rule,omitempty"`
}

type jsonContextStat struct {
	File            string `json:"file"`
	EstimatedTokens int    `json:"estimated_tokens"`
	ImportCount     int    `json:"import_count"`
	PluginDir       string `json:"plugin_dir,omitempty"`
}

type jsonContextGroup struct {
	Name            string `json:"name"`
	EstimatedTokens int    `json:"estimated_tokens"`
	FileCount       int    `json:"file_count"`
}

type jsonOutput struct {
	FilesScanned   int                `json:"files_scanned"`
	Diagnostics    []jsonDiagnostic   `json:"diagnostics"`
	ContextWindow  int                `json:"context_window,omitempty"`
	ContextStats   []jsonContextStat  `json:"context_stats,omitempty"`
	ContextSummary []jsonContextGroup `json:"context_summary,omitempty"`
	TotalTokens    int                `json:"total_estimated_tokens,omitempty"`
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
			Tip:      d.Tip,
			File:     d.File,
			Rule:     d.Rule,
		})
	}
	if len(result.ContextStats) > 0 {
		out.ContextWindow = result.ContextWindow
		hasFindings := result.HasContextFindings()
		if result.Verbose || hasFindings {
			for _, cs := range result.ContextStats {
				out.ContextStats = append(out.ContextStats, jsonContextStat(cs))
			}
		}
		summary, total := contextSummary(result.ContextStats)
		out.TotalTokens = total
		for _, g := range summary {
			out.ContextSummary = append(out.ContextSummary, jsonContextGroup(g))
		}
	}
	data, _ := json.MarshalIndent(out, "", "  ")
	return string(data) + "\n"
}

// MarkdownFormatter renders diagnostics as markdown.
type MarkdownFormatter struct{}

func (f *MarkdownFormatter) Format(result *AssayResult) string {
	var b strings.Builder

	fmt.Fprintf(&b, "# Assay Results\n\n**Files scanned:** %d\n\n", result.FilesScanned)

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
		fmt.Fprintf(&b, "## %s %s (%d)\n\n", l.emoji, l.title, len(diags))
		for _, d := range diags {
			file := ""
			if d.File != "" {
				file = fmt.Sprintf("`%s`: ", d.File)
			}
			fmt.Fprintf(&b, "- %s%s\n", file, d.Message)
			if d.Tip != "" {
				fmt.Fprintf(&b, "  > 💡 %s\n", d.Tip)
			}
		}
		b.WriteString("\n")
	}

	if len(result.ContextStats) > 0 {
		hasFindings := result.HasContextFindings()
		summary, total := contextSummary(result.ContextStats)

		b.WriteString("## :bar_chart: Context Usage\n\n")

		if result.Verbose || hasFindings {
			b.WriteString("| File | Est. Tokens | % of Context | @imports |\n")
			b.WriteString("|------|-------------|--------------|----------|\n")
			for _, cs := range result.ContextStats {
				pct := float64(cs.EstimatedTokens) * 100 / float64(result.ContextWindow)
				fmt.Fprintf(&b, "| `%s` | ~%d | %.1f%% | %d |\n", cs.File, cs.EstimatedTokens, pct, cs.ImportCount)
			}
			b.WriteString("\n")
		}

		if len(summary) > 1 || (len(summary) == 1 && summary[0].Name != "project") {
			b.WriteString("| Group | Est. Tokens | % of Context | Files |\n")
			b.WriteString("|-------|-------------|--------------|-------|\n")
			for _, g := range summary {
				name := g.Name
				if name == "project" {
					name = "project root"
				}
				pct := float64(g.EstimatedTokens) * 100 / float64(result.ContextWindow)
				fmt.Fprintf(&b, "| %s | ~%d | %.1f%% | %d |\n", name, g.EstimatedTokens, pct, g.FileCount)
			}
			b.WriteString("\n")
		}

		totalPct := float64(total) * 100 / float64(result.ContextWindow)
		fmt.Fprintf(&b, "**Total estimated context:** ~%d tokens (%.1f%% of %dK context window)\n\n", total, totalPct, result.ContextWindow/1000)
	}

	return b.String()
}
