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
