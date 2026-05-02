package foundry

import (
	"os"
	"path/filepath"
	"reflect"
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

func TestRecordInstalledFiles(t *testing.T) {
	dir := t.TempDir()
	lockPath := filepath.Join(dir, "ailloy.lock")

	seed := &LockFile{
		APIVersion: "v1",
		Molds:      []LockEntry{{Name: "test", Source: "github.com/x/y", Version: "v1", Commit: "c1"}},
	}
	if err := WriteLockFile(lockPath, seed); err != nil {
		t.Fatal(err)
	}

	files := []InstalledFile{
		{RelPath: "b/file.md", SHA256: "hash-b"},
		{RelPath: "a/dir/x.md", SHA256: "hash-a"},
		{RelPath: "a/dir/x.md", SHA256: "hash-a"},
	}

	if err := RecordInstalledFiles(lockPath, "github.com/x/y", files); err != nil {
		t.Fatalf("RecordInstalledFiles: %v", err)
	}

	loaded, err := ReadLockFile(lockPath)
	if err != nil {
		t.Fatal(err)
	}
	got := loaded.Molds[0].Files
	want := []string{"a/dir/x.md", "b/file.md"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Files = %v, want %v", got, want)
	}
	if loaded.Molds[0].FileHashes["a/dir/x.md"] != "hash-a" {
		t.Errorf("hash mismatch: %v", loaded.Molds[0].FileHashes)
	}
}

func TestRecordInstalledFiles_NoLockFile(t *testing.T) {
	err := RecordInstalledFiles("/nonexistent/ailloy.lock", "github.com/x/y", []InstalledFile{{RelPath: "a"}})
	if err == nil {
		t.Error("expected error for missing lockfile")
	}
}

func TestRecordInstalledFiles_EntryNotFound(t *testing.T) {
	dir := t.TempDir()
	lockPath := filepath.Join(dir, "ailloy.lock")
	if err := WriteLockFile(lockPath, &LockFile{APIVersion: "v1"}); err != nil {
		t.Fatal(err)
	}
	err := RecordInstalledFiles(lockPath, "github.com/missing/repo", []InstalledFile{{RelPath: "a"}})
	if err == nil {
		t.Error("expected error when entry not found")
	}
}

// Suppress import-unused warnings for fields used elsewhere.
var _ = time.Time{}
