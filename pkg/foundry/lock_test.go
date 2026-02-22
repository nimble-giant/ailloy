package foundry

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLockFile_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ailloy.lock")

	ts := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)
	original := &LockFile{
		APIVersion: "v1",
		Molds: []LockEntry{
			{
				Name:      "nimble-mold",
				Source:    "github.com/nimble-giant/nimble-mold",
				Version:   "v1.0.0",
				Commit:    "abc123",
				Timestamp: ts,
			},
			{
				Name:      "other-mold",
				Source:    "github.com/other/mold",
				Version:   "v2.0.0",
				Commit:    "def456",
				Subpath:   "sub/path",
				Timestamp: ts,
			},
		},
	}

	if err := WriteLockFile(path, original); err != nil {
		t.Fatalf("WriteLockFile: %v", err)
	}

	loaded, err := ReadLockFile(path)
	if err != nil {
		t.Fatalf("ReadLockFile: %v", err)
	}

	if loaded.APIVersion != "v1" {
		t.Errorf("APIVersion = %q, want v1", loaded.APIVersion)
	}
	if len(loaded.Molds) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(loaded.Molds))
	}

	e := loaded.Molds[0]
	if e.Name != "nimble-mold" {
		t.Errorf("Name = %q, want nimble-mold", e.Name)
	}
	if e.Source != "github.com/nimble-giant/nimble-mold" {
		t.Errorf("Source = %q", e.Source)
	}
	if e.Version != "v1.0.0" {
		t.Errorf("Version = %q", e.Version)
	}
	if e.Commit != "abc123" {
		t.Errorf("Commit = %q", e.Commit)
	}

	e2 := loaded.Molds[1]
	if e2.Subpath != "sub/path" {
		t.Errorf("Subpath = %q, want sub/path", e2.Subpath)
	}
}

func TestReadLockFile_NotFound(t *testing.T) {
	lf, err := ReadLockFile("/nonexistent/ailloy.lock")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if lf != nil {
		t.Error("expected nil for missing file")
	}
}

func TestReadLockFile_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ailloy.lock")
	if err := os.WriteFile(path, []byte(":::invalid"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := ReadLockFile(path)
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestLockFile_FindEntry(t *testing.T) {
	lf := &LockFile{
		Molds: []LockEntry{
			{Source: "github.com/a/b", Version: "v1.0.0"},
			{Source: "github.com/c/d", Version: "v2.0.0"},
		},
	}

	e := lf.FindEntry("github.com/a/b")
	if e == nil {
		t.Fatal("expected entry")
	}
	if e.Version != "v1.0.0" {
		t.Errorf("Version = %q", e.Version)
	}

	if lf.FindEntry("github.com/not/found") != nil {
		t.Error("expected nil for missing entry")
	}
}

func TestLockFile_FindEntry_Nil(t *testing.T) {
	var lf *LockFile
	if lf.FindEntry("anything") != nil {
		t.Error("expected nil on nil LockFile")
	}
}

func TestLockFile_UpsertEntry_New(t *testing.T) {
	lf := &LockFile{APIVersion: "v1"}
	lf.UpsertEntry(LockEntry{Source: "github.com/a/b", Version: "v1.0.0"})

	if len(lf.Molds) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(lf.Molds))
	}
	if lf.Molds[0].Version != "v1.0.0" {
		t.Errorf("Version = %q", lf.Molds[0].Version)
	}
}

func TestLockFile_UpsertEntry_Update(t *testing.T) {
	lf := &LockFile{
		APIVersion: "v1",
		Molds: []LockEntry{
			{Source: "github.com/a/b", Version: "v1.0.0"},
		},
	}
	lf.UpsertEntry(LockEntry{Source: "github.com/a/b", Version: "v2.0.0"})

	if len(lf.Molds) != 1 {
		t.Fatalf("expected 1 entry (updated), got %d", len(lf.Molds))
	}
	if lf.Molds[0].Version != "v2.0.0" {
		t.Errorf("Version = %q, want v2.0.0", lf.Molds[0].Version)
	}
}
