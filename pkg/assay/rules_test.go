package assay

import (
	"strings"
	"testing"

	"github.com/nimble-giant/ailloy/pkg/mold"
)

func TestLineCountRule(t *testing.T) {
	shortContent := "# Short\nLine 2\n"
	longContent := strings.Repeat("line\n", 200)

	tests := []struct {
		name     string
		content  string
		maxLines int
		wantDiag bool
	}{
		{"short file", shortContent, 150, false},
		{"long file", longContent, 150, true},
		{"custom threshold", longContent, 300, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			if tt.maxLines != 150 {
				cfg.Rules = map[string]RuleConfig{
					"line-count": {Options: map[string]any{"max-lines": tt.maxLines}},
				}
			}
			ctx := &RuleContext{
				Files:  []DetectedFile{{Path: "CLAUDE.md", Content: []byte(tt.content)}},
				Config: cfg,
			}
			rule := &lineCountRule{}
			diags := rule.Check(ctx)
			if tt.wantDiag && len(diags) == 0 {
				t.Error("expected diagnostic, got none")
			}
			if !tt.wantDiag && len(diags) > 0 {
				t.Errorf("expected no diagnostic, got: %v", diags)
			}
		})
	}
}

func TestStructureRule(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		wantDiag bool
	}{
		{"has headings", "# Title\nContent here\n", false},
		{"no headings", "Just plain text\nAnother line\n", true},
		{"empty file", "", false}, // handled by emptyFileRule
		{"h2 heading", "## Section\nContent\n", false},
		{"heading after frontmatter", "---\ntopic: foo\ncreated: 2026-03-12\n---\n\n# Title\nContent\n", false},
		{"no heading after frontmatter", "---\ntopic: foo\n---\n\nJust text.\n", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &RuleContext{
				Files:  []DetectedFile{{Path: "CLAUDE.md", Content: []byte(tt.content)}},
				Config: DefaultConfig(),
			}
			rule := &structureRule{}
			diags := rule.Check(ctx)
			if tt.wantDiag && len(diags) == 0 {
				t.Error("expected diagnostic, got none")
			}
			if !tt.wantDiag && len(diags) > 0 {
				t.Errorf("expected no diagnostic, got: %v", diags)
			}
		})
	}
}

func TestAgentsMDPresenceRule(t *testing.T) {
	tests := []struct {
		name     string
		files    []DetectedFile
		wantDiag bool
	}{
		{
			"has AGENTS.md",
			[]DetectedFile{
				{Path: "CLAUDE.md", Platform: PlatformClaude},
				{Path: "AGENTS.md", Platform: PlatformGeneric},
			},
			false,
		},
		{
			"no AGENTS.md with platform files",
			[]DetectedFile{
				{Path: "CLAUDE.md", Platform: PlatformClaude},
			},
			true,
		},
		{
			"no AGENTS.md and CLAUDE.md present — smart tip",
			[]DetectedFile{
				{Path: "CLAUDE.md", Platform: PlatformClaude},
				{Path: ".cursor/rules/style.md", Platform: PlatformCursor},
			},
			true,
		},
		{
			"only generic files",
			[]DetectedFile{
				{Path: "AGENTS.md", Platform: PlatformGeneric},
			},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &RuleContext{Files: tt.files, Config: DefaultConfig()}
			rule := &agentsMDPresenceRule{}
			diags := rule.Check(ctx)
			if tt.wantDiag && len(diags) == 0 {
				t.Error("expected diagnostic, got none")
			}
			if !tt.wantDiag && len(diags) > 0 {
				t.Errorf("expected no diagnostic, got: %v", diags)
			}
		})
	}

	// When CLAUDE.md is present the tip should mention it specifically.
	t.Run("tip mentions CLAUDE.md path", func(t *testing.T) {
		ctx := &RuleContext{
			Files: []DetectedFile{
				{Path: "CLAUDE.md", Platform: PlatformClaude},
			},
			Config: DefaultConfig(),
		}
		diags := (&agentsMDPresenceRule{}).Check(ctx)
		if len(diags) == 0 {
			t.Fatal("expected diagnostic")
		}
		if diags[0].Tip == "" {
			t.Error("expected non-empty tip")
		}
		if !containsSubstr(diags[0].Tip, "CLAUDE.md") {
			t.Errorf("expected tip to mention CLAUDE.md, got: %s", diags[0].Tip)
		}
		if !containsSubstr(diags[0].Tip, "@AGENTS.md") {
			t.Errorf("expected tip to mention @AGENTS.md, got: %s", diags[0].Tip)
		}
	})

	// Without CLAUDE.md the generic tip should be used instead.
	t.Run("generic tip without CLAUDE.md", func(t *testing.T) {
		ctx := &RuleContext{
			Files: []DetectedFile{
				{Path: ".cursor/rules/style.md", Platform: PlatformCursor},
			},
			Config: DefaultConfig(),
		}
		diags := (&agentsMDPresenceRule{}).Check(ctx)
		if len(diags) == 0 {
			t.Fatal("expected diagnostic")
		}
		if containsSubstr(diags[0].Tip, "CLAUDE.md") {
			t.Errorf("generic tip should not mention CLAUDE.md, got: %s", diags[0].Tip)
		}
	})
}

