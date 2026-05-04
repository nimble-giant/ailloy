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
			{Source: "github.com/k/foundry", Subpath: "molds/shortcut", Version: "v0.3.0"},
			{Source: "github.com/k/foundry", Subpath: "molds/linear", Version: "v0.3.0"},
		},
	}

	e := lf.FindEntry("github.com/a/b", "")
	if e == nil {
		t.Fatal("expected entry")
	}
	if e.Version != "v1.0.0" {
		t.Errorf("Version = %q", e.Version)
	}

	if lf.FindEntry("github.com/not/found", "") != nil {
		t.Error("expected nil for missing entry")
	}

	if got := lf.FindEntry("github.com/k/foundry", "molds/linear"); got == nil || got.Subpath != "molds/linear" {
		t.Errorf("FindEntry by subpath linear = %+v", got)
	}
	if got := lf.FindEntry("github.com/k/foundry", "molds/shortcut"); got == nil || got.Subpath != "molds/shortcut" {
		t.Errorf("FindEntry by subpath shortcut = %+v", got)
	}
}

func TestLockFile_FindEntry_Nil(t *testing.T) {
	var lf *LockFile
	if lf.FindEntry("anything", "") != nil {
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

// Regression: two molds locked from the same foundry repo at different
// subpaths must coexist in the lock file, not collapse to one entry.
func TestLockFile_UpsertEntry_SameSourceDifferentSubpaths(t *testing.T) {
	lf := &LockFile{APIVersion: "v1"}
	lf.UpsertEntry(LockEntry{Source: "github.com/k/foundry", Subpath: "molds/shortcut", Version: "v0.3.0"})
	lf.UpsertEntry(LockEntry{Source: "github.com/k/foundry", Subpath: "molds/linear", Version: "v0.3.0"})

	if len(lf.Molds) != 2 {
		t.Fatalf("expected 2 entries (different subpaths), got %d: %+v", len(lf.Molds), lf.Molds)
	}

	// Re-upsert of same (Source, Subpath) still replaces.
	lf.UpsertEntry(LockEntry{Source: "github.com/k/foundry", Subpath: "molds/shortcut", Version: "v0.4.0"})
	if len(lf.Molds) != 2 {
		t.Fatalf("re-upsert should replace, got %d entries", len(lf.Molds))
	}
	if got := lf.FindEntry("github.com/k/foundry", "molds/shortcut"); got == nil || got.Version != "v0.4.0" {
		t.Errorf("upsert did not replace: %+v", got)
	}
}

func TestLockFile_RoundTripIngotsAndOres(t *testing.T) {
	lf := &LockFile{
		APIVersion: "v1",
		Molds:      []LockEntry{{Name: "m", Source: "g/m", Version: "1.0.0"}},
		Ingots:     []LockEntry{{Name: "ig", Source: "g/ig", Version: "1.0.0"}},
		Ores:       []LockEntry{{Name: "or", Source: "g/or", Version: "1.0.0"}},
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "ailloy.lock")
	if err := WriteLockFile(path, lf); err != nil {
		t.Fatal(err)
	}
	got, err := ReadLockFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Molds) != 1 || len(got.Ingots) != 1 || len(got.Ores) != 1 {
		t.Errorf("counts: molds=%d ingots=%d ores=%d", len(got.Molds), len(got.Ingots), len(got.Ores))
	}
}

func TestLockFile_BackCompat_OldMoldsOnly(t *testing.T) {
	old := []byte(`apiVersion: v1
molds:
  - name: m
    source: g/m
    version: 1.0.0
`)
	dir := t.TempDir()
	path := filepath.Join(dir, "ailloy.lock")
	if err := os.WriteFile(path, old, 0644); err != nil {
		t.Fatal(err)
	}
	got, err := ReadLockFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Molds) != 1 {
		t.Errorf("molds: %+v", got.Molds)
	}
	if len(got.Ingots) != 0 || len(got.Ores) != 0 {
		t.Errorf("expected empty Ingots/Ores: %+v / %+v", got.Ingots, got.Ores)
	}
}

func TestLockFile_UpsertArtifactLockIdempotent(t *testing.T) {
	lf := &LockFile{APIVersion: "v1"}
	e := LockEntry{Name: "status", Source: "g/status-ore", Version: "1.0.0"}
	lf.UpsertArtifactLock("ore", e)
	lf.UpsertArtifactLock("ore", LockEntry{Name: "status", Source: "g/status-ore", Version: "1.1.0"})
	if len(lf.Ores) != 1 {
		t.Fatalf("len = %d, want 1: %+v", len(lf.Ores), lf.Ores)
	}
	if lf.Ores[0].Version != "1.1.0" {
		t.Errorf("version should update: %+v", lf.Ores[0])
	}
}

func TestLockFile_UpsertDistinctSubpathsCoexist(t *testing.T) {
	lf := &LockFile{APIVersion: "v1"}
	lf.UpsertArtifactLock("ore", LockEntry{Source: "g/foundry", Subpath: "ore-a", Version: "1.0.0"})
	lf.UpsertArtifactLock("ore", LockEntry{Source: "g/foundry", Subpath: "ore-b", Version: "1.0.0"})
	if len(lf.Ores) != 2 {
		t.Fatalf("two distinct subpaths should coexist; got %d: %+v", len(lf.Ores), lf.Ores)
	}
}

func TestLockFile_UpsertDistinctAliasesCoexist(t *testing.T) {
	lf := &LockFile{APIVersion: "v1"}
	lf.UpsertArtifactLock("ore", LockEntry{Source: "g/status-ore", Version: "1.0.0"})
	lf.UpsertArtifactLock("ore", LockEntry{Source: "g/status-ore", Version: "1.0.0", Alias: "github_status"})
	if len(lf.Ores) != 2 {
		t.Fatalf("alias variant should coexist with canonical; got %d: %+v", len(lf.Ores), lf.Ores)
	}
}

func TestLockFile_All_YieldsAllKinds(t *testing.T) {
	lf := &LockFile{
		APIVersion: "v1",
		Molds:      []LockEntry{{Name: "m"}},
		Ingots:     []LockEntry{{Name: "ig"}},
		Ores:       []LockEntry{{Name: "or"}},
	}
	all := lf.All()
	if len(all) != 3 || all[0].Kind != "mold" || all[1].Kind != "ingot" || all[2].Kind != "ore" {
		t.Errorf("all: %+v", all)
	}
}

func TestLockFile_OldFilesFieldIgnored(t *testing.T) {
	// Pre-migration locks may have files: / fileHashes: keys. With the schema
	// move, those fields no longer exist on LockEntry — YAML unmarshal must
	// silently ignore them rather than failing.
	dir := t.TempDir()
	path := filepath.Join(dir, "ailloy.lock")
	old := `apiVersion: v1
molds:
  - name: nimble-mold
    source: github.com/nimble-giant/nimble-mold
    version: v0.4.0
    commit: abc123
    files:
      - agents.md
    fileHashes:
      agents.md: deadbeef
`
	if err := os.WriteFile(path, []byte(old), 0644); err != nil {
		t.Fatal(err)
	}
	loaded, err := ReadLockFile(path)
	if err != nil {
		t.Fatalf("ReadLockFile: %v", err)
	}
	if len(loaded.Molds) != 1 {
		t.Fatalf("len(Molds) = %d, want 1", len(loaded.Molds))
	}
	if loaded.Molds[0].Version != "v0.4.0" || loaded.Molds[0].Commit != "abc123" {
		t.Errorf("entry not parsed correctly: %+v", loaded.Molds[0])
	}
}
