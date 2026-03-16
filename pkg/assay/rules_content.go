package assay

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/nimble-giant/ailloy/pkg/mold"
)

var headingRegex = regexp.MustCompile(`^#{1,6}\s+`)

func init() {
	Register(&lineCountRule{})
	Register(&structureRule{})
	Register(&agentsMDPresenceRule{})
	Register(&crossReferenceRule{})
	Register(&importValidationRule{})
	Register(&emptyFileRule{})
	Register(&duplicateTopicsRule{})
}

// lineCountRule warns when an instruction file exceeds a line threshold.
type lineCountRule struct{}

func (r *lineCountRule) Name() string                       { return "line-count" }
func (r *lineCountRule) DefaultSeverity() mold.DiagSeverity { return mold.SeverityWarning }
func (r *lineCountRule) Platforms() []Platform              { return nil }

func (r *lineCountRule) Check(ctx *RuleContext) []mold.Diagnostic {
	maxLines := 150
	if v := ctx.Config.RuleOption("line-count", "max-lines", nil); v != nil {
		switch n := v.(type) {
		case int:
			maxLines = n
		case uint64:
			maxLines = int(n)
		case float64:
			maxLines = int(n)
		}
	}

	var diags []mold.Diagnostic
	for _, f := range ctx.Files {
		lines := bytes.Count(f.Content, []byte("\n"))
		if len(f.Content) > 0 && f.Content[len(f.Content)-1] != '\n' {
			lines++ // count last line without trailing newline
		}
		if lines > maxLines {
			diags = append(diags, mold.Diagnostic{
				Severity: r.DefaultSeverity(),
				Message:  fmt.Sprintf("file has %d lines (threshold: %d); consider splitting into smaller files", lines, maxLines),
				File:     f.Path,
				Rule:     r.Name(),
			})
		}
	}
	return diags
}

// structureRule warns when an instruction file lacks markdown headings.
type structureRule struct{}

func (r *structureRule) Name() string                       { return "structure" }
func (r *structureRule) DefaultSeverity() mold.DiagSeverity { return mold.SeverityWarning }
func (r *structureRule) Platforms() []Platform              { return nil }

func (r *structureRule) Check(ctx *RuleContext) []mold.Diagnostic {
	var diags []mold.Diagnostic
	for _, f := range ctx.Files {
		if !strings.HasSuffix(f.Path, ".md") {
			continue
		}
		if len(bytes.TrimSpace(f.Content)) == 0 {
			continue // empty files handled by emptyFileRule
		}
		if !headingRegex.Match(f.Content) {
			diags = append(diags, mold.Diagnostic{
				Severity: r.DefaultSeverity(),
				Message:  "file lacks markdown headings; structured instructions improve AI adherence",
				File:     f.Path,
				Rule:     r.Name(),
			})
		}
	}
	return diags
}

// agentsMDPresenceRule suggests creating AGENTS.md if AI instruction files exist but AGENTS.md does not.
type agentsMDPresenceRule struct{}

func (r *agentsMDPresenceRule) Name() string                       { return "agents-md-presence" }
func (r *agentsMDPresenceRule) DefaultSeverity() mold.DiagSeverity { return mold.SeveritySuggestion }
func (r *agentsMDPresenceRule) Platforms() []Platform              { return nil }

func (r *agentsMDPresenceRule) Check(ctx *RuleContext) []mold.Diagnostic {
	hasAgentsMD := false
	hasPlatformFiles := false

	for _, f := range ctx.Files {
		if filepath.Base(f.Path) == "AGENTS.md" {
			hasAgentsMD = true
		}
		if f.Platform != PlatformGeneric {
			hasPlatformFiles = true
		}
	}

	if hasPlatformFiles && !hasAgentsMD {
		return []mold.Diagnostic{{
			Severity: r.DefaultSeverity(),
			Message:  "project has platform-specific AI instruction files but no AGENTS.md; consider creating one for cross-platform compatibility",
			Rule:     r.Name(),
		}}
	}
	return nil
}

// crossReferenceRule warns when CLAUDE.md exists alongside AGENTS.md but doesn't import it.
type crossReferenceRule struct{}

func (r *crossReferenceRule) Name() string                       { return "cross-reference" }
func (r *crossReferenceRule) DefaultSeverity() mold.DiagSeverity { return mold.SeverityWarning }
func (r *crossReferenceRule) Platforms() []Platform              { return []Platform{PlatformClaude} }