func TestCrossReferenceRule(t *testing.T) {
	tests := []struct {
		name     string
		files    []DetectedFile
		wantDiag bool
	}{
		{
			"has import",
			[]DetectedFile{
				{Path: "CLAUDE.md", Platform: PlatformClaude, Content: []byte("@AGENTS.md\n# Instructions")},
				{Path: "AGENTS.md", Platform: PlatformGeneric, Content: []byte("# Agents")},
			},
			false,
		},
		{
			"missing import",
			[]DetectedFile{
				{Path: "CLAUDE.md", Platform: PlatformClaude, Content: []byte("# Instructions only")},
				{Path: "AGENTS.md", Platform: PlatformGeneric, Content: []byte("# Agents")},
			},
			true,
		},
		{
			"no AGENTS.md",
			[]DetectedFile{
				{Path: "CLAUDE.md", Platform: PlatformClaude, Content: []byte("# Instructions")},
			},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &RuleContext{Files: tt.files, Config: DefaultConfig()}
			rule := &crossReferenceRule{}
			diags := rule.Check(ctx)
			if tt.wantDiag && len(diags) == 0 {
				t.Error("expected diagnostic, got none")
			}
			if !tt.wantDiag && len(diags) > 0 {
				t.Errorf("expected no diagnostic, got: %v", diags)
			}
		})
	}
}

func TestEmptyFileRule(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		wantDiag bool
	}{
		{"has content", "# Title\nContent here\n", false},
		{"empty", "", true},
		{"whitespace only", "   \n\t\n  ", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &RuleContext{
				Files:  []DetectedFile{{Path: "CLAUDE.md", Content: []byte(tt.content)}},
				Config: DefaultConfig(),
			}
			rule := &emptyFileRule{}
			diags := rule.Check(ctx)
			if tt.wantDiag && len(diags) == 0 {
				t.Error("expected diagnostic, got none")
			}
			if !tt.wantDiag && len(diags) > 0 {
				t.Errorf("expected no diagnostic, got: %v", diags)
			}
		})
	}
}

func TestDuplicateTopicsRule(t *testing.T) {
	similarContent := `Follow the conventional commit format for all messages.
Use imperative mood in the subject line. Keep the subject under 72 characters.
Include a body when the change is non-trivial. Reference issue numbers.`

	tests := []struct {
		name     string
		files    []DetectedFile
		wantDiag bool
	}{
		{
			"different headings",
			[]DetectedFile{
				{Path: "CLAUDE.md", Content: []byte("# Setup\nContent about setup")},
				{Path: "AGENTS.md", Content: []byte("# Testing\nContent about testing")},
			},
			false,
		},
		{
			"same heading different content",
			[]DetectedFile{
				{Path: "CLAUDE.md", Content: []byte("# Usage\nRun the linter with `ailloy assay .` to check files.")},
				{Path: "AGENTS.md", Content: []byte("# Usage\nInstall dependencies with `npm install` and start dev server.")},
			},
			false,
		},
		{
			"same heading similar content should warn",
			[]DetectedFile{
				{Path: "CLAUDE.md", Content: []byte("# Commit Guidelines\n" + similarContent)},
				{Path: "AGENTS.md", Content: []byte("# Commit Guidelines\n" + similarContent)},
			},
			true,
		},
		{
			"same heading nearly identical content should warn",
			[]DetectedFile{
				{Path: "CLAUDE.md", Content: []byte("# Commit Guidelines\n" + similarContent)},
				{Path: "AGENTS.md", Content: []byte("# Commit Guidelines\n" + similarContent + "\nAlso sign off commits.")},
			},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &RuleContext{Files: tt.files, Config: DefaultConfig()}
			rule := &duplicateTopicsRule{}
			diags := rule.Check(ctx)
			if tt.wantDiag && len(diags) == 0 {
				t.Error("expected diagnostic, got none")
			}
			if !tt.wantDiag && len(diags) > 0 {
				t.Errorf("expected no diagnostic, got: %v", diags)
			}
		})
	}
}

func TestContentSimilarity(t *testing.T) {
	tests := []struct {
		name      string
		a, b      string
		wantAbove float64
		wantBelow float64
	}{
		{"identical", "the quick brown fox jumps over the lazy dog", "the quick brown fox jumps over the lazy dog", 0.99, 1.01},
		{"very similar", "the quick brown fox jumps over the lazy dog", "the quick brown fox leaps over the lazy dog", 0.7, 1.0},
		{"completely different", "follow conventional commits for all messages", "install dependencies with npm and start the dev server", 0.0, 0.3},
		{"short identical", "hello", "hello", 0.99, 1.01},
		{"short different", "hello", "world", -0.1, 0.01},
		{"empty", "", "", -0.1, 0.01},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := contentSimilarity(tt.a, tt.b)
			if score < tt.wantAbove {
				t.Errorf("similarity %f is below expected minimum %f", score, tt.wantAbove)
			}
			if score > tt.wantBelow {
				t.Errorf("similarity %f is above expected maximum %f", score, tt.wantBelow)
			}
		})
	}
}

