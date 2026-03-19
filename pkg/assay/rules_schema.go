package assay

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/nimble-giant/ailloy/pkg/mold"
)

func init() {
	Register(&agentFrontmatterRule{})
	Register(&commandFrontmatterRule{})
	Register(&settingsSchemaRule{})
	Register(&pluginManifestRule{})
	Register(&pluginHooksRule{})
	Register(&descriptionLengthRule{})
	Register(&descriptionPointOfViewRule{})
	Register(&descriptionMissingTriggerRule{})
	Register(&nameFormatRule{})
	Register(&nameReservedWordsRule{})
	Register(&vagueNameRule{})
	Register(&skillBodyLengthRule{})
	Register(&commandsDeprecatedRule{})
	Register(&nameDirectoryMismatchRule{})
	Register(&descriptionMaxLengthRule{})
	Register(&compatibilityLengthRule{})
	Register(&skillTokenBudgetRule{})
	Register(&descriptionImperativeRule{})
}

// isUnderPluginSubdir returns true if path is at any depth under
// <pluginDir>/<subdir>/ (handles recursive file collection).
func isUnderPluginSubdir(path, pluginDir, subdir string) bool {
	prefix := filepath.Join(pluginDir, subdir) + string(filepath.Separator)
	return strings.HasPrefix(path, prefix)
}

// agentFrontmatterRule validates .claude/agents/*.yml files.
type agentFrontmatterRule struct{}

func (r *agentFrontmatterRule) Name() string                       { return "agent-frontmatter" }
func (r *agentFrontmatterRule) DefaultSeverity() mold.DiagSeverity { return mold.SeverityError }
func (r *agentFrontmatterRule) Platforms() []Platform              { return []Platform{PlatformClaude} }

func (r *agentFrontmatterRule) Check(ctx *RuleContext) []mold.Diagnostic {
	var diags []mold.Diagnostic
	for _, f := range ctx.Files {
		if f.Platform != PlatformClaude {
			continue
		}
		dir := filepath.Dir(f.Path)
		ext := filepath.Ext(f.Path)
		if ext != ".yml" && ext != ".yaml" {
			continue
		}
		isStandardAgent := dir == filepath.Join(".claude", "agents")
		isPluginAgent := f.PluginDir != "" && isUnderPluginSubdir(f.Path, f.PluginDir, "agents")
		if !isStandardAgent && !isPluginAgent {
			continue
		}

		var agent map[string]any
		if err := yaml.Unmarshal(f.Content, &agent); err != nil {
			diags = append(diags, mold.Diagnostic{
				Severity: mold.SeverityError,
				Message:  fmt.Sprintf("invalid YAML: %v", err),
				File:     f.Path,
				Rule:     r.Name(),
			})
			continue
		}

		if _, ok := agent["name"]; !ok {
			diags = append(diags, mold.Diagnostic{
				Severity: mold.SeverityError,
				Message:  "agent definition missing required field: name",
				File:     f.Path,
				Rule:     r.Name(),
			})
		}
		if _, ok := agent["description"]; !ok {
			diags = append(diags, mold.Diagnostic{
				Severity: mold.SeverityWarning,
				Message:  "agent definition missing recommended field: description",
				File:     f.Path,
				Rule:     r.Name(),
			})
		}
	}
	return diags
}

// commandFrontmatterRule validates .claude/commands/*.md and plugin commands/skills frontmatter.
type commandFrontmatterRule struct{}

func (r *commandFrontmatterRule) Name() string                       { return "command-frontmatter" }
func (r *commandFrontmatterRule) DefaultSeverity() mold.DiagSeverity { return mold.SeverityWarning }
func (r *commandFrontmatterRule) Platforms() []Platform              { return []Platform{PlatformClaude} }

var validCommandFields = map[string]bool{
	"allowed-tools": true,
	"argument-hint": true,
	"model":         true,
	"description":   true,
	"name":          true,
}

