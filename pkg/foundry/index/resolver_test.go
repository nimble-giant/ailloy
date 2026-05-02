package index

import "testing"

func TestCanonicalizeSource(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"github.com/owner/repo", "github.com/owner/repo"},
		{"https://github.com/owner/repo", "github.com/owner/repo"},
		{"https://github.com/owner/repo.git", "github.com/owner/repo"},
		{"https://GitHub.com/Owner/Repo", "github.com/owner/repo"},
		{"http://example.com/foundry.yaml", "example.com/foundry.yaml"},
		{"github.com/owner/repo/", "github.com/owner/repo"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got := canonicalizeSource(tt.in)
			if got != tt.want {
				t.Errorf("canonicalizeSource(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
