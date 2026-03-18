package assay

import (
	"bufio"
	"bytes"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/nimble-giant/ailloy/pkg/mold"
)

const maxInt = int(math.MaxInt)

var headingRegex = regexp.MustCompile(`(?m)^#{1,6}\s+`)

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
			if n <= uint64(maxInt) {
				maxLines = int(n)
			}
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
	var claudeMDPath string

	for _, f := range ctx.Files {
		base := filepath.Base(f.Path)
		if base == "AGENTS.md" {
			hasAgentsMD = true
		}
		if f.Platform != PlatformGeneric {
			hasPlatformFiles = true
		}
		if base == "CLAUDE.md" && claudeMDPath == "" {
			claudeMDPath = f.Path
		}
	}

	if hasPlatformFiles && !hasAgentsMD {
		var tip string
		if claudeMDPath != "" {
			tip = fmt.Sprintf(
				"move shared instructions to AGENTS.md and replace the body of %s with @AGENTS.md — "+
					"Claude Code will still import the file, and other agents will pick it up natively",
				claudeMDPath)
		} else {
			tip = "create AGENTS.md with your shared AI instructions; all agents (Claude, Codex, etc.) read it natively"
		}
		return []mold.Diagnostic{{
			Severity: r.DefaultSeverity(),
			Message:  "project has platform-specific AI instruction files but no AGENTS.md; consider creating one for cross-platform compatibility",
			Tip:      tip,
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

// duplicateTopicsRule warns when the same heading appears in multiple files
// with similar content, suggesting the content should be centralized.
type duplicateTopicsRule struct{}

func (r *duplicateTopicsRule) Name() string                       { return "duplicate-topics" }
func (r *duplicateTopicsRule) DefaultSeverity() mold.DiagSeverity { return mold.SeverityWarning }
func (r *duplicateTopicsRule) Platforms() []Platform              { return nil }

// sectionEntry pairs a file path with the body content under a heading.
type sectionEntry struct {
	path    string
	content string
}

func (r *duplicateTopicsRule) Check(ctx *RuleContext) []mold.Diagnostic {
	// Map normalized heading -> list of (file, section content) pairs.
	sections := make(map[string][]sectionEntry)

	for _, f := range ctx.Files {
		if !strings.HasSuffix(f.Path, ".md") {
			continue
		}
		for heading, body := range extractSections(f.Content) {
			sections[heading] = append(sections[heading], sectionEntry{
				path:    f.Path,
				content: body,
			})
		}
	}

	var diags []mold.Diagnostic
	for heading, entries := range sections {
		if len(entries) <= 1 {
			continue
		}
		// Deduplicate by file path — a heading could repeat within one file.
		byFile := make(map[string]string) // path -> content
		for _, e := range entries {
			if _, exists := byFile[e.path]; !exists {
				byFile[e.path] = e.content
			}
		}
		if len(byFile) <= 1 {
			continue
		}

		// Compare every pair of sections for similarity.
		// Only warn if at least one pair has substantially similar content.
		paths := make([]string, 0, len(byFile))
		bodies := make([]string, 0, len(byFile))
		for p, b := range byFile {
			paths = append(paths, p)
			bodies = append(bodies, b)
		}

		var similarFiles []string
		for i := 0; i < len(bodies); i++ {
			for j := i + 1; j < len(bodies); j++ {
				if contentSimilarity(bodies[i], bodies[j]) >= 0.7 {
					similarFiles = appendUnique(similarFiles, paths[i])
					similarFiles = appendUnique(similarFiles, paths[j])
				}
			}
		}

		if len(similarFiles) >= 2 {
			diags = append(diags, mold.Diagnostic{
				Severity: r.DefaultSeverity(),
				Message: fmt.Sprintf(
					"heading %q has similar content in multiple files — consider centralizing: %s",
					heading, strings.Join(similarFiles, ", ")),
				Tip:  "extract the shared content into a dedicated .md file and reference it with @path/to/file.md in each instruction file",
				Rule: r.Name(),
			})
		}
	}
	return diags
}

// extractSections splits markdown content into a map of normalized heading -> body text.
// Only the content between a heading and the next heading of equal or higher level is captured.
func extractSections(content []byte) map[string]string {
	sections := make(map[string]string)
	var currentHeading string
	var currentBody strings.Builder

	scanner := bufio.NewScanner(bytes.NewReader(content))
	for scanner.Scan() {
		line := scanner.Text()
		if headingRegex.MatchString(line) {
			// Flush previous section.
			if currentHeading != "" {
				sections[currentHeading] = strings.TrimSpace(currentBody.String())
			}
			currentHeading = strings.ToLower(strings.TrimSpace(
				headingRegex.ReplaceAllString(line, "")))
			currentBody.Reset()
		} else if currentHeading != "" {
			currentBody.WriteString(line)
			currentBody.WriteByte('\n')
		}
	}
	// Flush last section.
	if currentHeading != "" {
		sections[currentHeading] = strings.TrimSpace(currentBody.String())
	}
	return sections
}

// contentSimilarity returns a 0–1 score comparing two section bodies.
// It uses trigram (3-gram) overlap as a simple, effective measure of textual similarity.
func contentSimilarity(a, b string) float64 {
	a = normalizeContent(a)
	b = normalizeContent(b)

	// Empty sections are not meaningful duplicates.
	if len(a) == 0 || len(b) == 0 {
		return 0.0
	}

	// Treat very short content (likely just a link or one-liner) as unique
	// unless they are identical.
	if len(a) < 20 || len(b) < 20 {
		if a == b {
			return 1.0
		}
		return 0.0
	}

	triA := trigrams(a)
	triB := trigrams(b)
	if len(triA) == 0 || len(triB) == 0 {
		return 0.0
	}

	// Jaccard similarity over trigram sets.
	intersection := 0
	for tri := range triA {
		if triB[tri] {
			intersection++
		}
	}
	union := len(triA) + len(triB) - intersection
	if union == 0 {
		return 0.0
	}
	return float64(intersection) / float64(union)
}

// normalizeContent collapses whitespace and lowercases for comparison.
func normalizeContent(s string) string {
	fields := strings.Fields(strings.ToLower(s))
	return strings.Join(fields, " ")
}

// trigrams returns the set of character 3-grams in s.
func trigrams(s string) map[string]bool {
	if len(s) < 3 {
		return nil
	}
	set := make(map[string]bool, len(s)-2)
	for i := 0; i <= len(s)-3; i++ {
		set[s[i:i+3]] = true
	}
	return set
}

func appendUnique(slice []string, val string) []string {
	for _, s := range slice {
		if s == val {
			return slice
		}
	}
	return append(slice, val)
}
