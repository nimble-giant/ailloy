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

// Regression for the bug where two molds installed from the same foundry repo
// at different subpaths collapsed into a single manifest entry: the second
// install overwrote the first because UpsertEntry matched by Source alone.
func TestInstalledManifest_UpsertEntry_SameSourceDifferentSubpaths(t *testing.T) {
	m := &InstalledManifest{APIVersion: "v1"}

	m.UpsertEntry(InstalledEntry{
		Name:    "shortcut",
		Source:  "github.com/kriscoleman/replicated-foundry",
		Subpath: "molds/shortcut",
		Version: "v0.3.0",
		Commit:  "aaa",
	})
	m.UpsertEntry(InstalledEntry{
		Name:    "linear",
		Source:  "github.com/kriscoleman/replicated-foundry",
		Subpath: "molds/linear",
		Version: "v0.3.0",
		Commit:  "bbb",
	})
	if len(m.Molds) != 2 {
		t.Fatalf("expected 2 entries (different subpaths), got %d: %+v", len(m.Molds), m.Molds)
	}

	// Re-upserting the same (Source, Subpath) MUST still replace.
	m.UpsertEntry(InstalledEntry{
		Name:    "shortcut",
		Source:  "github.com/kriscoleman/replicated-foundry",
		Subpath: "molds/shortcut",
		Version: "v0.4.0",
		Commit:  "ccc",
	})
	if len(m.Molds) != 2 {
		t.Fatalf("re-upsert of same (source,subpath) should replace, not append: got %d", len(m.Molds))
	}
	if got := m.FindBySource("github.com/kriscoleman/replicated-foundry", "molds/shortcut"); got == nil || got.Version != "v0.4.0" {
		t.Errorf("upsert did not replace existing entry: %+v", got)
	}
}

func TestInstalledManifest_FindBySource(t *testing.T) {
	m := &InstalledManifest{
		APIVersion: "v1",
		Molds: []InstalledEntry{
			{Name: "a", Source: "github.com/x/a"},
			{Name: "b", Source: "github.com/x/b"},
			{Name: "shortcut", Source: "github.com/k/foundry", Subpath: "molds/shortcut"},
			{Name: "linear", Source: "github.com/k/foundry", Subpath: "molds/linear"},
		},
	}
	if got := m.FindBySource("github.com/x/b", ""); got == nil || got.Name != "b" {
		t.Errorf("FindBySource(b) = %+v, want entry b", got)
	}
	if got := m.FindBySource("github.com/x/missing", ""); got != nil {
		t.Errorf("FindBySource(missing) = %+v, want nil", got)
	}
	if got := m.FindBySource("github.com/k/foundry", "molds/linear"); got == nil || got.Name != "linear" {
		t.Errorf("FindBySource(foundry, molds/linear) = %+v, want linear", got)
	}
	if got := m.FindBySource("github.com/k/foundry", "molds/shortcut"); got == nil || got.Name != "shortcut" {
		t.Errorf("FindBySource(foundry, molds/shortcut) = %+v, want shortcut", got)
	}
	var nilM *InstalledManifest
	if got := nilM.FindBySource("any", ""); got != nil {
		t.Errorf("nil manifest FindBySource = %+v, want nil", got)
	}
}

func TestInstalledManifest_RoundTripIngotsAndOres(t *testing.T) {
	m := &InstalledManifest{
		APIVersion: "v1",
		Molds:      []InstalledEntry{{Name: "m", Source: "g/m", Version: "1.0.0"}},
		Ingots: []ArtifactEntry{
			{Name: "github-patterns", Source: "g/gp", Version: "1.0.0", Dependents: []string{"g/m"}},
		},
		Ores: []ArtifactEntry{
			{Name: "status", Source: "g/status-ore", Version: "1.0.0", Alias: "", Dependents: []string{"g/m", "user"}},
		},
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "installed.yaml")
	if err := WriteInstalledManifest(path, m); err != nil {
		t.Fatal(err)
	}
	got, err := ReadInstalledManifest(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Ingots) != 1 || got.Ingots[0].Name != "github-patterns" {
		t.Errorf("ingots: %+v", got.Ingots)
	}
	if len(got.Ores) != 1 || got.Ores[0].Name != "status" || got.Ores[0].Dependents[1] != "user" {
		t.Errorf("ores: %+v", got.Ores)
	}
}

