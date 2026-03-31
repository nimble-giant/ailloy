package assay

import (
	"os"
	"path/filepath"
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

func TestContextUsageRule(t *testing.T) {
	// Helper to create a temp dir with files and return a RuleContext
	setup := func(t *testing.T, files map[string]string, cfg *Config) (*RuleContext, string) {
		t.Helper()
		dir := t.TempDir()
		var detected []DetectedFile
		for name, content := range files {
			path := filepath.Join(dir, name)
			if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte(content), 0644); err != nil {
				t.Fatal(err)
			}
			// Only add CLAUDE.md as a detected file — others are referenced
			if name == "CLAUDE.md" {
				detected = append(detected, DetectedFile{
					Path:     name,
					Platform: PlatformClaude,
					Content:  []byte(content),
				})
			}
		}
		if cfg == nil {
			cfg = DefaultConfig()
		}
		return &RuleContext{RootDir: dir, Files: detected, Config: cfg}, dir
	}

	t.Run("small file no imports", func(t *testing.T) {
		ctx, _ := setup(t, map[string]string{
			"CLAUDE.md": "# Instructions\nBe helpful.\n",
		}, nil)
		diags := (&contextUsageRule{}).Check(ctx)
		if len(diags) > 0 {
			t.Errorf("expected no diagnostic, got: %v", diags)
		}
	})

	t.Run("stats always populated", func(t *testing.T) {
		ctx, _ := setup(t, map[string]string{
			"CLAUDE.md": "# Instructions\nBe helpful.\n",
		}, nil)
		(&contextUsageRule{}).Check(ctx)
		if len(ctx.ContextStats) != 1 {
			t.Fatalf("expected 1 context stat, got %d", len(ctx.ContextStats))
		}
		stat := ctx.ContextStats[0]
		if stat.File != "CLAUDE.md" {
			t.Errorf("expected file CLAUDE.md, got %s", stat.File)
		}
		if stat.EstimatedTokens <= 0 {
			t.Errorf("expected positive token estimate, got %d", stat.EstimatedTokens)
		}
	})

	t.Run("stats with imports", func(t *testing.T) {
		ctx, _ := setup(t, map[string]string{
			"CLAUDE.md": "# Instructions\n@shared.md\n",
			"shared.md": strings.Repeat("word ", 2000),
		}, nil)
		(&contextUsageRule{}).Check(ctx)
		if len(ctx.ContextStats) != 1 {
			t.Fatalf("expected 1 context stat, got %d", len(ctx.ContextStats))
		}
		stat := ctx.ContextStats[0]
		if stat.ImportCount != 1 {
			t.Errorf("expected 1 import, got %d", stat.ImportCount)
		}
	})

	t.Run("below warning threshold", func(t *testing.T) {
		// ~10K chars = ~2500 tokens, well under 15K default
		ctx, _ := setup(t, map[string]string{
			"CLAUDE.md": "# Instructions\n@shared.md\n",
			"shared.md": strings.Repeat("word ", 2000), // ~10K chars
		}, nil)
		diags := (&contextUsageRule{}).Check(ctx)
		if len(diags) > 0 {
			t.Errorf("expected no diagnostic, got: %v", diags)
		}
	})

	t.Run("above warning threshold", func(t *testing.T) {
		// Default warn = 10% of 200K = 20K tokens. ~100K chars = ~25K tokens, above warn.
		ctx, _ := setup(t, map[string]string{
			"CLAUDE.md": "# Instructions\n@shared.md\n",
			"shared.md": strings.Repeat("word ", 20000), // ~100K chars = ~25K tokens
		}, nil)
		diags := (&contextUsageRule{}).Check(ctx)
		warnCount := 0
		for _, d := range diags {
			if d.Severity == mold.SeverityWarning && strings.Contains(d.Message, "% of") {
				warnCount++
				// Verify import breakdown in tip
				if !strings.Contains(d.Tip, "@shared.md") {
					t.Errorf("expected tip to list @shared.md import, got: %s", d.Tip)
				}
			}
		}
		if warnCount != 1 {
			t.Errorf("expected 1 warning diagnostic, got %d; diags: %v", warnCount, diags)
		}
	})

	t.Run("import breakdown lists multiple imports", func(t *testing.T) {
		// Two imports that together exceed the warn threshold
		ctx, _ := setup(t, map[string]string{
			"CLAUDE.md": "@a.md\n@b.md\n",
			"a.md":      strings.Repeat("a", 50000), // ~12.5K tokens
			"b.md":      strings.Repeat("b", 50000), // ~12.5K tokens
		}, nil)
		diags := (&contextUsageRule{}).Check(ctx)
		for _, d := range diags {
			if d.Severity == mold.SeverityWarning && strings.Contains(d.Message, "% of") {
				if !strings.Contains(d.Tip, "@a.md") || !strings.Contains(d.Tip, "@b.md") {
					t.Errorf("expected tip to list both @a.md and @b.md, got: %s", d.Tip)
				}
				// Verify token counts appear for each import
				if !strings.Contains(d.Tip, "tokens") {
					t.Errorf("expected tip to include token counts, got: %s", d.Tip)
				}
			}
		}
	})

	t.Run("above error threshold", func(t *testing.T) {
		// Default error = 25% of 200K = 50K tokens. ~240K chars = ~60K tokens, above error.
		ctx, _ := setup(t, map[string]string{
			"CLAUDE.md": "# Instructions\n@big.md\n",
			"big.md":    strings.Repeat("word ", 48000), // ~240K chars = ~60K tokens
		}, nil)
		diags := (&contextUsageRule{}).Check(ctx)
		errCount := 0
		for _, d := range diags {
			if d.Severity == mold.SeverityError && strings.Contains(d.Message, "% of") {
				errCount++
			}
		}
		if errCount != 1 {
			t.Errorf("expected 1 error diagnostic, got %d; diags: %v", errCount, diags)
		}
	})

	t.Run("custom thresholds via pct", func(t *testing.T) {
		// 100K chars / 3.5 = ~28.5K tokens; custom warn at 20% of 184K = ~36.8K should not trigger
		cfg := DefaultConfig()
		cfg.Rules = map[string]RuleConfig{
			"context-usage": {Options: map[string]any{"warn-pct": 20, "error-pct": 40}},
		}
		ctx, _ := setup(t, map[string]string{
			"CLAUDE.md": "# Instructions\n@shared.md\n",
			"shared.md": strings.Repeat("word ", 20000), // ~100K chars = ~28.5K tokens
		}, cfg)
		diags := (&contextUsageRule{}).Check(ctx)
		for _, d := range diags {
			if d.Severity <= mold.SeverityWarning && strings.Contains(d.Message, "% of") {
				t.Errorf("expected no threshold diagnostic with custom config, got: %v", d)
			}
		}
	})

	t.Run("custom thresholds via absolute tokens", func(t *testing.T) {
		// Legacy: absolute token overrides take precedence
		cfg := DefaultConfig()
		cfg.Rules = map[string]RuleConfig{
			"context-usage": {Options: map[string]any{"warn-tokens": 30000, "error-tokens": 100000}},
		}
		ctx, _ := setup(t, map[string]string{
			"CLAUDE.md": "# Instructions\n@shared.md\n",
			"shared.md": strings.Repeat("word ", 20000), // ~25K tokens, under 30K warn
		}, cfg)
		diags := (&contextUsageRule{}).Check(ctx)
		for _, d := range diags {
			if d.Severity <= mold.SeverityWarning && strings.Contains(d.Message, "% of") {
				t.Errorf("expected no threshold diagnostic with custom absolute config, got: %v", d)
			}
		}
	})

	t.Run("non-claude file skipped", func(t *testing.T) {
		dir := t.TempDir()
		ctx := &RuleContext{
			RootDir: dir,
			Files: []DetectedFile{{
				Path:     ".cursorrules",
				Platform: PlatformCursor,
				Content:  []byte(strings.Repeat("word ", 48000)),
			}},
			Config: DefaultConfig(),
		}
		diags := (&contextUsageRule{}).Check(ctx)
		if len(diags) > 0 {
			t.Errorf("expected no diagnostic for non-Claude file, got: %v", diags)
		}
	})

	t.Run("unresolvable ref skipped", func(t *testing.T) {
		ctx, _ := setup(t, map[string]string{
			"CLAUDE.md": "# Instructions\n@nonexistent.md\n",
		}, nil)
		diags := (&contextUsageRule{}).Check(ctx)
		if len(diags) > 0 {
			t.Errorf("expected no diagnostic for unresolvable ref, got: %v", diags)
		}
	})

	t.Run("transitive imports", func(t *testing.T) {
		// A -> B -> C, each ~30K chars = total ~90K chars = ~22500 tokens > 20K warn (10% of 200K)
		ctx, _ := setup(t, map[string]string{
			"CLAUDE.md": strings.Repeat("x", 30000) + "\n@b.md\n",
			"b.md":      strings.Repeat("y", 30000) + "\n@c.md\n",
			"c.md":      strings.Repeat("z", 30000),
		}, nil)
		diags := (&contextUsageRule{}).Check(ctx)
		warnCount := 0
		for _, d := range diags {
			if d.Severity == mold.SeverityWarning && strings.Contains(d.Message, "% of") {
				warnCount++
			}
		}
		if warnCount != 1 {
			t.Errorf("expected 1 warning diagnostic for transitive imports, got %d; diags: %v", warnCount, diags)
		}
	})
}

