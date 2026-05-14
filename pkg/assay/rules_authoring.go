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

// defaultReferenceTOCLines is the line threshold above which a skill
// reference file should declare a Table of Contents heading.
//
// Anthropic guidance: "For reference files longer than 100 lines, include a
// table of contents at the top. This ensures Claude can see the full scope
// of available information even when previewing with partial reads."
const defaultReferenceTOCLines = 100

// tocHeadingRegex matches `## Contents`, `## Table of contents`, or
// `## Table of Contents` (case-insensitive) — the conventional TOC markers.
var tocHeadingRegex = regexp.MustCompile(`(?im)^#{2,6}\s+(contents|table of contents)\s*$`)

// referenceFileTOCRule warns when a markdown file inside a skill directory
// (other than SKILL.md) exceeds the line threshold without a TOC heading.
// See: https://platform.claude.com/docs/en/agents-and-tools/agent-skills/best-practices#structure-longer-reference-files-with-table-of-contents
type referenceFileTOCRule struct{}

func (r *referenceFileTOCRule) Name() string                       { return "reference-file-toc" }
func (r *referenceFileTOCRule) DefaultSeverity() mold.DiagSeverity { return mold.SeverityWarning }
func (r *referenceFileTOCRule) Platforms() []Platform              { return []Platform{PlatformClaude} }

func (r *referenceFileTOCRule) Check(ctx *RuleContext) []mold.Diagnostic {
	maxLines := defaultReferenceTOCLines
	if v := ctx.Config.RuleOption(r.Name(), "max-lines", nil); v != nil {
		maxLines = toInt(v, maxLines)
	}

	var diags []mold.Diagnostic
	for _, f := range ctx.Files {
		if f.Platform != PlatformClaude {
			continue
		}
		if filepath.Ext(f.Path) != ".md" {
			continue
		}
		if filepath.Base(f.Path) == "SKILL.md" {
			continue
		}
		if !isSkillPath(f) {
			continue
		}

		body := f.Content
		if fm := extractFrontmatter(f.Content); fm != nil {
			idx := bytes.Index(f.Content[3:], []byte("\n---"))
			if idx >= 0 {
				body = f.Content[3+idx+4:]
			}
		}

		lines := bytes.Count(body, []byte("\n"))
		if len(body) > 0 && body[len(body)-1] != '\n' {
			lines++
		}
		if lines <= maxLines {
			continue
		}
		if tocHeadingRegex.Match(body) {
			continue
		}

		diags = append(diags, mold.Diagnostic{
			Severity: r.DefaultSeverity(),
			Message:  fmt.Sprintf("skill reference file is %d lines (threshold: %d) and lacks a Table of Contents heading", lines, maxLines),
			Tip:      "add a `## Contents` section near the top so Claude can see the file's scope on partial reads; see https://platform.claude.com/docs/en/agents-and-tools/agent-skills/best-practices#structure-longer-reference-files-with-table-of-contents",
			File:     f.Path,
			Rule:     r.Name(),
		})
	}
	return diags
}

// markdownLinkRegex matches inline markdown links to local markdown files,
// e.g. `[label](path/to/file.md)` or `[label](./file.md)`. It deliberately
// excludes anchors-only (`#section`), absolute URLs (`http(s)://`), and
// non-markdown targets.
var markdownLinkRegex = regexp.MustCompile(`\[[^\]]*\]\(([^)\s#]+\.md)(?:#[^)]*)?\)`)

// referenceDepthRule warns when SKILL.md references files that themselves
// reference other markdown files — i.e., the reference graph is deeper than
// one level. Anthropic guidance: "Keep references one level deep from
// SKILL.md. All reference files should link directly from SKILL.md to
// ensure Claude reads complete files when needed."
type referenceDepthRule struct{}

func (r *referenceDepthRule) Name() string                       { return "reference-depth" }
func (r *referenceDepthRule) DefaultSeverity() mold.DiagSeverity { return mold.SeverityWarning }
func (r *referenceDepthRule) Platforms() []Platform              { return []Platform{PlatformClaude} }

