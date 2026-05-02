package foundry

import (
	"os"
	"path/filepath"
	"testing"
	"time"
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

func TestWithForceLatest_SetsForceLatestFlag(t *testing.T) {
	var cfg resolveConfig
	WithForceLatest()(&cfg)
	if !cfg.forceLatest {
		t.Fatal("WithForceLatest should set forceLatest to true")
	}
	if cfg.skipLock {
		t.Error("WithForceLatest should not affect skipLock — lock writes must still happen so the upgrade is persisted")
	}
}

func TestWithoutLock_SkipsLockFileIO(t *testing.T) {
	// Set up: chdir to a temp dir so any lock file write would land here.
	dir := t.TempDir()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })

	// Pre-create a lock file that would match our reference.
	// Without WithoutLock, ResolveWith would read this and use the locked version.
	lock := &LockFile{
		APIVersion: "v1",
		Molds: []LockEntry{{
			Name:      "test-mold",
			Source:    "github.com/test/mold",
			Version:   "v0.0.1",
			Commit:    "oldcommit",
			Timestamp: time.Now().UTC(),
		}},
	}
	if err := WriteLockFile(LockFileName, lock); err != nil {
		t.Fatal(err)
	}

	// Record the lock file's mod time before the resolve call.
	infoBefore, err := os.Stat(filepath.Join(dir, LockFileName))
	if err != nil {
		t.Fatal(err)
	}

	// Verify WithoutLock sets skipLock on the config.
	var cfg resolveConfig
	WithoutLock()(&cfg)
	if !cfg.skipLock {
		t.Fatal("WithoutLock should set skipLock to true")
	}

	// Verify the lock file was not modified (updateLock was not called).
	infoAfter, err := os.Stat(filepath.Join(dir, LockFileName))
	if err != nil {
		t.Fatal(err)
	}
	if infoBefore.ModTime() != infoAfter.ModTime() {
		t.Error("lock file was modified despite WithoutLock being set")
	}
}
