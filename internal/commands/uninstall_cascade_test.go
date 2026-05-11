package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nimble-giant/ailloy/pkg/foundry"
)

func TestUninstall_SoleMoldUsingOre_RemovesOre(t *testing.T) {
	tmp := t.TempDir()
	cwd, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}

	moldKey := "github.com/example/test-mold"
	if err := os.MkdirAll(filepath.Join(".ailloy", "ores", "status"), 0750); err != nil {
		t.Fatal(err)
	}
	manifest := &foundry.InstalledManifest{
		APIVersion: "v1",
		Molds:      []foundry.InstalledEntry{{Name: "test-mold", Source: moldKey, Version: "1.0.0"}},
		Ores: []foundry.ArtifactEntry{
			{Name: "status", Source: "github.com/example/status-ore", Version: "1.0.0", Dependents: []string{moldKey}},
		},
	}
	manifestPath := filepath.Join(".ailloy", "installed.yaml")
	if err := foundry.WriteInstalledManifest(manifestPath, manifest); err != nil {
		t.Fatal(err)
	}

	if err := cascadeUninstallArtifacts(manifestPath, moldKey, false); err != nil {
		t.Fatalf("cascadeUninstallArtifacts: %v", err)
	}

	got, _ := foundry.ReadInstalledManifest(manifestPath)
	if got != nil && len(got.Ores) != 0 {
		t.Errorf("ore should be cascade-removed: %+v", got.Ores)
	}
	if _, err := os.Stat(filepath.Join(".ailloy", "ores", "status")); !os.IsNotExist(err) {
		t.Errorf("ore install dir should be gone: %v", err)
	}
}

func TestUninstall_TwoMoldsShareOre_FirstUninstallLeavesOre(t *testing.T) {
	tmp := t.TempDir()
	cwd, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}

	moldA := "github.com/example/mold-a"
	moldB := "github.com/example/mold-b"
	if err := os.MkdirAll(filepath.Join(".ailloy", "ores", "status"), 0750); err != nil {
		t.Fatal(err)
	}
	manifest := &foundry.InstalledManifest{
		APIVersion: "v1",
		Ores: []foundry.ArtifactEntry{
			{Name: "status", Source: "github.com/example/status-ore", Dependents: []string{moldA, moldB}},
		},
	}
	manifestPath := filepath.Join(".ailloy", "installed.yaml")
	if err := foundry.WriteInstalledManifest(manifestPath, manifest); err != nil {
		t.Fatal(err)
	}

	if err := cascadeUninstallArtifacts(manifestPath, moldA, false); err != nil {
		t.Fatalf("cascadeUninstallArtifacts: %v", err)
	}

	got, _ := foundry.ReadInstalledManifest(manifestPath)
	if got == nil || len(got.Ores) != 1 {
		t.Fatalf("ore should remain: %+v", got)
	}
	if len(got.Ores[0].Dependents) != 1 || got.Ores[0].Dependents[0] != moldB {
		t.Errorf("only moldB should remain in dependents: %+v", got.Ores[0].Dependents)
	}
	if _, err := os.Stat(filepath.Join(".ailloy", "ores", "status")); err != nil {
		t.Errorf("ore install dir should still exist: %v", err)
	}
}

func TestCascadeUninstallTransitiveMolds_OrphanRemoved(t *testing.T) {
	tmp := t.TempDir()
	cwd, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}

	parentKey := "github.com/example/parent"
	leafSrc := "github.com/example/leaf"
	manifestPath := filepath.Join(".ailloy", "installed.yaml")

	// Pre-stage a transitive entry whose only parent is the one being removed,
	// AND lay down its installed file so UninstallMold has something to remove.
	leafFile := "leaf-out.md"
	if err := os.WriteFile(leafFile, []byte("# leaf\n"), 0644); err != nil {
		t.Fatal(err)
	}
	manifest := &foundry.InstalledManifest{
		APIVersion: "v1",
		Molds: []foundry.InstalledEntry{
			{
				Name: "leaf", Source: leafSrc, Version: "v1.0.0", Commit: "abc",
				InstalledAs: "transitive",
				InstalledBy: []string{parentKey},
				Files:       []string{leafFile},
				FileHashes:  map[string]string{leafFile: hashContent(t, leafFile)},
			},
		},
	}
	if err := foundry.WriteInstalledManifest(manifestPath, manifest); err != nil {
		t.Fatal(err)
	}

	if err := cascadeUninstallTransitiveMolds(manifestPath, parentKey, false, false); err != nil {
		t.Fatalf("cascade: %v", err)
	}

	got, _ := foundry.ReadInstalledManifest(manifestPath)
	if got != nil && len(got.Molds) != 0 {
		t.Errorf("orphan transitive should be removed: %+v", got.Molds)
	}
	if _, err := os.Stat(leafFile); !os.IsNotExist(err) {
		t.Errorf("orphan's file should be gone: %v", err)
	}
}