func TestContextUsageRule_CircularReference(t *testing.T) {
	dir := t.TempDir()
	// A imports B, B imports A
	if err := os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte("# Instructions\n@b.md\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.md"), []byte("# Shared\n@CLAUDE.md\n"), 0644); err != nil {
		t.Fatal(err)
	}

	ctx := &RuleContext{
		RootDir: dir,
		Files: []DetectedFile{{
			Path:     "CLAUDE.md",
			Platform: PlatformClaude,
			Content:  []byte("# Instructions\n@b.md\n"),
		}},
		Config: DefaultConfig(),
	}

	diags := (&contextUsageRule{}).Check(ctx)
	cycleFound := false
	for _, d := range diags {
		if strings.Contains(d.Message, "circular") {
			cycleFound = true
		}
	}
	if !cycleFound {
		t.Error("expected circular reference diagnostic, got none")
	}
}

func TestContextUsageRule_SharedImportCountedOnce(t *testing.T) {
	dir := t.TempDir()
	// CLAUDE.md imports b.md and c.md; both import shared.md
	// shared.md is large enough that counting it twice would cross a threshold
	sharedContent := strings.Repeat("s", 50000) // ~12500 tokens
	if err := os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte("@b.md\n@c.md\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.md"), []byte("@shared.md\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "c.md"), []byte("@shared.md\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "shared.md"), []byte(sharedContent), 0644); err != nil {
		t.Fatal(err)
	}

	ctx := &RuleContext{
		RootDir: dir,
		Files: []DetectedFile{{
			Path:     "CLAUDE.md",
			Platform: PlatformClaude,
			Content:  []byte("@b.md\n@c.md\n"),
		}},
		Config: DefaultConfig(),
	}

	diags := (&contextUsageRule{}).Check(ctx)
	// Total should be ~50K chars / 4 = ~12500 tokens (shared counted once)
	// This is under the 20K warning threshold (10% of 200K), so no threshold diagnostic expected
	for _, d := range diags {
		if strings.Contains(d.Message, "% of") {
			t.Errorf("shared import should be counted once; expected no threshold diagnostic, got: %v", d)
		}
	}
}