func (r *referenceDepthRule) Check(ctx *RuleContext) []mold.Diagnostic {
	var diags []mold.Diagnostic
	for _, f := range ctx.Files {
		if f.Platform != PlatformClaude {
			continue
		}
		if filepath.Base(f.Path) != "SKILL.md" {
			continue
		}
		if !isSkillPath(f) {
			continue
		}

		skillDir := filepath.Dir(f.Path)
		level1 := collectMarkdownRefs(f.Content, skillDir, ctx.RootDir)

		for _, ref := range level1 {
			content, err := os.ReadFile(filepath.Join(ctx.RootDir, ref)) //#nosec G304
			if err != nil {
				continue
			}
			refDir := filepath.Dir(ref)
			level2 := collectMarkdownRefs(content, refDir, ctx.RootDir)
			// Filter out self-references and references back to SKILL.md.
			var deep []string
			for _, l2 := range level2 {
				if l2 == ref || l2 == f.Path {
					continue
				}
				deep = append(deep, l2)
			}
			if len(deep) == 0 {
				continue
			}
			diags = append(diags, mold.Diagnostic{
				Severity: r.DefaultSeverity(),
				Message:  fmt.Sprintf("reference %q links onward to %s; nested references >1 level deep from SKILL.md may be partial-read by Claude", ref, strings.Join(deep, ", ")),
				Tip:      "flatten the structure so all reference files link directly from SKILL.md; see https://platform.claude.com/docs/en/agents-and-tools/agent-skills/best-practices#avoid-deeply-nested-references",
				File:     f.Path,
				Rule:     r.Name(),
			})
		}
	}
	return diags
}

// collectMarkdownRefs returns the relative-from-root paths of markdown files
// referenced by the given content. It checks both `[label](path.md)` links
// and `@path.md` imports, resolving each against the provided file directory.
func collectMarkdownRefs(content []byte, fileDir, rootDir string) []string {
	seen := make(map[string]bool)
	var refs []string

	add := func(rel string) {
		if seen[rel] {
			return
		}
		seen[rel] = true
		refs = append(refs, rel)
	}

	for _, m := range markdownLinkRegex.FindAllSubmatch(content, -1) {
		target := string(m[1])
		if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") {
			continue
		}
		resolved, ok := resolveLocalMarkdown(target, fileDir, rootDir)
		if !ok {
			continue
		}
		add(resolved)
	}

	for _, ref := range extractImportRefs(content) {
		if !strings.HasSuffix(ref, ".md") {
			continue
		}
		resolved, ok := resolveImportRef(ref, fileDir, rootDir)
		if !ok {
			continue
		}
		rel, err := filepath.Rel(rootDir, resolved)
		if err != nil {
			continue
		}
		add(filepath.ToSlash(rel))
	}

	return refs
}

// resolveLocalMarkdown resolves a relative markdown link against the
// containing file's directory, returning a project-relative slash path.
func resolveLocalMarkdown(target, fileDir, rootDir string) (string, bool) {
	candidate := filepath.Join(rootDir, fileDir, target)
	clean := filepath.Clean(candidate)
	if _, err := os.Stat(clean); err != nil {
		return "", false
	}
	rel, err := filepath.Rel(rootDir, clean)
	if err != nil {
		return "", false
	}
	return filepath.ToSlash(rel), true
}

// windowsPathRegex finds backslash-separated path tokens with a file
// extension, e.g. `scripts\helper.py` or `reference\guide.md`. It looks for
// 2+ segments joined by `\` ending in a short extension.
var windowsPathRegex = regexp.MustCompile(`\b[\w.\-]+\\[\w.\-\\]+\.[a-zA-Z0-9]{1,5}\b`)

// windowsPathsRule warns on backslash-separated path tokens in markdown
// content. Anthropic guidance: "Always use forward slashes in file paths,
// even on Windows... Windows-style paths cause errors on Unix systems."
type windowsPathsRule struct{}

func (r *windowsPathsRule) Name() string                       { return "windows-paths" }
func (r *windowsPathsRule) DefaultSeverity() mold.DiagSeverity { return mold.SeverityWarning }
func (r *windowsPathsRule) Platforms() []Platform              { return nil }

func (r *windowsPathsRule) Check(ctx *RuleContext) []mold.Diagnostic {
	var diags []mold.Diagnostic
	for _, f := range ctx.Files {
		if filepath.Ext(f.Path) != ".md" {
			continue
		}
		matches := windowsPathRegex.FindAll(stripCodeFences(f.Content), -1)
		if len(matches) == 0 {
			continue
		}
		seen := make(map[string]bool)
		var uniq []string
		for _, m := range matches {
			s := string(m)
			if seen[s] {
				continue
			}
			seen[s] = true
			uniq = append(uniq, s)
		}
		diags = append(diags, mold.Diagnostic{
			Severity: r.DefaultSeverity(),
			Message:  fmt.Sprintf("file references Windows-style paths (%s); use forward slashes for cross-platform compatibility", strings.Join(uniq, ", ")),
			Tip:      "replace `\\` with `/` in file paths; see https://platform.claude.com/docs/en/agents-and-tools/agent-skills/best-practices#avoid-windows-style-paths",
			File:     f.Path,
			Rule:     r.Name(),
		})
	}
	return diags
}

