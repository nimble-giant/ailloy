package assay

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/goccy/go-yaml"
	"github.com/nimble-giant/ailloy/pkg/mold"
)

func init() {
	Register(&agentFrontmatterRule{})
	Register(&commandFrontmatterRule{})
	Register(&settingsSchemaRule{})
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
		// Match .claude/agents/*.yml or .claude/agents/*.yaml
		dir := filepath.Dir(f.Path)
		ext := filepath.Ext(f.Path)
		if dir != filepath.Join(".claude", "agents") || (ext != ".yml" && ext != ".yaml") {
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

// commandFrontmatterRule validates .claude/commands/*.md frontmatter.
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
	var diags []mold.Diagnostic
	for _, f := range ctx.Files {
		if f.Platform != PlatformClaude {
			continue
		}
		dir := filepath.Dir(f.Path)
		if dir != filepath.Join(".claude", "commands") || filepath.Ext(f.Path) != ".md" {
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

		for key := range fm {
			if !validCommandFields[key] {
				diags = append(diags, mold.Diagnostic{
					Severity: mold.SeverityWarning,
					Message:  fmt.Sprintf("unknown command frontmatter field: %s", key),
					File:     f.Path,
					Rule:     r.Name(),
				})
			}
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
