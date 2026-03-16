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
	tests := []struct {
		name     string
		files    []DetectedFile
		wantDiag bool
	}{
		{
			"no duplicates",
			[]DetectedFile{
				{Path: "CLAUDE.md", Content: []byte("# Setup\nContent")},
				{Path: "AGENTS.md", Content: []byte("# Testing\nContent")},
			},
			false,
		},
		{
			"duplicate headings",
			[]DetectedFile{
				{Path: "CLAUDE.md", Content: []byte("# Setup\nContent")},
				{Path: "AGENTS.md", Content: []byte("# Setup\nDifferent content")},
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
