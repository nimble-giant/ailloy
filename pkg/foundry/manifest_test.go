package foundry

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestInstalledManifest_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".ailloy", "installed.yaml")

	ts := time.Date(2026, 5, 2, 15, 4, 5, 0, time.UTC)
	original := &InstalledManifest{
		APIVersion: "v1",
		Molds: []InstalledEntry{
			{
				Name:    "nimble-mold",
				Source:  "github.com/nimble-giant/nimble-mold",
				Version: "v0.1.10",
				Commit:  "2347a626798553252668a15dc98dd020ab9a9c0c",
				CastAt:  ts,
			},
			{
				Name:    "other-mold",
				Source:  "github.com/other/mold",
				Subpath: "sub/path",
				Version: "v2.0.0",
				Commit:  "def456",
				CastAt:  ts,
			},
		},
	}

	if err := WriteInstalledManifest(path, original); err != nil {
		t.Fatalf("WriteInstalledManifest: %v", err)
	}

	loaded, err := ReadInstalledManifest(path)
	if err != nil {
		t.Fatalf("ReadInstalledManifest: %v", err)
	}
	if loaded.APIVersion != "v1" {
		t.Errorf("APIVersion = %q, want v1", loaded.APIVersion)
	}
	if len(loaded.Molds) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(loaded.Molds))
	}
	e := loaded.Molds[0]
	if e.Name != "nimble-mold" || e.Source != "github.com/nimble-giant/nimble-mold" {
		t.Errorf("entry 0 mismatch: %+v", e)
	}
	if e.Commit != "2347a626798553252668a15dc98dd020ab9a9c0c" {
		t.Errorf("Commit = %q", e.Commit)
	}
	if !e.CastAt.Equal(ts) {
		t.Errorf("CastAt = %v, want %v", e.CastAt, ts)
	}
	if loaded.Molds[1].Subpath != "sub/path" {
		t.Errorf("Subpath = %q", loaded.Molds[1].Subpath)
	}
}

func TestReadInstalledManifest_MissingFile(t *testing.T) {
	dir := t.TempDir()
	got, err := ReadInstalledManifest(filepath.Join(dir, "missing.yaml"))
	if err != nil {
		t.Fatalf("expected nil error for missing file, got %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil manifest for missing file, got %+v", got)
	}
}

func TestWriteInstalledManifest_CreatesParentDir(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", ".ailloy", "installed.yaml")
	m := &InstalledManifest{APIVersion: "v1"}
	if err := WriteInstalledManifest(path, m); err != nil {
		t.Fatalf("WriteInstalledManifest: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("file not created: %v", err)
	}
}

func TestInstalledManifest_UpsertEntry(t *testing.T) {
	m := &InstalledManifest{APIVersion: "v1"}

	m.UpsertEntry(InstalledEntry{
		Name:    "a",
		Source:  "github.com/x/a",
		Version: "v1.0.0",
		Commit:  "aaa",
	})
	if len(m.Molds) != 1 {
		t.Fatalf("expected 1 entry after first upsert, got %d", len(m.Molds))
	}

	m.UpsertEntry(InstalledEntry{
		Name:    "a",
		Source:  "github.com/x/a",
		Version: "v1.1.0",
		Commit:  "bbb",
	})
	if len(m.Molds) != 1 {
		t.Fatalf("expected 1 entry after re-upsert, got %d", len(m.Molds))
	}
	if m.Molds[0].Version != "v1.1.0" || m.Molds[0].Commit != "bbb" {
		t.Errorf("upsert did not replace existing entry: %+v", m.Molds[0])
	}

	m.UpsertEntry(InstalledEntry{
		Name:    "b",
		Source:  "github.com/x/b",
		Version: "v0.1.0",
		Commit:  "ccc",
	})
	if len(m.Molds) != 2 {
		t.Fatalf("expected 2 entries after second source upsert, got %d", len(m.Molds))
	}
}

func TestInstalledManifest_FindBySource(t *testing.T) {
	m := &InstalledManifest{
		APIVersion: "v1",
		Molds: []InstalledEntry{
			{Name: "a", Source: "github.com/x/a"},
			{Name: "b", Source: "github.com/x/b"},
		},
	}
	if got := m.FindBySource("github.com/x/b"); got == nil || got.Name != "b" {
		t.Errorf("FindBySource(b) = %+v, want entry b", got)
	}
	if got := m.FindBySource("github.com/x/missing"); got != nil {
		t.Errorf("FindBySource(missing) = %+v, want nil", got)
	}
	var nilM *InstalledManifest
	if got := nilM.FindBySource("any"); got != nil {
		t.Errorf("nil manifest FindBySource = %+v, want nil", got)
	}
}
