package commands

import (
	"testing"
)

func TestGetMoldIcon(t *testing.T) {
	tests := []struct {
		name     string
		expected string
	}{
		{"claude-code", "🤖"},
		{"claude-code-review", "🤖"},
		{"brainstorm", "💡"},
		{"claude-code", "🤖"},
		{"claude-code-action", "🤖"},
		{"create-issue", "🎯"},
		{"start-issue", "🎯"},
		{"pr-description", "🔄"},
		{"open-pr", "🔄"},
		// "pr-review" contains "pr" which matches before "review" in the switch
		{"pr-review", "🔄"},
		// "pr-comments" contains "pr" which matches before "comment" in the switch
		{"pr-comments", "🔄"},
		// "preflight" contains "pr" (first two chars) which matches the "pr" case
		{"preflight", "🔄"},
		// "update-pr" contains "pr" which matches the "pr" case
		{"update-pr", "🔄"},
		{"unknown-template", "📋"},
		{"", "📋"},
		// These names uniquely match later cases without containing "pr" or "issue"
		{"code-review", "👀"},
		{"my-comment", "💬"},
		{"my-update", "🔧"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getMoldIcon(tt.name)
			if result != tt.expected {
				t.Errorf("getMoldIcon(%q) = %q, want %q", tt.name, result, tt.expected)
			}
		})
	}
}