func TestAgentFrontmatterRule(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		wantErrs int
		wantWarn int
	}{
		{
			"valid agent",
			"name: my-agent\ndescription: Does things\n",
			0, 0,
		},
		{
			"missing name",
			"description: Does things\n",
			1, 0,
		},
		{
			"missing description",
			"name: my-agent\n",
			0, 1,
		},
		{
			"invalid yaml",
			": invalid: yaml: [",
			1, 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &RuleContext{
				Files: []DetectedFile{{
					Path:     ".claude/agents/test.yml",
					Platform: PlatformClaude,
					Content:  []byte(tt.content),
				}},
				Config: DefaultConfig(),
			}
			rule := &agentFrontmatterRule{}
			diags := rule.Check(ctx)
			errs := 0
			warns := 0
			for _, d := range diags {
				if d.Severity == mold.SeverityError {
					errs++
				}
				if d.Severity == mold.SeverityWarning {
					warns++
				}
			}
			if errs != tt.wantErrs {
				t.Errorf("expected %d errors, got %d", tt.wantErrs, errs)
			}
			if warns != tt.wantWarn {
				t.Errorf("expected %d warnings, got %d", tt.wantWarn, warns)
			}
		})
	}
}

func TestCommandFrontmatterRule(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		wantDiag  bool
		wantCount int // expected number of diagnostics (0 means unchecked)
	}{
		{
			"valid frontmatter",
			"---\nallowed-tools: [Read, Write]\nmodel: sonnet\n---\n# Command\n",
			false, 0,
		},
		{
			"unknown field",
			"---\nunknown-field: value\n---\n# Command\n",
			true, 1, // multiple unknown fields → single diagnostic
		},
		{
			"multiple unknown fields collapsed",
			"---\ntopic: foo\nsource: bar\ncreated: 2026-01-01\n---\n# Command\n",
			true, 1, // three unknown fields → still one diagnostic
		},
		{
			"no frontmatter",
			"# Command\nJust content\n",
			false, 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &RuleContext{
				Files: []DetectedFile{{
					Path:     ".claude/commands/test.md",
					Platform: PlatformClaude,
					Content:  []byte(tt.content),
				}},
				Config: DefaultConfig(),
			}
			rule := &commandFrontmatterRule{}
			diags := rule.Check(ctx)
			if tt.wantDiag && len(diags) == 0 {
				t.Error("expected diagnostic, got none")
			}
			if !tt.wantDiag && len(diags) > 0 {
				t.Errorf("expected no diagnostic, got: %v", diags)
			}
			if tt.wantCount > 0 && len(diags) != tt.wantCount {
				t.Errorf("expected %d diagnostic(s), got %d: %v", tt.wantCount, len(diags), diags)
			}
		})
	}
}

func TestCommandFrontmatterRule_FixData(t *testing.T) {
	ctx := &RuleContext{
		Files: []DetectedFile{{
			Path:     ".claude/commands/test.md",
			Platform: PlatformClaude,
			Content:  []byte("---\ntopic: foo\ncreated: 2026-01-01\n---\n# Cmd\n"),
		}},
		Config: DefaultConfig(),
	}
	diags := (&commandFrontmatterRule{}).Check(ctx)
	if len(diags) == 0 {
		t.Fatal("expected diagnostic")
	}
	d := diags[0]
	if d.FixData == nil {
		t.Fatal("expected FixData to be set")
	}
	fields, ok := d.FixData["fields"].([]string)
	if !ok {
		t.Fatalf("expected FixData[\"fields\"] to be []string, got %T", d.FixData["fields"])
	}
	if len(fields) != 2 {
		t.Errorf("expected 2 unknown fields, got %d: %v", len(fields), fields)
	}
	// Tip should mention both --fix shortcut and manual command
	if !containsSubstr(d.Tip, "lint --fix") {
		t.Errorf("tip should mention 'lint --fix', got: %s", d.Tip)
	}
	if !containsSubstr(d.Tip, "config allow-fields") {
		t.Errorf("tip should mention 'config allow-fields', got: %s", d.Tip)
	}
}

