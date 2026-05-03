package index

import (
	"errors"
	"fmt"
	"testing"
)

func TestClassifyGitError_Auth(t *testing.T) {
	cases := []struct {
		name string
		out  string
	}{
		{"https authentication failed", "fatal: Authentication failed for 'https://github.com/owner/private'"},
		{"http 403", "fatal: unable to access 'https://github.com/owner/private/': The requested URL returned error: 403"},
		{"missing username", "fatal: could not read Username for 'https://github.com': terminal prompts disabled"},
		{"ssh permission denied", "git@github.com: Permission denied (publickey)."},
		{"http 401", "fatal: unable to access 'https://example.com/x': The requested URL returned error: 401"},
	}
	base := fmt.Errorf("exit status 128")
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := classifyGitError(base, []byte(tc.out))
			if !errors.Is(got, ErrForbidden) {
				t.Fatalf("errors.Is(_, ErrForbidden) = false; err = %v", got)
			}
		})
	}
}

func TestClassifyGitError_NotFound(t *testing.T) {
	cases := []string{
		"remote: Repository not found.",
		"fatal: repository 'https://github.com/x/y' not found",
		"fatal: unable to access 'https://github.com/x/y': The requested URL returned error: 404",
	}
	base := fmt.Errorf("exit status 128")
	for _, out := range cases {
		t.Run(out, func(t *testing.T) {
			got := classifyGitError(base, []byte(out))
			if !errors.Is(got, ErrNotFound) {
				t.Fatalf("errors.Is(_, ErrNotFound) = false; err = %v", got)
			}
		})
	}
}

func TestClassifyGitError_Unknown(t *testing.T) {
	got := classifyGitError(fmt.Errorf("exit status 1"), []byte("some unrelated git failure"))
	if errors.Is(got, ErrForbidden) || errors.Is(got, ErrNotFound) {
		t.Fatalf("classifyGitError should not classify generic errors; got %v", got)
	}
	if got == nil {
		t.Fatal("expected non-nil error")
	}
}

func TestClassifyGitError_NilPasses(t *testing.T) {
	if err := classifyGitError(nil, []byte("anything")); err != nil {
		t.Fatalf("nil input should pass through, got %v", err)
	}
}