func TestCascadeUninstallTransitiveMolds_DirectKeptStripsParent(t *testing.T) {
	tmp := t.TempDir()
	cwd, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}

	parentKey := "github.com/example/parent"
	manifestPath := filepath.Join(".ailloy", "installed.yaml")

	// Direct cast that happens to also list parent in InstalledBy. Must NOT
	// be removed; parent should just be stripped from InstalledBy.
	manifest := &foundry.InstalledManifest{
		APIVersion: "v1",
		Molds: []foundry.InstalledEntry{
			{
				Name: "directly-cast", Source: "github.com/example/d", Version: "1.0.0",
				InstalledAs: "direct",
				InstalledBy: []string{parentKey},
			},
		},
	}
	if err := foundry.WriteInstalledManifest(manifestPath, manifest); err != nil {
		t.Fatal(err)
	}

	if err := cascadeUninstallTransitiveMolds(manifestPath, parentKey, false, false); err != nil {
		t.Fatalf("cascade: %v", err)
	}

	got, _ := foundry.ReadInstalledManifest(manifestPath)
	if got == nil || len(got.Molds) != 1 {
		t.Fatalf("direct mold must remain: %+v", got)
	}
	if len(got.Molds[0].InstalledBy) != 0 {
		t.Errorf("parent edge should be stripped: %+v", got.Molds[0].InstalledBy)
	}
}

func TestCascadeUninstallTransitiveMolds_SharedTransitiveKept(t *testing.T) {
	tmp := t.TempDir()
	cwd, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}

	parentA := "github.com/example/a"
	parentB := "github.com/example/b"
	manifestPath := filepath.Join(".ailloy", "installed.yaml")

	manifest := &foundry.InstalledManifest{
		APIVersion: "v1",
		Molds: []foundry.InstalledEntry{
			{
				Name: "shared", Source: "github.com/example/shared", Version: "1.0.0",
				InstalledAs: "transitive",
				InstalledBy: []string{parentA, parentB},
			},
		},
	}
	if err := foundry.WriteInstalledManifest(manifestPath, manifest); err != nil {
		t.Fatal(err)
	}

	// Remove parentA; shared has parentB still — must be kept.
	if err := cascadeUninstallTransitiveMolds(manifestPath, parentA, false, false); err != nil {
		t.Fatalf("cascade: %v", err)
	}

	got, _ := foundry.ReadInstalledManifest(manifestPath)
	if got == nil || len(got.Molds) != 1 {
		t.Fatalf("shared transitive should remain: %+v", got)
	}
	if len(got.Molds[0].InstalledBy) != 1 || got.Molds[0].InstalledBy[0] != parentB {
		t.Errorf("InstalledBy = %v; want [%s]", got.Molds[0].InstalledBy, parentB)
	}
}

// hashContent computes the sha256 hex of a file's contents — used by the
// transitive cascade tests to populate FileHashes so UninstallMold doesn't
// flag the file as user-modified.
func hashContent(t *testing.T, path string) string {
	t.Helper()
	got, err := hashFileForDeps(path)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func TestUninstall_OreWithUserSentinel_NotAutoRemoved(t *testing.T) {
	tmp := t.TempDir()
	cwd, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}

	moldKey := "github.com/example/mold"
	if err := os.MkdirAll(filepath.Join(".ailloy", "ores", "status"), 0750); err != nil {
		t.Fatal(err)
	}
	manifest := &foundry.InstalledManifest{
		APIVersion: "v1",
		Ores: []foundry.ArtifactEntry{
			{Name: "status", Source: "github.com/example/status-ore", Dependents: []string{moldKey, "user"}},
		},
	}
	manifestPath := filepath.Join(".ailloy", "installed.yaml")
	if err := foundry.WriteInstalledManifest(manifestPath, manifest); err != nil {
		t.Fatal(err)
	}

	if err := cascadeUninstallArtifacts(manifestPath, moldKey, false); err != nil {
		t.Fatalf("cascadeUninstallArtifacts: %v", err)
	}

	got, _ := foundry.ReadInstalledManifest(manifestPath)
	if got == nil || len(got.Ores) != 1 {
		t.Fatalf("ore should remain (user sentinel): %+v", got)
	}
	if len(got.Ores[0].Dependents) != 1 || got.Ores[0].Dependents[0] != "user" {
		t.Errorf("user sentinel should remain: %+v", got.Ores[0].Dependents)
	}
}
