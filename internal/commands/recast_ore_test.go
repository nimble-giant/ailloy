package commands

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/nimble-giant/ailloy/pkg/foundry"
	"github.com/nimble-giant/ailloy/pkg/mold"
)

// TestRecast_OrePromotesToNewerVersion is intentionally skipped; full
// version-promotion testing requires a fake-foundry helper that produces
// multiple tags, deferred to Phase 12 e2e coverage.
func TestRecast_OrePromotesToNewerVersion(t *testing.T) {
	t.Skip("requires updated buildLocalOre helper that supports versioning — covered by Phase 12 e2e")
}

// TestRecast_DropsOreFromMold_DecrementsDependentsAndGCs verifies the
// cascade-decrement helper used by recast: when a mold no longer declares an
// ore in its mold.yaml, the ore should be stripped of that mold's dependent
// entry and (if no other dependents remain) removed from disk and from the
// installed manifest.
func TestRecast_DropsOreFromMold_DecrementsDependentsAndGCs(t *testing.T) {
	tmp := t.TempDir()
	cwd, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}

	moldName := "test-mold"
	moldKey := "github.com/example/test-mold"

	// Pretend an ore was previously installed for this mold.
	oreInstallDir := filepath.Join(".ailloy", "ores", "status")
	if err := os.MkdirAll(oreInstallDir, 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(oreInstallDir, "ore.yaml"),
		[]byte("apiVersion: v1\nkind: ore\nname: status\nversion: 1.0.0\n"),
		0644,
	); err != nil {
		t.Fatal(err)
	}

	manifest := &foundry.InstalledManifest{
		APIVersion: "v1",
		Molds: []foundry.InstalledEntry{
			{Name: moldName, Source: moldKey, Version: "1.0.0", Commit: "abc", CastAt: time.Now().UTC()},
		},
		Ores: []foundry.ArtifactEntry{
			{
				Name:        "status",
				Source:      "github.com/example/status-ore",
				Version:     "1.0.0",
				Commit:      "def",
				InstalledAt: time.Now().UTC(),
				Dependents:  []string{moldKey},
			},
		},
	}
	manifestPath := filepath.Join(".ailloy", "installed.yaml")
	if err := foundry.WriteInstalledManifest(manifestPath, manifest); err != nil {
		t.Fatal(err)
	}

	// Mold.yaml on disk no longer declares the ore.
	freshMoldDeps := []mold.Dependency{}
	if err := pruneRemovedDeps(manifestPath, moldKey, freshMoldDeps); err != nil {
		t.Fatalf("pruneRemovedDeps: %v", err)
	}

	got, _ := foundry.ReadInstalledManifest(manifestPath)
	if got != nil && len(got.Ores) != 0 {
		t.Errorf("ore should be gone from manifest: %+v", got.Ores)
	}
	if _, err := os.Stat(oreInstallDir); !os.IsNotExist(err) {
		t.Errorf("ore install dir should be gone: %v", err)
	}
}

// TestPruneRemovedDeps_KeepsStillDeclared verifies that deps still declared in
// the mold's manifest are not pruned.
func TestPruneRemovedDeps_KeepsStillDeclared(t *testing.T) {
	tmp := t.TempDir()
	cwd, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}

	moldKey := "github.com/example/test-mold"
	oreInstallDir := filepath.Join(".ailloy", "ores", "status")
	if err := os.MkdirAll(oreInstallDir, 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(oreInstallDir, "ore.yaml"),
		[]byte("apiVersion: v1\nkind: ore\nname: status\nversion: 1.0.0\n"),
		0644,
	); err != nil {
		t.Fatal(err)
	}

	manifest := &foundry.InstalledManifest{
		APIVersion: "v1",
		Ores: []foundry.ArtifactEntry{
			{
				Name:        "status",
				Source:      "github.com/example/status-ore",
				Version:     "1.0.0",
				InstalledAt: time.Now().UTC(),
				Dependents:  []string{moldKey},
			},
		},
	}
	manifestPath := filepath.Join(".ailloy", "installed.yaml")
	if err := foundry.WriteInstalledManifest(manifestPath, manifest); err != nil {
		t.Fatal(err)
	}

	deps := []mold.Dependency{{Ore: "github.com/example/status-ore", Version: "1.0.0"}}
	if err := pruneRemovedDeps(manifestPath, moldKey, deps); err != nil {
		t.Fatalf("pruneRemovedDeps: %v", err)
	}

	got, _ := foundry.ReadInstalledManifest(manifestPath)
	if got == nil || len(got.Ores) != 1 {
		t.Errorf("ore should still be present: %+v", got)
	}
	if _, err := os.Stat(oreInstallDir); err != nil {
		t.Errorf("ore install dir should still be present: %v", err)
	}
}

// TestPruneRemovedDeps_KeepsEntryWithOtherDependents verifies that an ore
// shared across molds keeps its directory and manifest entry when only one
// of its dependents drops it.
func TestPruneRemovedDeps_KeepsEntryWithOtherDependents(t *testing.T) {
	tmp := t.TempDir()
	cwd, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}

	moldKey := "github.com/example/dropping-mold"
	otherMold := "github.com/example/other-mold"

	oreInstallDir := filepath.Join(".ailloy", "ores", "status")
	if err := os.MkdirAll(oreInstallDir, 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(oreInstallDir, "ore.yaml"),
		[]byte("apiVersion: v1\nkind: ore\nname: status\nversion: 1.0.0\n"),
		0644,
	); err != nil {
		t.Fatal(err)
	}

	manifest := &foundry.InstalledManifest{
		APIVersion: "v1",
		Ores: []foundry.ArtifactEntry{
			{
				Name:        "status",
				Source:      "github.com/example/status-ore",
				Version:     "1.0.0",
				InstalledAt: time.Now().UTC(),
				Dependents:  []string{moldKey, otherMold},
			},
		},
	}
	manifestPath := filepath.Join(".ailloy", "installed.yaml")
	if err := foundry.WriteInstalledManifest(manifestPath, manifest); err != nil {
		t.Fatal(err)
	}

	if err := pruneRemovedDeps(manifestPath, moldKey, nil); err != nil {
		t.Fatalf("pruneRemovedDeps: %v", err)
	}

	got, _ := foundry.ReadInstalledManifest(manifestPath)
	if got == nil || len(got.Ores) != 1 {
		t.Fatalf("ore should still be present: %+v", got)
	}
	if got.Ores[0].Dependents == nil || len(got.Ores[0].Dependents) != 1 || got.Ores[0].Dependents[0] != otherMold {
		t.Errorf("expected dependents == [%s]; got %+v", otherMold, got.Ores[0].Dependents)
	}
	if _, err := os.Stat(oreInstallDir); err != nil {
		t.Errorf("ore install dir should still be present: %v", err)
	}
}