func (r *commandFrontmatterRule) Check(ctx *RuleContext) []mold.Diagnostic {
	// Build the set of allowed fields: built-in fields + any user-configured extras.
	allowed := make(map[string]bool, len(validCommandFields))
	for k, v := range validCommandFields {
		allowed[k] = v
	}
	if extras, ok := ctx.Config.RuleOption(r.Name(), "extra-allowed-fields", nil).([]any); ok {
		for _, e := range extras {
			if s, ok := e.(string); ok {
				allowed[s] = true
			}
		}
	}

	var diags []mold.Diagnostic
	for _, f := range ctx.Files {
		if f.Platform != PlatformClaude {
			continue
		}
		if filepath.Ext(f.Path) != ".md" {
			continue
		}
		dir := filepath.Dir(f.Path)
		isStandardCmd := dir == filepath.Join(".claude", "commands")
		isPluginCmd := f.PluginDir != "" && isUnderPluginSubdir(f.Path, f.PluginDir, "commands")
		isPluginSkill := f.PluginDir != "" && isUnderPluginSubdir(f.Path, f.PluginDir, "skills")
		if !isStandardCmd && !isPluginCmd && !isPluginSkill {
			continue
		}

		// Extract YAML frontmatter (between --- delimiters)
		frontmatter := extractFrontmatter(f.Content)
		if frontmatter == nil {
			continue // No frontmatter is fine for commands
		}

		var fm map[string]any
		if err := yaml.Unmarshal(frontmatter, &fm); err != nil {
			diags = append(diags, mold.Diagnostic{
				Severity: mold.SeverityError,
				Message:  fmt.Sprintf("invalid frontmatter YAML: %v", err),
				File:     f.Path,
				Rule:     r.Name(),
			})
			continue
		}

		// Check for multiline values in fields Claude Code requires to be single-line.
		// A YAML parser happily accepts indented block scalars; Claude Code does not.
		// We inspect both the parsed value (catches literal "|" blocks) and the raw
		// frontmatter text (catches folded plain scalars where the YAML parser joins
		// continuation lines with spaces, hiding the newlines).
		singleLineFields := []string{"description", "name", "model", "argument-hint"}
		for _, key := range singleLineFields {
			val, ok := fm[key]
			if !ok {
				continue
			}
			isMultiline := false
			if s, ok := val.(string); ok && strings.Contains(s, "\n") {
				isMultiline = true
			}
			if !isMultiline && isMultilineInRaw(frontmatter, key) {
				isMultiline = true
			}
			if isMultiline {
				diags = append(diags, mold.Diagnostic{
					Severity: mold.SeverityError,
					Message:  fmt.Sprintf("frontmatter field %q must be a single-line string (multiline values may not be parsed correctly)", key),
					Tip:      "keep the value on the same line as the key: `" + key + ": your text here`",
					File:     f.Path,
					Rule:     r.Name(),
				})
			}
		}

		// Collect all unknown fields for this file and emit a single diagnostic.
		var unknown []string
		for key := range fm {
			if !allowed[key] {
				unknown = append(unknown, key)
			}
		}
		if len(unknown) > 0 {
			sort.Strings(unknown)
			diags = append(diags, mold.Diagnostic{
				Severity: mold.SeverityWarning,
				Message:  fmt.Sprintf("unknown command frontmatter fields: %s", strings.Join(unknown, ", ")),
				Tip:      "if these are intentional custom fields, run: ailloy lint --fix  (or: ailloy config allow-fields " + strings.Join(unknown, " ") + ")",
				File:     f.Path,
				Rule:     r.Name(),
				FixData:  map[string]any{"fields": unknown},
			})
		}
	}
	return diags
}

// settingsSchemaRule validates .claude/settings.json structure.
type settingsSchemaRule struct{}

func (r *settingsSchemaRule) Name() string                       { return "settings-schema" }
func (r *settingsSchemaRule) DefaultSeverity() mold.DiagSeverity { return mold.SeverityError }
func (r *settingsSchemaRule) Platforms() []Platform              { return []Platform{PlatformClaude} }

func (r *settingsSchemaRule) Check(ctx *RuleContext) []mold.Diagnostic {
	var diags []mold.Diagnostic
	for _, f := range ctx.Files {
		if f.Path != filepath.Join(".claude", "settings.json") {
			continue
		}

		var settings map[string]any
		if err := json.Unmarshal(f.Content, &settings); err != nil {
			diags = append(diags, mold.Diagnostic{
				Severity: mold.SeverityError,
				Message:  fmt.Sprintf("invalid JSON: %v", err),
				File:     f.Path,
				Rule:     r.Name(),
			})
			continue
		}

		// Validate hooks if present
		if hooks, ok := settings["hooks"]; ok {
			if hooksMap, ok := hooks.(map[string]any); ok {
				validEvents := map[string]bool{
					"PreToolUse": true, "PostToolUse": true,
					"Notification": true, "Stop": true,
					"SubagentStop": true, "SubagentTool": true,
				}
				for event := range hooksMap {
					if !validEvents[event] {
						diags = append(diags, mold.Diagnostic{
							Severity: mold.SeverityWarning,
							Message:  fmt.Sprintf("unknown hook event type: %s", event),
							File:     f.Path,
							Rule:     r.Name(),
						})
					}
				}
			}
		}
	}
	return diags
}

// pluginManifestRule validates .claude-plugin/plugin.json in Claude plugin directories.
type pluginManifestRule struct{}

func (r *pluginManifestRule) Name() string                       { return "plugin-manifest" }
func (r *pluginManifestRule) DefaultSeverity() mold.DiagSeverity { return mold.SeverityError }
func (r *pluginManifestRule) Platforms() []Platform              { return []Platform{PlatformClaude} }

func (r *pluginManifestRule) Check(ctx *RuleContext) []mold.Diagnostic {
	var diags []mold.Diagnostic
	for _, f := range ctx.Files {
		if f.PluginDir == "" {
			continue
		}
		if f.Path != filepath.Join(f.PluginDir, ".claude-plugin", "plugin.json") {
			continue
		}

		var manifest map[string]any
		if err := json.Unmarshal(f.Content, &manifest); err != nil {
			diags = append(diags, mold.Diagnostic{
				Severity: mold.SeverityError,
				Message:  fmt.Sprintf("invalid plugin manifest JSON: %v", err),
				File:     f.Path,
				Rule:     r.Name(),
			})
			continue
		}

		for _, field := range []string{"name", "version", "description"} {
			if _, ok := manifest[field]; !ok {
				diags = append(diags, mold.Diagnostic{
					Severity: mold.SeverityError,
					Message:  fmt.Sprintf("plugin manifest missing required field: %s", field),
					File:     f.Path,
					Rule:     r.Name(),
				})
			}
		}
	}
	return diags
}