func TestContextUsageRule_SkipsNonMDFiles(t *testing.T) {
	ctx := &RuleContext{
		RootDir: t.TempDir(),
		Files: []DetectedFile{
			{Path: ".claude/settings.json", Platform: PlatformClaude, Content: []byte(`{"key": "value"}`)},
			{Path: ".claude/agents/default.yaml", Platform: PlatformClaude, Content: []byte("name: default\n")},
		},
		Config: DefaultConfig(),
	}
	(&contextUsageRule{}).Check(ctx)
	if len(ctx.ContextStats) != 0 {
		t.Errorf("expected 0 context stats for non-.md files, got %d", len(ctx.ContextStats))
	}
}

func TestContextUsageRule_PluginDirPropagated(t *testing.T) {
	dir := t.TempDir()
	content := "# Plugin instructions\nBe helpful.\n"
	if err := os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	ctx := &RuleContext{
		RootDir: dir,
		Files: []DetectedFile{
			{Path: "CLAUDE.md", Platform: PlatformClaude, Content: []byte(content), PluginDir: "plugins/my-plugin"},
		},
		Config: DefaultConfig(),
	}
	(&contextUsageRule{}).Check(ctx)
	if len(ctx.ContextStats) != 1 {
		t.Fatalf("expected 1 context stat, got %d", len(ctx.ContextStats))
	}
	if ctx.ContextStats[0].PluginDir != "plugins/my-plugin" {
		t.Errorf("expected PluginDir 'plugins/my-plugin', got %q", ctx.ContextStats[0].PluginDir)
	}
}

