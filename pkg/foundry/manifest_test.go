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