// pluginHooksRule validates hooks/*.json files in Claude plugin directories.
type pluginHooksRule struct{}

func (r *pluginHooksRule) Name() string                       { return "plugin-hooks" }
func (r *pluginHooksRule) DefaultSeverity() mold.DiagSeverity { return mold.SeverityError }
func (r *pluginHooksRule) Platforms() []Platform              { return []Platform{PlatformClaude} }

func (r *pluginHooksRule) Check(ctx *RuleContext) []mold.Diagnostic {
	var diags []mold.Diagnostic
	for _, f := range ctx.Files {
		if f.PluginDir == "" || filepath.Ext(f.Path) != ".json" {
			continue
		}
		if !isUnderPluginSubdir(f.Path, f.PluginDir, "hooks") {
			continue
		}

		var hooks map[string]any
		if err := json.Unmarshal(f.Content, &hooks); err != nil {
			diags = append(diags, mold.Diagnostic{
				Severity: mold.SeverityError,
				Message:  fmt.Sprintf("invalid hooks JSON: %v", err),
				File:     f.Path,
				Rule:     r.Name(),
			})
			continue
		}

		hooksVal, ok := hooks["hooks"]
		if !ok {
			diags = append(diags, mold.Diagnostic{
				Severity: mold.SeverityWarning,
				Message:  "hooks file missing top-level \"hooks\" array",
				File:     f.Path,
				Rule:     r.Name(),
			})
			continue
		}

		hooksList, ok := hooksVal.([]any)
		if !ok {
			diags = append(diags, mold.Diagnostic{
				Severity: mold.SeverityError,
				Message:  "\"hooks\" must be an array",
				File:     f.Path,
				Rule:     r.Name(),
			})
			continue
		}

		for i, item := range hooksList {
			hook, ok := item.(map[string]any)
			if !ok {
				continue
			}
			for _, field := range []string{"name", "event"} {
				if _, ok := hook[field]; !ok {
					diags = append(diags, mold.Diagnostic{
						Severity: mold.SeverityError,
						Message:  fmt.Sprintf("hook[%d] missing required field: %s", i, field),
						File:     f.Path,
						Rule:     r.Name(),
					})
				}
			}
		}
	}
	return diags
}

// descriptionLengthRule warns when a description field is too long.
type descriptionLengthRule struct{}

func (r *descriptionLengthRule) Name() string                       { return "description-length" }
func (r *descriptionLengthRule) DefaultSeverity() mold.DiagSeverity { return mold.SeverityError }
func (r *descriptionLengthRule) Platforms() []Platform              { return []Platform{PlatformClaude} }

const defaultMaxDescriptionLength = 100

func (r *descriptionLengthRule) Check(ctx *RuleContext) []mold.Diagnostic {
	maxLen := defaultMaxDescriptionLength
	if v := ctx.Config.RuleOption(r.Name(), "max-length", nil); v != nil {
		switch n := v.(type) {
		case int:
			maxLen = n
		case uint64:
			if n <= uint64(maxInt) {
				maxLen = int(n)
			}
		case float64:
			maxLen = int(n)
		}
	}

	var diags []mold.Diagnostic
	for _, f := range ctx.Files {
		if f.Platform != PlatformClaude {
			continue
		}

		var desc string
		var matched bool

		ext := filepath.Ext(f.Path)

		switch {
		// Agent YAML files
		case (ext == ".yml" || ext == ".yaml") && r.isAgentFile(f):
			var agent map[string]any
			if err := yaml.Unmarshal(f.Content, &agent); err != nil {
				continue
			}
			if d, ok := agent["description"].(string); ok {
				desc = d
				matched = true
			}

		// Command/skill markdown files with frontmatter
		case ext == ".md" && r.isCommandOrSkillFile(f):
			frontmatter := extractFrontmatter(f.Content)
			if frontmatter == nil {
				continue
			}
			var fm map[string]any
			if err := yaml.Unmarshal(frontmatter, &fm); err != nil {
				continue
			}
			if d, ok := fm["description"].(string); ok {
				desc = d
				matched = true
			}

		// Plugin manifest
		case f.PluginDir != "" && ext == ".json" &&
			f.Path == filepath.Join(f.PluginDir, ".claude-plugin", "plugin.json"):
			var manifest map[string]any
			if err := json.Unmarshal(f.Content, &manifest); err != nil {
				continue
			}
			if d, ok := manifest["description"].(string); ok {
				desc = d
				matched = true
			}
		}

		if matched && len(desc) > maxLen {
			diags = append(diags, mold.Diagnostic{
				Severity: r.DefaultSeverity(),
				Message:  fmt.Sprintf("description is %d characters (max %d); long descriptions are truncated or ignored by AI tools", len(desc), maxLen),
				Tip:      "keep descriptions concise — aim for a single short sentence; see https://platform.claude.com/docs/en/agents-and-tools/agent-skills/best-practices#writing-effective-descriptions",
				File:     f.Path,
				Rule:     r.Name(),
			})
		}
	}
	return diags
}

