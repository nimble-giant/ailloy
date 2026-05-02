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

func TestResolve_NoLockFile_DoesNotCreateLock(t *testing.T) {
	dir := t.TempDir()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })

	// Apply default config — no opts.
	var cfg resolveConfig
	applyResolveDefaults(&cfg)

	if cfg.lockPath != LockFileName {
		t.Errorf("default lockPath = %q, want %q", cfg.lockPath, LockFileName)
	}

	// Simulate the lock-write decision: if no file at lockPath, no write should happen.
	if _, err := os.Stat(cfg.lockPath); !os.IsNotExist(err) {
		t.Fatalf("expected no lock file in fresh dir, stat err = %v", err)
	}
	if shouldUseLock(cfg.lockPath) {
		t.Errorf("shouldUseLock should return false when lock does not exist")
	}
}

func TestResolve_LockExists_UsesIt(t *testing.T) {
	dir := t.TempDir()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })

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

	if !shouldUseLock(LockFileName) {
		t.Error("shouldUseLock should return true when lock exists")
	}
}

func TestWithLockPath_OverridesDefault(t *testing.T) {
	var cfg resolveConfig
	applyResolveDefaults(&cfg)
	WithLockPath("/tmp/custom.lock")(&cfg)
	if cfg.lockPath != "/tmp/custom.lock" {
		t.Errorf("lockPath = %q, want /tmp/custom.lock", cfg.lockPath)
	}
}

func TestRecordInstalledFiles(t *testing.T) {
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, ".ailloy", "installed.yaml")

	seed := &InstalledManifest{
		APIVersion: "v1",
		Molds:      []InstalledEntry{{Name: "test", Source: "github.com/x/y", Version: "v1", Commit: "c1"}},
	}
	if err := WriteInstalledManifest(manifestPath, seed); err != nil {
		t.Fatal(err)
	}

	files := []InstalledFile{
		{RelPath: "b/file.md", SHA256: "hash-b"},
		{RelPath: "a/dir/x.md", SHA256: "hash-a"},
		{RelPath: "a/dir/x.md", SHA256: "hash-a"},
	}

	if err := RecordInstalledFiles(manifestPath, "github.com/x/y", files); err != nil {
		t.Fatalf("RecordInstalledFiles: %v", err)
	}

	loaded, err := ReadInstalledManifest(manifestPath)
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

func TestRecordInstalledFiles_NoManifest(t *testing.T) {
	err := RecordInstalledFiles("/nonexistent/.ailloy/installed.yaml", "github.com/x/y", []InstalledFile{{RelPath: "a"}})
	if err == nil {
		t.Error("expected error for missing manifest")
	}
}

func TestRecordInstalledFiles_EntryNotFound(t *testing.T) {
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, ".ailloy", "installed.yaml")
	if err := WriteInstalledManifest(manifestPath, &InstalledManifest{APIVersion: "v1"}); err != nil {
		t.Fatal(err)
	}
	err := RecordInstalledFiles(manifestPath, "github.com/missing/repo", []InstalledFile{{RelPath: "a"}})
	if err == nil {
		t.Error("expected error when entry not found")
	}
}

// Suppress import-unused warnings for fields used elsewhere.
var _ = time.Time{}