func TestCommandFrontmatterRule_MultilineDescription(t *testing.T) {
	cases := []struct {
		name    string
		content string
		wantErr bool
	}{
		{
			"single-line description — ok",
			"---\nname: my-skill\ndescription: Does a thing.\n---\n# Skill\n",
			false,
		},
		{
			"multiline description (indented block) — error",
			"---\nname: my-skill\ndescription:\n  Does a thing\n  spanning two lines.\n---\n# Skill\n",
			true,
		},
		{
			"multiline name — error",
			"---\nname:\n  my-skill\ndescription: fine\n---\n# Skill\n",
			true,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ctx := &RuleContext{
				Files: []DetectedFile{{
					Path:     ".claude/commands/test.md",
					Platform: PlatformClaude,
					Content:  []byte(c.content),
				}},
				Config: DefaultConfig(),
			}
			diags := (&commandFrontmatterRule{}).Check(ctx)
			var hasErr bool
			for _, d := range diags {
				if d.Severity == mold.SeverityError && containsSubstr(d.Message, "single-line") {
					hasErr = true
				}
			}
			if c.wantErr && !hasErr {
				t.Error("expected multiline error diagnostic, got none")
			}
			if !c.wantErr && hasErr {
				t.Error("expected no multiline error diagnostic, got one")
			}
		})
	}
}

func TestCommandFrontmatterRule_ExtraAllowedFields(t *testing.T) {
	enabled := true
	ctx := &RuleContext{
		Files: []DetectedFile{{
			Path:     ".claude/commands/test.md",
			Platform: PlatformClaude,
			Content:  []byte("---\ntopic: foo\nsource: bar\ntags: [a,b]\n---\n# Cmd\n"),
		}},
		Config: &Config{
			Rules: map[string]RuleConfig{
				"command-frontmatter": {
					Enabled: &enabled,
					Options: map[string]any{
						"extra-allowed-fields": []any{"topic", "source", "tags"},
					},
				},
			},
		},
	}
	rule := &commandFrontmatterRule{}
	diags := rule.Check(ctx)
	if len(diags) != 0 {
		t.Errorf("expected no diagnostics with extra-allowed-fields, got: %v", diags)
	}
}

func TestDescriptionLengthRule(t *testing.T) {
	shortDesc := "Does things"
	longDesc := strings.Repeat("a", 101) // 101 chars, over default 100

	tests := []struct {
		name     string
		files    []DetectedFile
		maxLen   int
		wantDiag bool
	}{
		{
			"short agent description",
			[]DetectedFile{{
				Path:     ".claude/agents/test.yml",
				Platform: PlatformClaude,
				Content:  []byte("name: test\ndescription: " + shortDesc + "\n"),
			}},
			0, false,
		},
		{
			"long agent description",
			[]DetectedFile{{
				Path:     ".claude/agents/test.yml",
				Platform: PlatformClaude,
				Content:  []byte("name: test\ndescription: " + longDesc + "\n"),
			}},
			0, true,
		},
		{
			"short command description",
			[]DetectedFile{{
				Path:     ".claude/commands/test.md",
				Platform: PlatformClaude,
				Content:  []byte("---\ndescription: " + shortDesc + "\n---\n# Cmd\n"),
			}},
			0, false,
		},
		{
			"long command description",
			[]DetectedFile{{
				Path:     ".claude/commands/test.md",
				Platform: PlatformClaude,
				Content:  []byte("---\ndescription: " + longDesc + "\n---\n# Cmd\n"),
			}},
			0, true,
		},
		{
			"custom max-length",
			[]DetectedFile{{
				Path:     ".claude/agents/test.yml",
				Platform: PlatformClaude,
				Content:  []byte("name: test\ndescription: " + longDesc + "\n"),
			}},
			200, false,
		},
		{
			"no description field — no warning",
			[]DetectedFile{{
				Path:     ".claude/agents/test.yml",
				Platform: PlatformClaude,
				Content:  []byte("name: test\n"),
			}},
			0, false,
		},
		{
			"long plugin manifest description",
			[]DetectedFile{{
				Path:      "plugins/my-plugin/.claude-plugin/plugin.json",
				Platform:  PlatformClaude,
				PluginDir: "plugins/my-plugin",
				Content:   []byte(`{"name":"test","version":"1.0","description":"` + longDesc + `"}`),
			}},
			0, true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			if tt.maxLen > 0 {
				cfg.Rules = map[string]RuleConfig{
					"description-length": {Options: map[string]any{"max-length": tt.maxLen}},
				}
			}
			ctx := &RuleContext{Files: tt.files, Config: cfg}
			rule := &descriptionLengthRule{}
			diags := rule.Check(ctx)
			if tt.wantDiag && len(diags) == 0 {
				t.Error("expected diagnostic, got none")
			}
			if !tt.wantDiag && len(diags) > 0 {
				t.Errorf("expected no diagnostic, got: %v", diags)
			}
		})
	}
}