func (r *descriptionLengthRule) isAgentFile(f DetectedFile) bool {
	dir := filepath.Dir(f.Path)
	return dir == filepath.Join(".claude", "agents") ||
		(f.PluginDir != "" && isUnderPluginSubdir(f.Path, f.PluginDir, "agents"))
}

func (r *descriptionLengthRule) isCommandOrSkillFile(f DetectedFile) bool {
	dir := filepath.Dir(f.Path)
	return dir == filepath.Join(".claude", "commands") ||
		(f.PluginDir != "" && isUnderPluginSubdir(f.Path, f.PluginDir, "commands")) ||
		(f.PluginDir != "" && isUnderPluginSubdir(f.Path, f.PluginDir, "skills"))
}

// descriptionPointOfViewRule warns when a description uses first or second person.
type descriptionPointOfViewRule struct{}

func (r *descriptionPointOfViewRule) Name() string { return "description-point-of-view" }
func (r *descriptionPointOfViewRule) DefaultSeverity() mold.DiagSeverity {
	return mold.SeverityWarning
}
func (r *descriptionPointOfViewRule) Platforms() []Platform { return []Platform{PlatformClaude} }

var povPatterns = regexp.MustCompile(`(?i)(?:^|\s)(?:I can|I will|I help|I am|I'm|you can|you should|you will|your )`)

func (r *descriptionPointOfViewRule) Check(ctx *RuleContext) []mold.Diagnostic {
	var diags []mold.Diagnostic
	for _, f := range ctx.Files {
		desc, ok := extractDescription(f)
		if !ok || desc == "" {
			continue
		}
		if povPatterns.MatchString(desc) {
			diags = append(diags, mold.Diagnostic{
				Severity: r.DefaultSeverity(),
				Message:  "description uses first or second person; write in third person for reliable skill discovery",
				Tip:      `write "Processes Excel files" instead of "I can help you process Excel files"; see https://platform.claude.com/docs/en/agents-and-tools/agent-skills/best-practices#writing-effective-descriptions`,
				File:     f.Path,
				Rule:     r.Name(),
			})
		}
	}
	return diags
}

// descriptionMissingTriggerRule suggests adding a "when to use" clause to descriptions.
type descriptionMissingTriggerRule struct{}

func (r *descriptionMissingTriggerRule) Name() string { return "description-missing-trigger" }
func (r *descriptionMissingTriggerRule) DefaultSeverity() mold.DiagSeverity {
	return mold.SeveritySuggestion
}
func (r *descriptionMissingTriggerRule) Platforms() []Platform { return []Platform{PlatformClaude} }

var triggerPhrases = regexp.MustCompile(`(?i)(?:use when|trigger when|use for|use if|use this when|invoke when|run when|activate when)`)

func (r *descriptionMissingTriggerRule) Check(ctx *RuleContext) []mold.Diagnostic {
	var diags []mold.Diagnostic
	for _, f := range ctx.Files {
		desc, ok := extractDescription(f)
		if !ok || desc == "" {
			continue
		}
		if !triggerPhrases.MatchString(desc) {
			diags = append(diags, mold.Diagnostic{
				Severity: r.DefaultSeverity(),
				Message:  "description says what the skill does but not when to use it",
				Tip:      `add a trigger clause like "Use when..." so the agent knows when to select this skill; see https://platform.claude.com/docs/en/agents-and-tools/agent-skills/best-practices#writing-effective-descriptions`,
				File:     f.Path,
				Rule:     r.Name(),
			})
		}
	}
	return diags
}

// nameFormatRule validates that skill/command/agent names follow platform requirements.
type nameFormatRule struct{}

func (r *nameFormatRule) Name() string                       { return "name-format" }
func (r *nameFormatRule) DefaultSeverity() mold.DiagSeverity { return mold.SeverityError }
func (r *nameFormatRule) Platforms() []Platform              { return []Platform{PlatformClaude} }

// validNameRegex allows lowercase alphanumeric and single hyphens, not at start or end.
var validNameRegex = regexp.MustCompile(`^[a-z0-9]([a-z0-9]|-[a-z0-9])*$`)

var consecutiveHyphens = regexp.MustCompile(`--`)

const maxNameLength = 64

func (r *nameFormatRule) Check(ctx *RuleContext) []mold.Diagnostic {
	var diags []mold.Diagnostic
	for _, f := range ctx.Files {
		name, ok := extractName(f)
		if !ok || name == "" {
			continue
		}
		if len(name) > maxNameLength {
			diags = append(diags, mold.Diagnostic{
				Severity: r.DefaultSeverity(),
				Message:  fmt.Sprintf("name is %d characters (max %d)", len(name), maxNameLength),
				Tip:      "see https://agentskills.io/specification#name-field",
				File:     f.Path,
				Rule:     r.Name(),
			})
		} else if strings.HasSuffix(name, "-") {
			diags = append(diags, mold.Diagnostic{
				Severity: r.DefaultSeverity(),
				Message:  fmt.Sprintf("name %q must not end with a hyphen", name),
				Tip:      "see https://agentskills.io/specification#name-field",
				File:     f.Path,
				Rule:     r.Name(),
			})
		} else if consecutiveHyphens.MatchString(name) {
			diags = append(diags, mold.Diagnostic{
				Severity: r.DefaultSeverity(),
				Message:  fmt.Sprintf("name %q must not contain consecutive hyphens", name),
				Tip:      "see https://agentskills.io/specification#name-field",
				File:     f.Path,
				Rule:     r.Name(),
			})
		} else if !validNameRegex.MatchString(name) {
			diags = append(diags, mold.Diagnostic{
				Severity: r.DefaultSeverity(),
				Message:  fmt.Sprintf("name %q must contain only lowercase letters, numbers, and hyphens", name),
				Tip:      "see https://agentskills.io/specification#name-field",
				File:     f.Path,
				Rule:     r.Name(),
			})
		}
	}
	return diags
}

