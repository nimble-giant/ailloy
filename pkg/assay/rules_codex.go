package assay

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/nimble-giant/ailloy/pkg/mold"
	"github.com/pelletier/go-toml/v2"
)

func init() {
	Register(&codexSkillFrontmatterRule{})
	Register(&codexAgentSchemaRule{})
}

// codexSkillFrontmatterRule validates the required Agent Skills fields for
// repository skills loaded by Codex from .agents/skills/<name>/SKILL.md.
type codexSkillFrontmatterRule struct{}

func (r *codexSkillFrontmatterRule) Name() string { return "codex-skill-frontmatter" }
func (r *codexSkillFrontmatterRule) DefaultSeverity() mold.DiagSeverity {
	return mold.SeverityError
}
func (r *codexSkillFrontmatterRule) Platforms() []Platform {
	return []Platform{PlatformCodex}
}

func (r *codexSkillFrontmatterRule) Check(ctx *RuleContext) []mold.Diagnostic {
	var diags []mold.Diagnostic
	for _, f := range ctx.Files {
		if f.Platform != PlatformCodex || filepath.Base(f.Path) != "SKILL.md" || !isSkillPath(f) {
			continue
		}

		frontmatter := extractFrontmatter(f.Content)
		if frontmatter == nil {
			diags = append(diags, mold.Diagnostic{
				Severity: r.DefaultSeverity(),
				Message:  "Codex skill is missing YAML frontmatter",
				Tip:      "start SKILL.md with frontmatter containing non-empty `name` and `description` fields",
				File:     f.Path,
				Rule:     r.Name(),
			})
			continue
		}

		var metadata map[string]any
		if err := yaml.Unmarshal(frontmatter, &metadata); err != nil {
			diags = append(diags, mold.Diagnostic{
				Severity: r.DefaultSeverity(),
				Message:  fmt.Sprintf("invalid skill frontmatter YAML: %v", err),
				File:     f.Path,
				Rule:     r.Name(),
			})
			continue
		}

		for _, field := range []string{"name", "description"} {
			value, ok := metadata[field]
			text, isString := value.(string)
			switch {
			case !ok:
				diags = append(diags, mold.Diagnostic{
					Severity: r.DefaultSeverity(),
					Message:  fmt.Sprintf("Codex skill missing required frontmatter field: %s", field),
					File:     f.Path,
					Rule:     r.Name(),
				})
			case !isString || strings.TrimSpace(text) == "":
				diags = append(diags, mold.Diagnostic{
					Severity: r.DefaultSeverity(),
					Message:  fmt.Sprintf("Codex skill frontmatter field %q must be a non-empty string", field),
					File:     f.Path,
					Rule:     r.Name(),
				})
			}
		}
	}
	return diags
}

// codexAgentSchemaRule validates project-scoped Codex custom agents at
// .codex/agents/<name>.toml.
type codexAgentSchemaRule struct{}

func (r *codexAgentSchemaRule) Name() string { return "codex-agent-schema" }
func (r *codexAgentSchemaRule) DefaultSeverity() mold.DiagSeverity {
	return mold.SeverityError
}
func (r *codexAgentSchemaRule) Platforms() []Platform {
	return []Platform{PlatformCodex}
}

func (r *codexAgentSchemaRule) Check(ctx *RuleContext) []mold.Diagnostic {
	var diags []mold.Diagnostic
	for _, f := range ctx.Files {
		if f.Platform != PlatformCodex ||
			filepath.Dir(f.Path) != filepath.Join(".codex", "agents") ||
			filepath.Ext(f.Path) != ".toml" {
			continue
		}

		var agent map[string]any
		if err := toml.Unmarshal(f.Content, &agent); err != nil {
			diags = append(diags, mold.Diagnostic{
				Severity: r.DefaultSeverity(),
				Message:  fmt.Sprintf("invalid Codex agent TOML: %v", err),
				File:     f.Path,
				Rule:     r.Name(),
			})
			continue
		}

		for _, field := range []string{"name", "description", "developer_instructions"} {
			value, ok := agent[field]
			text, isString := value.(string)
			switch {
			case !ok:
				diags = append(diags, mold.Diagnostic{
					Severity: r.DefaultSeverity(),
					Message:  fmt.Sprintf("Codex agent missing required field: %s", field),
					File:     f.Path,
					Rule:     r.Name(),
				})
			case !isString || strings.TrimSpace(text) == "":
				diags = append(diags, mold.Diagnostic{
					Severity: r.DefaultSeverity(),
					Message:  fmt.Sprintf("Codex agent field %q must be a non-empty string", field),
					File:     f.Path,
					Rule:     r.Name(),
				})
			}
		}
	}
	return diags
}