func TestDescriptionPointOfViewRule(t *testing.T) {
	tests := []struct {
		name     string
		desc     string
		wantDiag bool
	}{
		{"third person — ok", "Processes Excel files and generates reports", false},
		{"first person I can", "I can help you process Excel files", true},
		{"second person you can", "You can use this to process Excel files", true},
		{"first person I will", "I will analyze the data", true},
		{"second person your", "Handles your document processing", true},
		{"no description", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := "name: test\n"
			if tt.desc != "" {
				content += "description: " + tt.desc + "\n"
			}
			ctx := &RuleContext{
				Files: []DetectedFile{{
					Path:     ".claude/agents/test.yml",
					Platform: PlatformClaude,
					Content:  []byte(content),
				}},
				Config: DefaultConfig(),
			}
			diags := (&descriptionPointOfViewRule{}).Check(ctx)
			if tt.wantDiag && len(diags) == 0 {
				t.Error("expected diagnostic, got none")
			}
			if !tt.wantDiag && len(diags) > 0 {
				t.Errorf("expected no diagnostic, got: %v", diags)
			}
		})
	}
}

func TestDescriptionPointOfViewRule_CommandFrontmatter(t *testing.T) {
	ctx := &RuleContext{
		Files: []DetectedFile{{
			Path:     ".claude/commands/test.md",
			Platform: PlatformClaude,
			Content:  []byte("---\nname: test\ndescription: I can help with things\n---\n# Cmd\n"),
		}},
		Config: DefaultConfig(),
	}
	diags := (&descriptionPointOfViewRule{}).Check(ctx)
	if len(diags) == 0 {
		t.Error("expected diagnostic for first person in command description")
	}
}

func TestDescriptionMissingTriggerRule(t *testing.T) {
	tests := []struct {
		name     string
		desc     string
		wantDiag bool
	}{
		{"has trigger — Use when", "Processes PDFs. Use when working with PDF files.", false},
		{"has trigger — Trigger when", "Analyzes data. Trigger when user mentions spreadsheets.", false},
		{"has trigger — Use for", "Commits code. Use for generating commit messages.", false},
		{"has trigger — Use if", "Formats code. Use if the user asks for formatting.", false},
		{"missing trigger", "Processes Excel files and generates reports", true},
		{"no description", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := "name: test\n"
			if tt.desc != "" {
				content += "description: " + tt.desc + "\n"
			}
			ctx := &RuleContext{
				Files: []DetectedFile{{
					Path:     ".claude/agents/test.yml",
					Platform: PlatformClaude,
					Content:  []byte(content),
				}},
				Config: DefaultConfig(),
			}
			diags := (&descriptionMissingTriggerRule{}).Check(ctx)
			if tt.wantDiag && len(diags) == 0 {
				t.Error("expected diagnostic, got none")
			}
			if !tt.wantDiag && len(diags) > 0 {
				t.Errorf("expected no diagnostic, got: %v", diags)
			}
		})
	}
}

func TestNameFormatRule(t *testing.T) {
	tests := []struct {
		name     string
		skillNm  string
		wantDiag bool
	}{
		{"valid lowercase", "my-skill", false},
		{"valid with numbers", "pdf-v2", false},
		{"uppercase letters", "My-Skill", true},
		{"spaces", "my skill", true},
		{"underscores", "my_skill", true},
		{"too long", strings.Repeat("a", 65), true},
		{"exactly 64", strings.Repeat("a", 64), false},
		{"starts with hyphen", "-my-skill", true},
		{"ends with hyphen", "my-skill-", true},
		{"consecutive hyphens", "my--skill", true},
		{"single char", "a", false},
		{"empty name", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := ""
			if tt.skillNm != "" {
				content = "name: " + tt.skillNm + "\ndescription: test\n"
			} else {
				content = "description: test\n"
			}
			ctx := &RuleContext{
				Files: []DetectedFile{{
					Path:     ".claude/agents/test.yml",
					Platform: PlatformClaude,
					Content:  []byte(content),
				}},
				Config: DefaultConfig(),
			}
			diags := (&nameFormatRule{}).Check(ctx)
			if tt.wantDiag && len(diags) == 0 {
				t.Error("expected diagnostic, got none")
			}
			if !tt.wantDiag && len(diags) > 0 {
				t.Errorf("expected no diagnostic, got: %v", diags)
			}
		})
	}
}

func TestNameFormatRule_CommandFrontmatter(t *testing.T) {
	ctx := &RuleContext{
		Files: []DetectedFile{{
			Path:     ".claude/commands/test.md",
			Platform: PlatformClaude,
			Content:  []byte("---\nname: My_Invalid_Name\ndescription: test\n---\n# Cmd\n"),
		}},
		Config: DefaultConfig(),
	}
	diags := (&nameFormatRule{}).Check(ctx)
	if len(diags) == 0 {
		t.Error("expected diagnostic for invalid command name")
	}
}

