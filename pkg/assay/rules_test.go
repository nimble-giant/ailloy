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
		name     string
		content  string
		wantDiag bool
	}{
		{
			"valid frontmatter",
			"---\nallowed-tools: [Read, Write]\nmodel: sonnet\n---\n# Command\n",
			false,
		},
		{
			"unknown field",
			"---\nunknown-field: value\n---\n# Command\n",
			true,
		},
		{
			"no frontmatter",
			"# Command\nJust content\n",
			false,
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