// stripCodeFences removes fenced code blocks (```...```) so the path
// regex doesn't fire on quoted Windows shell examples.
func stripCodeFences(content []byte) []byte {
	var out bytes.Buffer
	scanner := bufio.NewScanner(bytes.NewReader(content))
	inFence := false
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			inFence = !inFence
			out.WriteByte('\n')
			continue
		}
		if inFence {
			out.WriteByte('\n')
			continue
		}
		out.WriteString(line)
		out.WriteByte('\n')
	}
	return out.Bytes()
}

// gerundVerbWhitelist enumerates the action-oriented first-segment names
// Anthropic explicitly accepts as alternatives to gerund form. The list is
// intentionally small — coverage is biased toward common skill verbs to
// keep false positives low.
var gerundVerbWhitelist = map[string]bool{
	"process":   true,
	"analyze":   true,
	"generate":  true,
	"build":     true,
	"create":    true,
	"manage":    true,
	"write":     true,
	"test":      true,
	"debug":     true,
	"deploy":    true,
	"fetch":     true,
	"parse":     true,
	"validate":  true,
	"convert":   true,
	"extract":   true,
	"review":    true,
	"lint":      true,
	"format":    true,
	"run":       true,
	"update":    true,
	"add":       true,
	"remove":    true,
	"start":     true,
	"open":      true,
	"close":     true,
	"render":    true,
	"compile":   true,
	"setup":     true,
	"configure": true,
	"install":   true,
	"sync":      true,
	"check":     true,
	"find":      true,
	"search":    true,
	"replace":   true,
	"translate": true,
	"refactor":  true,
	"summarize": true,
}

// nameGerundFormRule suggests gerund-form names (verb + -ing) for skills,
// commands, and agents. It accepts any name containing an `-ing` segment
// (covers gerund and noun-phrase forms like `pdf-processing`) and any name
// whose first segment is a known action verb. Everything else gets a
// suggestion-level nudge.
//
// See: https://platform.claude.com/docs/en/agents-and-tools/agent-skills/best-practices#naming-conventions
type nameGerundFormRule struct{}

func (r *nameGerundFormRule) Name() string { return "name-gerund-form" }
func (r *nameGerundFormRule) DefaultSeverity() mold.DiagSeverity {
	return mold.SeveritySuggestion
}
func (r *nameGerundFormRule) Platforms() []Platform { return []Platform{PlatformClaude} }

func (r *nameGerundFormRule) Check(ctx *RuleContext) []mold.Diagnostic {
	var diags []mold.Diagnostic
	for _, f := range ctx.Files {
		// Anthropic's gerund-form guidance targets skills/commands/agents
		// (units of agent behavior), not plugin packages. Skip plugin manifests
		// — they follow package-naming conventions (e.g., "supabase", "imessage").
		if filepath.Ext(f.Path) == ".json" {
			continue
		}
		name, ok := extractName(f)
		if !ok || name == "" {
			continue
		}
		// Skip names already caught by vague-name; those have their own message.
		if vagueNames[strings.ToLower(name)] {
			continue
		}
		segments := strings.Split(strings.ToLower(name), "-")
		if len(segments) == 0 {
			continue
		}
		if gerundVerbWhitelist[segments[0]] {
			continue
		}
		hasIng := false
		for _, seg := range segments {
			if strings.HasSuffix(seg, "ing") && len(seg) > 3 {
				hasIng = true
				break
			}
		}
		if hasIng {
			continue
		}
		diags = append(diags, mold.Diagnostic{
			Severity: r.DefaultSeverity(),
			Message:  fmt.Sprintf("name %q does not use gerund or action-verb form; gerund form (e.g., %q) triggers more reliably", name, gerundSuggestion(name)),
			Tip:      "prefer names like `processing-pdfs` or `managing-databases`; see https://platform.claude.com/docs/en/agents-and-tools/agent-skills/best-practices#naming-conventions",
			File:     f.Path,
			Rule:     r.Name(),
		})
	}
	return diags
}

// gerundSuggestion returns a best-effort gerund-form rewrite of name for the
// diagnostic message. It is a simple cosmetic transform — not a guarantee
// of correctness — so users see a concrete example, not just abstract guidance.
func gerundSuggestion(name string) string {
	segments := strings.Split(strings.ToLower(name), "-")
	if len(segments) == 0 {
		return name
	}
	first := segments[0]
	switch {
	case strings.HasSuffix(first, "e") && len(first) > 1:
		first = first[:len(first)-1] + "ing"
	default:
		first = first + "ing"
	}
	segments[0] = first
	return strings.Join(segments, "-")
}