func TestNameReservedWordsRule(t *testing.T) {
	tests := []struct {
		name     string
		skillNm  string
		wantDiag bool
	}{
		{"normal name", "my-skill", false},
		{"contains claude", "claude-helper", true},
		{"contains anthropic", "anthropic-tools", true},
		{"claude embedded", "my-claude-skill", true},
		{"no name", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := ""
			if tt.skillNm != "" {
				content = "name: " + tt.skillNm + "\n"
			} else {
				content = "description: test\n"
			}
			ctx := &RuleContext{
				Files: []DetectedFile{{
					Path:     ".claude/agents/test.yml",
					Platform: PlatformClaude,
					Content:  []byte(content),
				}},
				Config: DefaultConfig(),
			}
			diags := (&nameReservedWordsRule{}).Check(ctx)
			if tt.wantDiag && len(diags) == 0 {
				t.Error("expected diagnostic, got none")
			}
			if !tt.wantDiag && len(diags) > 0 {
				t.Errorf("expected no diagnostic, got: %v", diags)
			}
		})
	}
}

func TestVagueNameRule(t *testing.T) {
	tests := []struct {
		name     string
		skillNm  string
		wantDiag bool
	}{
		{"specific name", "processing-pdfs", false},
		{"vague helper", "helper", true},
		{"vague utils", "utils", true},
		{"vague tools", "tools", true},
		{"vague data", "data", true},
		{"vague files", "files", true},
		{"vague misc", "misc", true},
		{"no name", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := ""
			if tt.skillNm != "" {
				content = "name: " + tt.skillNm + "\n"
			} else {
				content = "description: test\n"
			}
			ctx := &RuleContext{
				Files: []DetectedFile{{
					Path:     ".claude/agents/test.yml",
					Platform: PlatformClaude,
					Content:  []byte(content),
				}},
				Config: DefaultConfig(),
			}
			diags := (&vagueNameRule{}).Check(ctx)
			if tt.wantDiag && len(diags) == 0 {
				t.Error("expected diagnostic, got none")
			}
			if !tt.wantDiag && len(diags) > 0 {
				t.Errorf("expected no diagnostic, got: %v", diags)
			}
		})
	}
}

func TestSkillBodyLengthRule(t *testing.T) {
	shortBody := strings.Repeat("line\n", 100)
	longBody := strings.Repeat("line\n", 501)

	tests := []struct {
		name     string
		content  string
		maxLines int
		wantDiag bool
	}{
		{
			"short skill body",
			"---\nname: my-skill\ndescription: test\n---\n" + shortBody,
			0, false,
		},
		{
			"long skill body",
			"---\nname: my-skill\ndescription: test\n---\n" + longBody,
			0, true,
		},
		{
			"custom max-lines",
			"---\nname: my-skill\ndescription: test\n---\n" + longBody,
			600, false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			if tt.maxLines > 0 {
				cfg.Rules = map[string]RuleConfig{
					"skill-body-length": {Options: map[string]any{"max-lines": tt.maxLines}},
				}
			}
			ctx := &RuleContext{
				Files: []DetectedFile{{
					Path:      "plugins/my-plugin/skills/big-skill/SKILL.md",
					Platform:  PlatformClaude,
					PluginDir: "plugins/my-plugin",
					Content:   []byte(tt.content),
				}},
				Config: cfg,
			}
			diags := (&skillBodyLengthRule{}).Check(ctx)
			if tt.wantDiag && len(diags) == 0 {
				t.Error("expected diagnostic, got none")
			}
			if !tt.wantDiag && len(diags) > 0 {
				t.Errorf("expected no diagnostic, got: %v", diags)
			}
		})
	}
}

func TestSkillBodyLengthRule_NonSkillIgnored(t *testing.T) {
	longBody := strings.Repeat("line\n", 501)
	ctx := &RuleContext{
		Files: []DetectedFile{{
			Path:     ".claude/commands/test.md",
			Platform: PlatformClaude,
			Content:  []byte("---\nname: test\n---\n" + longBody),
		}},
		Config: DefaultConfig(),
	}
	diags := (&skillBodyLengthRule{}).Check(ctx)
	if len(diags) > 0 {
		t.Errorf("expected no diagnostic for non-skill file, got: %v", diags)
	}
}