func TestInstalledManifest_BackCompat_OldMoldsOnly(t *testing.T) {
	old := []byte(`apiVersion: v1
molds:
  - name: m
    source: g/m
    version: 1.0.0
`)
	dir := t.TempDir()
	path := filepath.Join(dir, "installed.yaml")
	if err := os.WriteFile(path, old, 0644); err != nil {
		t.Fatal(err)
	}
	got, err := ReadInstalledManifest(path)
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

func TestArtifact_UpsertIdempotent(t *testing.T) {
	m := &InstalledManifest{APIVersion: "v1"}
	entry := ArtifactEntry{Name: "status", Source: "g/status-ore", Version: "1.0.0", Dependents: []string{"g/m1"}}
	m.UpsertArtifact("ore", entry)
	// Re-upsert with another dependent — should merge dependents, not duplicate.
	m.UpsertArtifact("ore", ArtifactEntry{Name: "status", Source: "g/status-ore", Version: "1.0.0", Dependents: []string{"g/m1", "g/m2"}})
	if len(m.Ores) != 1 {
		t.Fatalf("len = %d, want 1: %+v", len(m.Ores), m.Ores)
	}
	if len(m.Ores[0].Dependents) != 2 {
		t.Errorf("dependents should merge to 2: %+v", m.Ores[0].Dependents)
	}
}

func TestArtifact_UpsertDistinctSubpathsCoexist(t *testing.T) {
	m := &InstalledManifest{APIVersion: "v1"}
	m.UpsertArtifact("ore", ArtifactEntry{Name: "a", Source: "g/foundry", Subpath: "ore-a", Version: "1.0.0"})
	m.UpsertArtifact("ore", ArtifactEntry{Name: "b", Source: "g/foundry", Subpath: "ore-b", Version: "1.0.0"})
	if len(m.Ores) != 2 {
		t.Fatalf("two distinct subpaths should coexist; got %d: %+v", len(m.Ores), m.Ores)
	}
}

func TestArtifact_UpsertDistinctAliasesCoexist(t *testing.T) {
	m := &InstalledManifest{APIVersion: "v1"}
	m.UpsertArtifact("ore", ArtifactEntry{Name: "status", Source: "g/status-ore", Version: "1.0.0"})
	m.UpsertArtifact("ore", ArtifactEntry{Name: "status", Source: "g/status-ore", Version: "1.0.0", Alias: "github_status"})
	if len(m.Ores) != 2 {
		t.Fatalf("alias variant should coexist with canonical; got %d: %+v", len(m.Ores), m.Ores)
	}
}

func TestArtifact_RemoveDependent_EmptyTriggersOrphan(t *testing.T) {
	m := &InstalledManifest{
		APIVersion: "v1",
		Ores: []ArtifactEntry{
			{Name: "status", Source: "g/status-ore", Dependents: []string{"g/m1"}},
			{Name: "shared", Source: "g/shared-ore", Dependents: []string{"g/m1", "g/m2"}},
		},
	}
	orphans := m.RemoveDependent("g/m1")
	// "status" had only g/m1 → becomes orphan and is removed; "shared" keeps g/m2.
	if len(orphans) != 1 || orphans[0].Name != "status" {
		t.Errorf("orphans: %+v", orphans)
	}
	if len(m.Ores) != 1 || m.Ores[0].Name != "shared" {
		t.Errorf("expected only 'shared' remaining: %+v", m.Ores)
	}
}

func TestArtifact_FindByName_HonorsAlias(t *testing.T) {
	m := &InstalledManifest{
		APIVersion: "v1",
		Ores: []ArtifactEntry{
			{Name: "status", Source: "g/status-ore-a"},                         // canonical name
			{Name: "status", Source: "g/status-ore-b", Alias: "github_status"}, // installed --as
		},
	}
	if e := m.FindArtifact("ore", "status"); e == nil || e.Source != "g/status-ore-a" {
		t.Errorf("canonical lookup: %+v", e)
	}
	if e := m.FindArtifact("ore", "github_status"); e == nil || e.Source != "g/status-ore-b" {
		t.Errorf("alias lookup: %+v", e)
	}
	if e := m.FindArtifact("ore", "missing"); e != nil {
		t.Errorf("missing should be nil: %+v", e)
	}
}

func TestArtifact_All_YieldsAllKinds(t *testing.T) {
	m := &InstalledManifest{
		APIVersion: "v1",
		Molds:      []InstalledEntry{{Name: "m"}},
		Ingots:     []ArtifactEntry{{Name: "ig"}},
		Ores:       []ArtifactEntry{{Name: "or"}},
	}
	all := m.All()
	if len(all) != 3 {
		t.Fatalf("len = %d, want 3: %+v", len(all), all)
	}
	if all[0].Kind != "mold" || all[1].Kind != "ingot" || all[2].Kind != "ore" {
		t.Errorf("ordering: %+v", all)
	}
}

func TestManifest_AbsentDoesNotBreakLockReads(t *testing.T) {
	dir := t.TempDir()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })

	// Pre-existing lock, no manifest — typical pre-upgrade state.
	lock := &LockFile{
		APIVersion: "v1",
		Molds: []LockEntry{{
			Name: "x", Source: "github.com/o/x", Version: "v1.0.0", Commit: "abc",
			Timestamp: time.Now().UTC(),
		}},
	}
	if err := WriteLockFile(LockFileName, lock); err != nil {
		t.Fatal(err)
	}

	manifest, err := ReadInstalledManifest(InstalledManifestPath)
	if err != nil {
		t.Fatalf("ReadInstalledManifest should not error on missing file: %v", err)
	}
	if manifest != nil {
		t.Errorf("expected nil manifest, got %+v", manifest)
	}
	if !shouldUseLock(LockFileName) {
		t.Error("shouldUseLock should still be true when only lock exists")
	}
}