// nameReservedWordsRule errors when a name contains reserved words.
type nameReservedWordsRule struct{}

func (r *nameReservedWordsRule) Name() string                       { return "name-reserved-words" }
func (r *nameReservedWordsRule) DefaultSeverity() mold.DiagSeverity { return mold.SeverityError }
func (r *nameReservedWordsRule) Platforms() []Platform              { return []Platform{PlatformClaude} }

var reservedWords = []string{"anthropic", "claude"}

func (r *nameReservedWordsRule) Check(ctx *RuleContext) []mold.Diagnostic {
	var diags []mold.Diagnostic
	for _, f := range ctx.Files {
		name, ok := extractName(f)
		if !ok || name == "" {
			continue
		}
		lower := strings.ToLower(name)
		for _, word := range reservedWords {
			if strings.Contains(lower, word) {
				diags = append(diags, mold.Diagnostic{
					Severity: r.DefaultSeverity(),
					Message:  fmt.Sprintf("name %q contains reserved word %q", name, word),
					Tip:      "names cannot contain \"anthropic\" or \"claude\"; see https://platform.claude.com/docs/en/agents-and-tools/agent-skills/best-practices#naming-conventions",
					File:     f.Path,
					Rule:     r.Name(),
				})
				break
			}
		}
	}
	return diags
}

// vagueNameRule warns when a name is overly generic.
type vagueNameRule struct{}

func (r *vagueNameRule) Name() string                       { return "vague-name" }
func (r *vagueNameRule) DefaultSeverity() mold.DiagSeverity { return mold.SeverityWarning }
func (r *vagueNameRule) Platforms() []Platform              { return []Platform{PlatformClaude} }

var vagueNames = map[string]bool{
	"helper": true, "helpers": true,
	"utils": true, "util": true, "utility": true, "utilities": true,
	"tools": true, "tool": true,
	"misc": true, "other": true, "general": true,
	"documents": true, "data": true, "files": true, "stuff": true,
}

func (r *vagueNameRule) Check(ctx *RuleContext) []mold.Diagnostic {
	var diags []mold.Diagnostic
	for _, f := range ctx.Files {
		name, ok := extractName(f)
		if !ok || name == "" {
			continue
		}
		if vagueNames[strings.ToLower(name)] {
			diags = append(diags, mold.Diagnostic{
				Severity: r.DefaultSeverity(),
				Message:  fmt.Sprintf("name %q is too generic for reliable skill discovery", name),
				Tip:      "use a specific, descriptive name like \"processing-pdfs\" or \"managing-databases\"; see https://platform.claude.com/docs/en/agents-and-tools/agent-skills/best-practices#naming-conventions",
				File:     f.Path,
				Rule:     r.Name(),
			})
		}
	}
	return diags
}

// skillBodyLengthRule warns when a SKILL.md body exceeds a line threshold.
type skillBodyLengthRule struct{}

func (r *skillBodyLengthRule) Name() string                       { return "skill-body-length" }
func (r *skillBodyLengthRule) DefaultSeverity() mold.DiagSeverity { return mold.SeverityWarning }
func (r *skillBodyLengthRule) Platforms() []Platform              { return []Platform{PlatformClaude} }

const defaultMaxSkillBodyLines = 500

func (r *skillBodyLengthRule) Check(ctx *RuleContext) []mold.Diagnostic {
	maxLines := defaultMaxSkillBodyLines
	if v := ctx.Config.RuleOption(r.Name(), "max-lines", nil); v != nil {
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
		if f.Platform != PlatformClaude {
			continue
		}
		// Only applies to SKILL.md files — reference files under skills/ are excluded
		// since they are the split targets recommended by progressive disclosure.
		if filepath.Base(f.Path) != "SKILL.md" {
			continue
		}
		if !isSkillPath(f) {
			continue
		}

		// Count lines in the body (after frontmatter)
		body := f.Content
		if fm := extractFrontmatter(f.Content); fm != nil {
			// Skip past the closing --- of frontmatter
			idx := bytes.Index(f.Content[3:], []byte("\n---"))
			if idx >= 0 {
				body = f.Content[3+idx+4:] // skip "---\n" + frontmatter + "\n---"
			}
		}

		lines := bytes.Count(body, []byte("\n"))
		if len(body) > 0 && body[len(body)-1] != '\n' {
			lines++
		}
		if lines > maxLines {
			diags = append(diags, mold.Diagnostic{
				Severity: r.DefaultSeverity(),
				Message:  fmt.Sprintf("skill body is %d lines (max %d); split into separate files using progressive disclosure", lines, maxLines),
				Tip:      "keep SKILL.md under 500 lines and use references to additional files; see https://platform.claude.com/docs/en/agents-and-tools/agent-skills/best-practices#progressive-disclosure-patterns",
				File:     f.Path,
				Rule:     r.Name(),
			})
		}
	}
	return diags
}