func TestCommandsDeprecatedRule(t *testing.T) {
	tests := []struct {
		name     string
		files    []DetectedFile
		wantDiag bool
		wantTip  string
	}{
		{
			"standard command triggers warning",
			[]DetectedFile{{
				Path:     ".claude/commands/create-issue.md",
				Platform: PlatformClaude,
				Content:  []byte("---\nname: create-issue\ndescription: Create issues\n---\n# Create Issue\n"),
			}},
			true,
			".claude/skills/create-issue/SKILL.md",
		},
		{
			"plugin command triggers warning",
			[]DetectedFile{{
				Path:      "plugins/my-plugin/commands/helper.md",
				Platform:  PlatformClaude,
				PluginDir: "plugins/my-plugin",
				Content:   []byte("---\nname: helper\n---\n# Helper\n"),
			}},
			true,
			"plugins/my-plugin/skills/helper/SKILL.md",
		},
		{
			"skill file does not trigger",
			[]DetectedFile{{
				Path:      "plugins/my-plugin/skills/brainstorm/SKILL.md",
				Platform:  PlatformClaude,
				PluginDir: "plugins/my-plugin",
				Content:   []byte("---\nname: brainstorm\n---\n# Brainstorm\n"),
			}},
			false,
			"",
		},
		{
			"non-claude file ignored",
			[]DetectedFile{{
				Path:     ".claude/commands/test.md",
				Platform: PlatformGeneric,
				Content:  []byte("# Test\n"),
			}},
			false,
			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &RuleContext{Files: tt.files, Config: DefaultConfig()}
			diags := (&commandsDeprecatedRule{}).Check(ctx)
			if tt.wantDiag && len(diags) == 0 {
				t.Error("expected diagnostic, got none")
			}
			if !tt.wantDiag && len(diags) > 0 {
				t.Errorf("expected no diagnostic, got: %v", diags)
			}
			if tt.wantTip != "" && len(diags) > 0 {
				if !containsSubstr(diags[0].Message, tt.wantTip) {
					t.Errorf("expected message to contain migration path %q, got: %s", tt.wantTip, diags[0].Message)
				}
				if !containsSubstr(diags[0].Tip, "agentskills.io") {
					t.Errorf("expected tip to reference agentskills.io, got: %s", diags[0].Tip)
				}
			}
		})
	}
}

func TestNameDirectoryMismatchRule(t *testing.T) {
	tests := []struct {
		name     string
		files    []DetectedFile
		wantDiag bool
	}{
		{
			"name matches directory",
			[]DetectedFile{{
				Path:      "plugins/my-plugin/skills/pdf-processing/SKILL.md",
				Platform:  PlatformClaude,
				PluginDir: "plugins/my-plugin",
				Content:   []byte("---\nname: pdf-processing\ndescription: test\n---\n# PDF\n"),
			}},
			false,
		},
		{
			"name does not match directory",
			[]DetectedFile{{
				Path:      "plugins/my-plugin/skills/pdf-tools/SKILL.md",
				Platform:  PlatformClaude,
				PluginDir: "plugins/my-plugin",
				Content:   []byte("---\nname: pdf-processing\ndescription: test\n---\n# PDF\n"),
			}},
			true,
		},
		{
			"standard skill path matches",
			[]DetectedFile{{
				Path:     ".claude/skills/my-skill/SKILL.md",
				Platform: PlatformClaude,
				Content:  []byte("---\nname: my-skill\ndescription: test\n---\n# Skill\n"),
			}},
			false,
		},
		{
			"standard skill path mismatches",
			[]DetectedFile{{
				Path:     ".claude/skills/wrong-name/SKILL.md",
				Platform: PlatformClaude,
				Content:  []byte("---\nname: my-skill\ndescription: test\n---\n# Skill\n"),
			}},
			true,
		},
		{
			"command file ignored",
			[]DetectedFile{{
				Path:     ".claude/commands/test.md",
				Platform: PlatformClaude,
				Content:  []byte("---\nname: different\n---\n# Cmd\n"),
			}},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &RuleContext{Files: tt.files, Config: DefaultConfig()}
			diags := (&nameDirectoryMismatchRule{}).Check(ctx)
			if tt.wantDiag && len(diags) == 0 {
				t.Error("expected diagnostic, got none")
			}
			if !tt.wantDiag && len(diags) > 0 {
				t.Errorf("expected no diagnostic, got: %v", diags)
			}
		})
	}
}

func TestDescriptionMaxLengthRule(t *testing.T) {
	shortDesc := "Does things well"
	longDesc := strings.Repeat("a", 1025) // over 1024

	tests := []struct {
		name     string
		desc     string
		wantDiag bool
	}{
		{"under limit", shortDesc, false},
		{"at limit", strings.Repeat("a", 1024), false},
		{"over limit", longDesc, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &RuleContext{
				Files: []DetectedFile{{
					Path:     ".claude/agents/test.yml",
					Platform: PlatformClaude,
					Content:  []byte("name: test\ndescription: " + tt.desc + "\n"),
				}},
				Config: DefaultConfig(),
			}
			diags := (&descriptionMaxLengthRule{}).Check(ctx)
			if tt.wantDiag && len(diags) == 0 {
				t.Error("expected diagnostic, got none")
			}
			if !tt.wantDiag && len(diags) > 0 {
				t.Errorf("expected no diagnostic, got: %v", diags)
			}
		})
	}
}

func TestCompatibilityLengthRule(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		wantDiag bool
	}{
		{
			"no compatibility field",
			"---\nname: test\ndescription: test\n---\n# Skill\n",
			false,
		},
		{
			"short compatibility",
			"---\nname: test\ndescription: test\ncompatibility: Requires git\n---\n# Skill\n",
			false,
		},
		{
			"long compatibility",
			"---\nname: test\ndescription: test\ncompatibility: " + strings.Repeat("a", 501) + "\n---\n# Skill\n",
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &RuleContext{
				Files: []DetectedFile{{
					Path:      "plugins/my-plugin/skills/test/SKILL.md",
					Platform:  PlatformClaude,
					PluginDir: "plugins/my-plugin",
					Content:   []byte(tt.content),
				}},
				Config: DefaultConfig(),
			}
			diags := (&compatibilityLengthRule{}).Check(ctx)
			if tt.wantDiag && len(diags) == 0 {
				t.Error("expected diagnostic, got none")
			}
			if !tt.wantDiag && len(diags) > 0 {
				t.Errorf("expected no diagnostic, got: %v", diags)
			}
		})
	}
}