func TestContextSummary(t *testing.T) {
	stats := []FileContextStat{
		{File: "CLAUDE.md", EstimatedTokens: 500, PluginDir: ""},
		{File: ".claude/commands/foo.md", EstimatedTokens: 200, PluginDir: ""},
		{File: "plugins/a/.claude-plugin/commands/bar.md", EstimatedTokens: 300, PluginDir: "plugins/a"},
		{File: "plugins/a/.claude-plugin/commands/baz.md", EstimatedTokens: 100, PluginDir: "plugins/a"},
		{File: "plugins/b/.claude-plugin/commands/qux.md", EstimatedTokens: 400, PluginDir: "plugins/b"},
	}

	groups, total := contextSummary(stats)

	if total != 1500 {
		t.Errorf("expected total 1500, got %d", total)
	}
	if len(groups) != 3 {
		t.Fatalf("expected 3 groups (project + 2 plugins), got %d", len(groups))
	}
	// project group
	if groups[0].Name != "project" || groups[0].EstimatedTokens != 700 || groups[0].FileCount != 2 {
		t.Errorf("unexpected project group: %+v", groups[0])
	}
	// plugin a
	if groups[1].Name != "plugins/a" || groups[1].EstimatedTokens != 400 || groups[1].FileCount != 2 {
		t.Errorf("unexpected plugin a group: %+v", groups[1])
	}
	// plugin b
	if groups[2].Name != "plugins/b" || groups[2].EstimatedTokens != 400 || groups[2].FileCount != 1 {
		t.Errorf("unexpected plugin b group: %+v", groups[2])
	}
}

func TestContextUsageRule_RollupThresholds(t *testing.T) {
	dir := t.TempDir()
	// 3 plugin files, each ~8K tokens individually (under 20K warn),
	// but plugin total = ~24K which exceeds the 20K warn threshold.
	for _, name := range []string{"a.md", "b.md", "c.md"} {
		content := strings.Repeat("x", 32000) // 32K chars / 4 = 8K tokens each
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	ctx := &RuleContext{
		RootDir: dir,
		Files: []DetectedFile{
			{Path: "a.md", Platform: PlatformClaude, Content: []byte(strings.Repeat("x", 32000)), PluginDir: "plugins/big"},
			{Path: "b.md", Platform: PlatformClaude, Content: []byte(strings.Repeat("x", 32000)), PluginDir: "plugins/big"},
			{Path: "c.md", Platform: PlatformClaude, Content: []byte(strings.Repeat("x", 32000)), PluginDir: "plugins/big"},
		},
		Config: DefaultConfig(),
	}
	diags := (&contextUsageRule{}).Check(ctx)

	// No individual file should trigger (each ~8K tokens)
	for _, d := range diags {
		if d.File != "" && strings.Contains(d.Message, "file expands") {
			t.Errorf("expected no per-file threshold diagnostic, got: %v", d)
		}
	}

	// But the plugin rollup should trigger a warning (~24K > 20K)
	rollupFound := false
	for _, d := range diags {
		if strings.Contains(d.Message, "plugin plugins/big total") {
			rollupFound = true
			if d.Severity != mold.SeverityWarning {
				t.Errorf("expected warning severity for rollup, got %v", d.Severity)
			}
		}
	}
	if !rollupFound {
		t.Errorf("expected rollup threshold warning for plugin, got none; diags: %v", diags)
	}
}
