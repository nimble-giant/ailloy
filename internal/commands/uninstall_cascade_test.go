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