// commandsDeprecatedRule warns when .claude/commands/ files are found,
// suggesting migration to the .claude/skills/<name>/SKILL.md format.
type commandsDeprecatedRule struct{}

func (r *commandsDeprecatedRule) Name() string { return "commands-deprecated" }
func (r *commandsDeprecatedRule) DefaultSeverity() mold.DiagSeverity {
	return mold.SeverityWarning
}
func (r *commandsDeprecatedRule) Platforms() []Platform { return []Platform{PlatformClaude} }

func (r *commandsDeprecatedRule) Check(ctx *RuleContext) []mold.Diagnostic {
	var diags []mold.Diagnostic
	for _, f := range ctx.Files {
		if f.Platform != PlatformClaude {
			continue
		}
		if filepath.Ext(f.Path) != ".md" {
			continue
		}
		dir := filepath.Dir(f.Path)
		isStandardCmd := dir == filepath.Join(".claude", "commands")
		isPluginCmd := f.PluginDir != "" && isUnderPluginSubdir(f.Path, f.PluginDir, "commands")
		if !isStandardCmd && !isPluginCmd {
			continue
		}

		base := strings.TrimSuffix(filepath.Base(f.Path), ".md")
		var suggestion string
		if isStandardCmd {
			suggestion = fmt.Sprintf(".claude/skills/%s/SKILL.md", base)
		} else {
			suggestion = fmt.Sprintf("%s/skills/%s/SKILL.md", f.PluginDir, base)
		}
		diags = append(diags, mold.Diagnostic{
			Severity: r.DefaultSeverity(),
			Message:  fmt.Sprintf(".claude/commands/ is deprecated; migrate to %s", suggestion),
			Tip:      "see https://platform.claude.com/docs/en/agent-sdk/slash-commands#creating-custom-slash-commands and https://agentskills.io/specification",
			File:     f.Path,
			Rule:     r.Name(),
		})
	}
	return diags
}

// nameDirectoryMismatchRule errors when the name field doesn't match the parent directory.
type nameDirectoryMismatchRule struct{}

func (r *nameDirectoryMismatchRule) Name() string { return "name-directory-mismatch" }
func (r *nameDirectoryMismatchRule) DefaultSeverity() mold.DiagSeverity {
	return mold.SeverityError
}
func (r *nameDirectoryMismatchRule) Platforms() []Platform { return []Platform{PlatformClaude} }

func (r *nameDirectoryMismatchRule) Check(ctx *RuleContext) []mold.Diagnostic {
	var diags []mold.Diagnostic
	for _, f := range ctx.Files {
		if f.Platform != PlatformClaude {
			continue
		}
		if filepath.Ext(f.Path) != ".md" {
			continue
		}
		// Only applies to skill directories (.claude/skills/<name>/SKILL.md or plugin skills/)
		if !isSkillPath(f) {
			continue
		}

		name, ok := extractName(f)
		if !ok || name == "" {
			continue
		}

		parentDir := filepath.Base(filepath.Dir(f.Path))
		if name != parentDir {
			diags = append(diags, mold.Diagnostic{
				Severity: r.DefaultSeverity(),
				Message:  fmt.Sprintf("name %q does not match parent directory %q", name, parentDir),
				Tip:      "the name field must match the skill's directory name; see https://agentskills.io/specification#name-field",
				File:     f.Path,
				Rule:     r.Name(),
			})
		}
	}
	return diags
}

// descriptionMaxLengthRule errors when a description exceeds the platform maximum of 1024 characters.
type descriptionMaxLengthRule struct{}

func (r *descriptionMaxLengthRule) Name() string { return "description-max-length" }
func (r *descriptionMaxLengthRule) DefaultSeverity() mold.DiagSeverity {
	return mold.SeverityError
}
func (r *descriptionMaxLengthRule) Platforms() []Platform { return []Platform{PlatformClaude} }

const platformMaxDescriptionLength = 1024

func (r *descriptionMaxLengthRule) Check(ctx *RuleContext) []mold.Diagnostic {
	var diags []mold.Diagnostic
	for _, f := range ctx.Files {
		desc, ok := extractDescription(f)
		if !ok || desc == "" {
			continue
		}
		if len(desc) > platformMaxDescriptionLength {
			diags = append(diags, mold.Diagnostic{
				Severity: r.DefaultSeverity(),
				Message:  fmt.Sprintf("description is %d characters (platform max %d); skill will fail to register", len(desc), platformMaxDescriptionLength),
				Tip:      "see https://agentskills.io/specification#description-field",
				File:     f.Path,
				Rule:     r.Name(),
			})
		}
	}
	return diags
}