func (r *crossReferenceRule) Check(ctx *RuleContext) []mold.Diagnostic {
	var claudeFiles []DetectedFile
	hasAgentsMD := false

	for _, f := range ctx.Files {
		if filepath.Base(f.Path) == "AGENTS.md" && filepath.Dir(f.Path) == "." {
			hasAgentsMD = true
		}
		if f.Platform == PlatformClaude && filepath.Base(f.Path) == "CLAUDE.md" {
			claudeFiles = append(claudeFiles, f)
		}
	}

	if !hasAgentsMD {
		return nil
	}

	var diags []mold.Diagnostic
	for _, f := range claudeFiles {
		content := strings.ToLower(string(f.Content))
		if !strings.Contains(content, "@agents.md") {
			diags = append(diags, mold.Diagnostic{
				Severity: r.DefaultSeverity(),
				Message:  "AGENTS.md exists but is not referenced via @AGENTS.md import",
				File:     f.Path,
				Rule:     r.Name(),
			})
		}
	}
	return diags
}

// importValidationRule checks that @path/to/file references resolve to existing files.
type importValidationRule struct{}

func (r *importValidationRule) Name() string                       { return "import-validation" }
func (r *importValidationRule) DefaultSeverity() mold.DiagSeverity { return mold.SeverityError }
func (r *importValidationRule) Platforms() []Platform              { return []Platform{PlatformClaude} }

var importRefRegex = regexp.MustCompile(`(?m)^@(\S+)`)

func (r *importValidationRule) Check(ctx *RuleContext) []mold.Diagnostic {
	var diags []mold.Diagnostic
	for _, f := range ctx.Files {
		if f.Platform != PlatformClaude {
			continue
		}
		scanner := bufio.NewScanner(bytes.NewReader(f.Content))
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if !strings.HasPrefix(line, "@") {
				continue
			}
			matches := importRefRegex.FindStringSubmatch(line)
			if len(matches) < 2 {
				continue
			}
			ref := matches[1]
			// Resolve relative to the file's directory
			refDir := filepath.Dir(f.Path)
			refPath := filepath.Join(ctx.RootDir, refDir, ref)
			if _, err := os.Stat(refPath); err != nil {
				// Also try from project root
				rootPath := filepath.Join(ctx.RootDir, ref)
				if _, err := os.Stat(rootPath); err != nil {
					diags = append(diags, mold.Diagnostic{
						Severity: r.DefaultSeverity(),
						Message:  fmt.Sprintf("import @%s does not resolve to an existing file", ref),
						File:     f.Path,
						Rule:     r.Name(),
					})
				}
			}
		}
	}
	return diags
}

// emptyFileRule warns when an instruction file exists but has no meaningful content.
type emptyFileRule struct{}

func (r *emptyFileRule) Name() string                       { return "empty-file" }
func (r *emptyFileRule) DefaultSeverity() mold.DiagSeverity { return mold.SeverityWarning }
func (r *emptyFileRule) Platforms() []Platform              { return nil }

func (r *emptyFileRule) Check(ctx *RuleContext) []mold.Diagnostic {
	var diags []mold.Diagnostic
	for _, f := range ctx.Files {
		if len(bytes.TrimSpace(f.Content)) == 0 {
			diags = append(diags, mold.Diagnostic{
				Severity: r.DefaultSeverity(),
				Message:  "file exists but has no meaningful content",
				File:     f.Path,
				Rule:     r.Name(),
			})
		}
	}
	return diags
}

// duplicateTopicsRule warns when the same heading appears in multiple instruction files.
type duplicateTopicsRule struct{}

func (r *duplicateTopicsRule) Name() string                       { return "duplicate-topics" }
func (r *duplicateTopicsRule) DefaultSeverity() mold.DiagSeverity { return mold.SeverityWarning }
func (r *duplicateTopicsRule) Platforms() []Platform              { return nil }

func (r *duplicateTopicsRule) Check(ctx *RuleContext) []mold.Diagnostic {
	// Map heading text -> list of files containing it
	headings := make(map[string][]string)

	for _, f := range ctx.Files {
		if !strings.HasSuffix(f.Path, ".md") {
			continue
		}
		scanner := bufio.NewScanner(bytes.NewReader(f.Content))
		for scanner.Scan() {
			line := scanner.Text()
			if headingRegex.MatchString(line) {
				// Normalize: strip # prefix and whitespace
				heading := strings.TrimSpace(headingRegex.ReplaceAllString(line, ""))
				heading = strings.ToLower(heading)
				if heading != "" {
					headings[heading] = append(headings[heading], f.Path)
				}
			}
		}
	}

	var diags []mold.Diagnostic
	for heading, paths := range headings {
		if len(paths) <= 1 {
			continue
		}
		// Deduplicate paths
		unique := make(map[string]bool)
		for _, p := range paths {
			unique[p] = true
		}
		if len(unique) <= 1 {
			continue
		}
		var fileList []string
		for p := range unique {
			fileList = append(fileList, p)
		}
		diags = append(diags, mold.Diagnostic{
			Severity: r.DefaultSeverity(),
			Message:  fmt.Sprintf("heading %q appears in multiple files: %s", heading, strings.Join(fileList, ", ")),
			Rule:     r.Name(),
		})
	}
	return diags
}
