package assay

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
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
					Message:  fmt.Sprintf("frontmatter field %q must be a single-line string (multiline values are not parsed by Claude Code)", key),
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
