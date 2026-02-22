package foundry

import (
	"testing"
)

func TestLockedSatisfies(t *testing.T) {
	tests := []struct {
		name  string
		ref   *Reference
		entry *LockEntry
		want  bool
	}{
		{
			name:  "latest always satisfies",
			ref:   &Reference{Type: Latest},
			entry: &LockEntry{Version: "v1.0.0"},
			want:  true,
		},
		{
			name:  "constraint always satisfies",
			ref:   &Reference{Type: Constraint, Version: "^1.0.0"},
			entry: &LockEntry{Version: "v1.2.3"},
			want:  true,
		},
		{
			name:  "exact matches",
			ref:   &Reference{Type: Exact, Version: "v1.0.0"},
			entry: &LockEntry{Version: "v1.0.0"},
			want:  true,
		},
		{
			name:  "exact with v prefix normalization",
			ref:   &Reference{Type: Exact, Version: "1.0.0"},
			entry: &LockEntry{Version: "v1.0.0"},
			want:  true,
		},
		{
			name:  "exact mismatch",
			ref:   &Reference{Type: Exact, Version: "v2.0.0"},
			entry: &LockEntry{Version: "v1.0.0"},
			want:  false,
		},
		{
			name:  "branch never satisfies",
			ref:   &Reference{Type: Branch, Version: "main"},
			entry: &LockEntry{Version: "main"},
			want:  false,
		},
		{
			name:  "sha never satisfies",
			ref:   &Reference{Type: SHA, Version: "abc1234"},
			entry: &LockEntry{Version: "abc1234"},
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := lockedSatisfies(tt.ref, tt.entry); got != tt.want {
				t.Errorf("lockedSatisfies() = %v, want %v", got, tt.want)
			}
		})
	}
}