// compatibilityLengthRule errors when the compatibility field exceeds 500 characters.
type compatibilityLengthRule struct{}

func (r *compatibilityLengthRule) Name() string                       { return "compatibility-length" }
func (r *compatibilityLengthRule) DefaultSeverity() mold.DiagSeverity { return mold.SeverityError }
func (r *compatibilityLengthRule) Platforms() []Platform              { return []Platform{PlatformClaude} }

const maxCompatibilityLength = 500

func (r *compatibilityLengthRule) Check(ctx *RuleContext) []mold.Diagnostic {
	var diags []mold.Diagnostic
	for _, f := range ctx.Files {
		if f.Platform != PlatformClaude {
			continue
		}
		if filepath.Ext(f.Path) != ".md" {
			continue
		}
		if !isCommandOrSkillPath(f) {
			continue
		}

		fm := extractFrontmatter(f.Content)
		if fm == nil {
			continue
		}
		var m map[string]any
		if err := yaml.Unmarshal(fm, &m); err != nil {
			continue
		}
		compat, ok := m["compatibility"].(string)
		if !ok || compat == "" {
			continue
		}
		if len(compat) > maxCompatibilityLength {
			diags = append(diags, mold.Diagnostic{
				Severity: r.DefaultSeverity(),
				Message:  fmt.Sprintf("compatibility field is %d characters (max %d)", len(compat), maxCompatibilityLength),
				Tip:      "see https://agentskills.io/specification#compatibility-field",
				File:     f.Path,
				Rule:     r.Name(),
			})
		}
	}
	return diags
}

// skillTokenBudgetRule warns when a skill body likely exceeds 5000 tokens.
type skillTokenBudgetRule struct{}

func (r *skillTokenBudgetRule) Name() string                       { return "skill-token-budget" }
func (r *skillTokenBudgetRule) DefaultSeverity() mold.DiagSeverity { return mold.SeverityWarning }
func (r *skillTokenBudgetRule) Platforms() []Platform              { return []Platform{PlatformClaude} }

const defaultMaxSkillTokens = 5000
const estimatedCharsPerToken = 4

func (r *skillTokenBudgetRule) Check(ctx *RuleContext) []mold.Diagnostic {
	maxTokens := defaultMaxSkillTokens
	if v := ctx.Config.RuleOption(r.Name(), "max-tokens", nil); v != nil {
		switch n := v.(type) {
		case int:
			maxTokens = n
		case uint64:
			if n <= uint64(maxInt) {
				maxTokens = int(n)
			}
		case float64:
			maxTokens = int(n)
		}
	}

	var diags []mold.Diagnostic
	for _, f := range ctx.Files {
		if f.Platform != PlatformClaude {
			continue
		}
		// Only applies to SKILL.md files — reference files are excluded.
		if filepath.Base(f.Path) != "SKILL.md" {
			continue
		}
		if !isSkillPath(f) {
			continue
		}

		// Extract body after frontmatter
		body := f.Content
		if fm := extractFrontmatter(f.Content); fm != nil {
			idx := bytes.Index(f.Content[3:], []byte("\n---"))
			if idx >= 0 {
				body = f.Content[3+idx+4:]
			}
		}

		estimatedTokens := len(body) / estimatedCharsPerToken
		if estimatedTokens > maxTokens {
			diags = append(diags, mold.Diagnostic{
				Severity: r.DefaultSeverity(),
				Message:  fmt.Sprintf("skill body is ~%d tokens (recommended max %d); move reference material to separate files", estimatedTokens, maxTokens),
				Tip:      "see https://agentskills.io/specification#progressive-disclosure and https://agentskills.io/skill-creation/best-practices#spending-context-wisely",
				File:     f.Path,
				Rule:     r.Name(),
			})
		}
	}
	return diags
}

// descriptionImperativeRule suggests using imperative phrasing in descriptions.
type descriptionImperativeRule struct{}

func (r *descriptionImperativeRule) Name() string { return "description-imperative" }
func (r *descriptionImperativeRule) DefaultSeverity() mold.DiagSeverity {
	return mold.SeveritySuggestion
}
func (r *descriptionImperativeRule) Platforms() []Platform { return []Platform{PlatformClaude} }

var declarativePrefixes = regexp.MustCompile(`(?i)^(?:this skill |this tool |this command |this agent |a skill that |a tool that |a command that |an agent that )`)

func (r *descriptionImperativeRule) Check(ctx *RuleContext) []mold.Diagnostic {
	var diags []mold.Diagnostic
	for _, f := range ctx.Files {
		desc, ok := extractDescription(f)
		if !ok || desc == "" {
			continue
		}
		if declarativePrefixes.MatchString(desc) {
			diags = append(diags, mold.Diagnostic{
				Severity: r.DefaultSeverity(),
				Message:  "description uses declarative phrasing; prefer imperative form for better skill triggering",
				Tip:      `write "Analyzes CSV files..." or "Use this skill when..." instead of "This skill does..."; see https://agentskills.io/skill-creation/optimizing-descriptions#writing-effective-descriptions`,
				File:     f.Path,
				Rule:     r.Name(),
			})
		}
	}
	return diags
}

