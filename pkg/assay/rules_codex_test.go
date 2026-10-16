package assay

import "testing"

func TestCodexSkillFrontmatterRule(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    int
	}{
		{
			name:    "valid",
			content: "---\nname: reviewing\ndescription: Reviews pull requests.\n---\n# Reviewing\n",
		},
		{
			name:    "missing frontmatter",
			content: "# Reviewing\n",
			want:    1,
		},
		{
			name:    "missing required fields",
			content: "---\nlicense: Apache-2.0\n---\n# Reviewing\n",
			want:    2,
		},
		{
			name:    "invalid YAML",
			content: "---\nname: [reviewing\ndescription: Reviews pull requests.\n---\n",
			want:    1,
		},
		{
			name:    "empty required fields",
			content: "---\nname: ''\ndescription: ''\n---\n# Reviewing\n",
			want:    2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &RuleContext{
				Files: []DetectedFile{{
					Path:     ".agents/skills/reviewing/SKILL.md",
					Platform: PlatformCodex,
					Content:  []byte(tt.content),
				}},
				Config: DefaultConfig(),
			}
			got := (&codexSkillFrontmatterRule{}).Check(ctx)
			if len(got) != tt.want {
				t.Errorf("diagnostics = %v, want %d", got, tt.want)
			}
		})
	}
}

func TestCodexAgentSchemaRule(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    int
	}{
		{
			name: "valid",
			content: `name = "reviewer"
description = "Reviews changes for correctness."
developer_instructions = """
Review the requested changes and cite concrete evidence.
"""
sandbox_mode = "read-only"
`,
		},
		{
			name:    "invalid TOML",
			content: `name = "reviewer`,
			want:    1,
		},
		{
			name:    "missing required fields",
			content: `name = "reviewer"`,
			want:    2,
		},
		{
			name: "empty required field",
			content: `name = ""
description = "Reviews changes."
developer_instructions = "Review."
`,
			want: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &RuleContext{
				Files: []DetectedFile{{
					Path:     ".codex/agents/reviewer.toml",
					Platform: PlatformCodex,
					Content:  []byte(tt.content),
				}},
				Config: DefaultConfig(),
			}
			got := (&codexAgentSchemaRule{}).Check(ctx)
			if len(got) != tt.want {
				t.Errorf("diagnostics = %v, want %d", got, tt.want)
			}
		})
	}
}

func TestAssayCodexSkillsAndAgents(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".agents/skills/reviewing/SKILL.md", "# missing frontmatter\n")
	writeFile(t, dir, ".codex/agents/reviewer.toml", `name = "reviewer"`)

	result, err := Assay(dir, &Config{Platforms: []string{"codex"}})
	if err != nil {
		t.Fatal(err)
	}

	rules := make(map[string]bool)
	for _, d := range result.Diagnostics {
		rules[d.Rule] = true
	}
	if !rules["codex-skill-frontmatter"] {
		t.Error("expected codex-skill-frontmatter diagnostic")
	}
	if !rules["codex-agent-schema"] {
		t.Error("expected codex-agent-schema diagnostic")
	}
}
