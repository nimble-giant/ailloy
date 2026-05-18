package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nimble-giant/ailloy/pkg/blanks"
)

func TestEnforceAilloyVersion(t *testing.T) {
	tests := []struct {
		name     string
		requires string
		current  string
		wantErr  bool
	}{
		{name: "no constraint passes", requires: "", current: "0.6.32", wantErr: false},
		{name: "satisfied minimum", requires: ">=0.6.33", current: "0.6.33", wantErr: false},
		{name: "satisfied newer", requires: ">=0.6.33", current: "0.7.0", wantErr: false},
		{name: "unsatisfied minimum", requires: ">=0.6.33", current: "0.6.32", wantErr: true},
		{name: "v-prefixed current still compared", requires: ">=0.6.33", current: "v0.6.32", wantErr: true},
		{name: "dev build skips check", requires: ">=0.6.33", current: "dev", wantErr: false},
		{name: "empty current skips check", requires: ">=0.6.33", current: "", wantErr: false},
		{name: "malformed constraint passes", requires: "latest", current: "0.6.32", wantErr: false},
		{name: "caret constraint satisfied", requires: "^0.6.0", current: "0.6.40", wantErr: false},
	}

	orig := evolveCurrentVersion
	defer func() { evolveCurrentVersion = orig }()

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			evolveCurrentVersion = tc.current
			err := enforceAilloyVersion(tc.requires)
			if tc.wantErr && err == nil {
				t.Fatalf("enforceAilloyVersion(%q) with current %q: want error, got nil", tc.requires, tc.current)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("enforceAilloyVersion(%q) with current %q: want nil, got %v", tc.requires, tc.current, err)
			}
		})
	}
}

// TestEnforceAilloyVersion_ErrorMessage verifies the error names the required
// version, the current version, and an upgrade hint (issue #229 acceptance).
func TestEnforceAilloyVersion_ErrorMessage(t *testing.T) {
	orig := evolveCurrentVersion
	defer func() { evolveCurrentVersion = orig }()
	evolveCurrentVersion = "0.6.32"

	err := enforceAilloyVersion(">=0.6.33")
	if err == nil {
		t.Fatal("want error, got nil")
	}
	msg := err.Error()
	for _, want := range []string{">=0.6.33", "v0.6.32", "brew upgrade ailloy"} {
		if !strings.Contains(msg, want) {
			t.Errorf("error message %q missing %q", msg, want)
		}
	}
}

// TestCheckAilloyRequirement reads requires.ailloy straight from a mold.yaml on
// disk and enforces it against the running version.
func TestCheckAilloyRequirement(t *testing.T) {
	orig := evolveCurrentVersion
	defer func() { evolveCurrentVersion = orig }()
	evolveCurrentVersion = "0.6.32"

	newReader := func(t *testing.T, manifest string) *blanks.MoldReader {
		t.Helper()
		moldDir := t.TempDir()
		if err := os.WriteFile(filepath.Join(moldDir, "mold.yaml"), []byte(manifest), 0o600); err != nil {
			t.Fatal(err)
		}
		reader, err := blanks.NewMoldReaderFromPath(moldDir)
		if err != nil {
			t.Fatalf("NewMoldReaderFromPath: %v", err)
		}
		return reader
	}

	t.Run("blocks when constraint unmet", func(t *testing.T) {
		reader := newReader(t, "apiVersion: v1\nkind: mold\nname: gated\nversion: 0.1.0\nrequires:\n  ailloy: \">=0.6.33\"\n")
		if err := checkAilloyRequirement(reader); err == nil {
			t.Fatal("want error for unmet requires.ailloy, got nil")
		}
	})

	t.Run("passes when no constraint", func(t *testing.T) {
		reader := newReader(t, "apiVersion: v1\nkind: mold\nname: open\nversion: 0.1.0\n")
		if err := checkAilloyRequirement(reader); err != nil {
			t.Fatalf("want nil, got %v", err)
		}
	})
}