func TestSkillTokenBudgetRule(t *testing.T) {
	// 5000 tokens * 4 chars/token = 20000 chars
	shortBody := strings.Repeat("word ", 1000) // ~5000 chars = ~1250 tokens
	longBody := strings.Repeat("word ", 5000)  // ~25000 chars = ~6250 tokens

	tests := []struct {
		name      string
		content   string
		maxTokens int
		wantDiag  bool
	}{
		{
			"under budget",
			"---\nname: my-skill\ndescription: test\n---\n" + shortBody,
			0, false,
		},
		{
			"over budget",
			"---\nname: my-skill\ndescription: test\n---\n" + longBody,
			0, true,
		},
		{
			"custom max-tokens",
			"---\nname: my-skill\ndescription: test\n---\n" + longBody,
			10000, false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			if tt.maxTokens > 0 {
				cfg.Rules = map[string]RuleConfig{
					"skill-token-budget": {Options: map[string]any{"max-tokens": tt.maxTokens}},
				}
			}
			ctx := &RuleContext{
				Files: []DetectedFile{{
					Path:      "plugins/my-plugin/skills/big-skill/SKILL.md",
					Platform:  PlatformClaude,
					PluginDir: "plugins/my-plugin",
					Content:   []byte(tt.content),
				}},
				Config: cfg,
			}
			diags := (&skillTokenBudgetRule{}).Check(ctx)
			if tt.wantDiag && len(diags) == 0 {
				t.Error("expected diagnostic, got none")
			}
			if !tt.wantDiag && len(diags) > 0 {
				t.Errorf("expected no diagnostic, got: %v", diags)
			}
		})
	}
}

func TestSkillTokenBudgetRule_NonSkillIgnored(t *testing.T) {
	longBody := strings.Repeat("word ", 5000)
	ctx := &RuleContext{
		Files: []DetectedFile{{
			Path:     ".claude/commands/test.md",
			Platform: PlatformClaude,
			Content:  []byte("---\nname: test\n---\n" + longBody),
		}},
		Config: DefaultConfig(),
	}
	diags := (&skillTokenBudgetRule{}).Check(ctx)
	if len(diags) > 0 {
		t.Errorf("expected no diagnostic for non-skill file, got: %v", diags)
	}
}

func TestDescriptionImperativeRule(t *testing.T) {
	tests := []struct {
		name     string
		desc     string
		wantDiag bool
	}{
		{"imperative — ok", "Analyzes CSV files and generates reports", false},
		{"trigger phrase — ok", "Use this skill when working with PDFs", false},
		{"declarative this skill", "This skill processes Excel files", true},
		{"declarative this tool", "This tool helps with code review", true},
		{"declarative a skill that", "A skill that generates reports", true},
		{"declarative a tool that", "A tool that analyzes data", true},
		{"no description", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := "name: test\n"
			if tt.desc != "" {
				content += "description: " + tt.desc + "\n"
			}
			ctx := &RuleContext{
				Files: []DetectedFile{{
					Path:     ".claude/agents/test.yml",
					Platform: PlatformClaude,
					Content:  []byte(content),
				}},
				Config: DefaultConfig(),
			}
			diags := (&descriptionImperativeRule{}).Check(ctx)
			if tt.wantDiag && len(diags) == 0 {
				t.Error("expected diagnostic, got none")
			}
			if !tt.wantDiag && len(diags) > 0 {
				t.Errorf("expected no diagnostic, got: %v", diags)
			}
		})
	}
}

func TestSettingsSchemaRule(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		wantDiag bool
	}{
		{
			"valid json",
			`{"permissions": {"allow": ["Read"]}}`,
			false,
		},
		{
			"invalid json",
			`{invalid`,
			true,
		},
		{
			"unknown hook event",
			`{"hooks": {"UnknownEvent": []}}`,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &RuleContext{
				Files: []DetectedFile{{
					Path:     ".claude/settings.json",
					Platform: PlatformClaude,
					Content:  []byte(tt.content),
				}},
				Config: DefaultConfig(),
			}
			rule := &settingsSchemaRule{}
			diags := rule.Check(ctx)
			if tt.wantDiag && len(diags) == 0 {
				t.Error("expected diagnostic, got none")
			}
			if !tt.wantDiag && len(diags) > 0 {
				t.Errorf("expected no diagnostic, got: %v", diags)
			}
		})
	}
}