// isSkillPath returns true if the file is in a skills directory (not commands).
// Works for any AI tool that supports the Agent Skills spec, not just .claude/.
func isSkillPath(f DetectedFile) bool {
	// Match any path containing skills/<name>/<file> (at least one subdirectory after skills/)
	parts := strings.Split(filepath.ToSlash(f.Path), "/")
	for i, p := range parts {
		if p == "skills" && i+2 <= len(parts)-1 {
			return true
		}
	}
	return false
}

// extractDescription extracts the description field from a detected file.
// Returns the description and true if found, or empty string and false if not applicable.
func extractDescription(f DetectedFile) (string, bool) {
	if f.Platform != PlatformClaude {
		return "", false
	}
	ext := filepath.Ext(f.Path)
	switch {
	case (ext == ".yml" || ext == ".yaml") && isAgentPath(f):
		var agent map[string]any
		if err := yaml.Unmarshal(f.Content, &agent); err != nil {
			return "", false
		}
		if d, ok := agent["description"].(string); ok {
			return d, true
		}
	case ext == ".md" && isCommandOrSkillPath(f):
		fm := extractFrontmatter(f.Content)
		if fm == nil {
			return "", false
		}
		var m map[string]any
		if err := yaml.Unmarshal(fm, &m); err != nil {
			return "", false
		}
		if d, ok := m["description"].(string); ok {
			return d, true
		}
	case f.PluginDir != "" && ext == ".json" &&
		f.Path == filepath.Join(f.PluginDir, ".claude-plugin", "plugin.json"):
		var manifest map[string]any
		if err := json.Unmarshal(f.Content, &manifest); err != nil {
			return "", false
		}
		if d, ok := manifest["description"].(string); ok {
			return d, true
		}
	}
	return "", false
}

// extractName extracts the name field from a detected file.
func extractName(f DetectedFile) (string, bool) {
	if f.Platform != PlatformClaude {
		return "", false
	}
	ext := filepath.Ext(f.Path)
	switch {
	case (ext == ".yml" || ext == ".yaml") && isAgentPath(f):
		var agent map[string]any
		if err := yaml.Unmarshal(f.Content, &agent); err != nil {
			return "", false
		}
		if n, ok := agent["name"].(string); ok {
			return n, true
		}
	case ext == ".md" && isCommandOrSkillPath(f):
		fm := extractFrontmatter(f.Content)
		if fm == nil {
			return "", false
		}
		var m map[string]any
		if err := yaml.Unmarshal(fm, &m); err != nil {
			return "", false
		}
		if n, ok := m["name"].(string); ok {
			return n, true
		}
	case f.PluginDir != "" && ext == ".json" &&
		f.Path == filepath.Join(f.PluginDir, ".claude-plugin", "plugin.json"):
		var manifest map[string]any
		if err := json.Unmarshal(f.Content, &manifest); err != nil {
			return "", false
		}
		if n, ok := manifest["name"].(string); ok {
			return n, true
		}
	}
	return "", false
}

// isAgentPath returns true if the file is in an agent directory.
func isAgentPath(f DetectedFile) bool {
	dir := filepath.Dir(f.Path)
	return dir == filepath.Join(".claude", "agents") ||
		(f.PluginDir != "" && isUnderPluginSubdir(f.Path, f.PluginDir, "agents"))
}

// isCommandOrSkillPath returns true if the file is in a command or skill directory.
func isCommandOrSkillPath(f DetectedFile) bool {
	dir := filepath.Dir(f.Path)
	if dir == filepath.Join(".claude", "commands") {
		return true
	}
	if f.PluginDir != "" && isUnderPluginSubdir(f.Path, f.PluginDir, "commands") {
		return true
	}
	if f.PluginDir != "" && isUnderPluginSubdir(f.Path, f.PluginDir, "skills") {
		return true
	}
	// Standard skills: .claude/skills/<name>/*.md
	if isSkillPath(f) {
		return true
	}
	return false
}

// isMultilineInRaw reports whether field has a block or folded value in raw YAML frontmatter.
// It catches both block scalars (|, >) and plain scalars where the value begins on the next
// indented line — cases where the YAML parser folds newlines to spaces, hiding the multiline nature.
func isMultilineInRaw(frontmatter []byte, field string) bool {
	prefix := []byte(field + ":")
	scanner := bufio.NewScanner(bytes.NewReader(frontmatter))
	pending := false
	for scanner.Scan() {
		line := scanner.Bytes()
		if pending {
			if len(line) > 0 && (line[0] == ' ' || line[0] == '\t') {
				return true
			}
			return false
		}
		if bytes.HasPrefix(line, prefix) {
			rest := bytes.TrimSpace(line[len(prefix):])
			if len(rest) == 0 || rest[0] == '|' || rest[0] == '>' {
				pending = true
			}
		}
	}
	return false
}

// extractFrontmatter extracts YAML frontmatter from markdown content.
func extractFrontmatter(content []byte) []byte {
	trimmed := bytes.TrimSpace(content)
	if !bytes.HasPrefix(trimmed, []byte("---")) {
		return nil
	}
	// Find closing ---
	rest := trimmed[3:]
	idx := bytes.Index(rest, []byte("\n---"))
	if idx < 0 {
		return nil
	}
	return bytes.TrimSpace(rest[:idx])
}
