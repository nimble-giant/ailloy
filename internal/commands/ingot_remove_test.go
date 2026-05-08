package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nimble-giant/ailloy/pkg/foundry"
)

func TestRunIngotRemove_NotInstalled_Errors(t *testing.T) {
	dir := t.TempDir()
	cwd, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	if err := runIngotRemove(nil, []string{"github-patterns"}); err == nil {
		t.Error("expected error when no ingots installed")
	}
}

func TestRunIngotRemove_HasMoldDependents_ErrorsWithoutForce(t *testing.T) {
	dir := t.TempDir()
	cwd, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	t.Cleanup(func() { ingotRemoveForce = false })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	manifest := &foundry.InstalledManifest{
		APIVersion: "v1",
		Ingots: []foundry.ArtifactEntry{
			{Name: "github-patterns", Source: "g/gp", Dependents: []string{"g/some-mold"}},
		},
	}
	manifestPath := filepath.Join(".ailloy", "installed.yaml")
	if err := foundry.WriteInstalledManifest(manifestPath, manifest); err != nil {
		t.Fatal(err)
	}
	if err := runIngotRemove(nil, []string{"github-patterns"}); err == nil {
		t.Error("expected error when mold depends on ingot without --force")
	}
}

func TestRunIngotRemove_OnlyUserDependent_Removes(t *testing.T) {
	dir := t.TempDir()
	cwd, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(".ailloy", "ingots", "github-patterns"), 0750); err != nil {
		t.Fatal(err)
	}
	manifest := &foundry.InstalledManifest{
		APIVersion: "v1",
		Ingots: []foundry.ArtifactEntry{
			{Name: "github-patterns", Source: "g/gp", Dependents: []string{"user"}},
		},
	}
	manifestPath := filepath.Join(".ailloy", "installed.yaml")
	if err := foundry.WriteInstalledManifest(manifestPath, manifest); err != nil {
		t.Fatal(err)
	}
	if err := runIngotRemove(nil, []string{"github-patterns"}); err != nil {
		t.Fatalf("runIngotRemove: %v", err)
	}
	if _, err := os.Stat(filepath.Join(".ailloy", "ingots", "github-patterns")); !os.IsNotExist(err) {
		t.Errorf("ingot dir should be gone: %v", err)
	}
	got, _ := foundry.ReadInstalledManifest(manifestPath)
	if got != nil && len(got.Ingots) != 0 {
		t.Errorf("ingot entry should be gone: %+v", got.Ingots)
	}
}

func TestRunIngotRemove_Force_OverridesMoldDependents(t *testing.T) {
	dir := t.TempDir()
	cwd, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	t.Cleanup(func() { ingotRemoveForce = false })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(".ailloy", "ingots", "github-patterns"), 0750); err != nil {
		t.Fatal(err)
	}
	manifest := &foundry.InstalledManifest{
		APIVersion: "v1",
		Ingots: []foundry.ArtifactEntry{
			{Name: "github-patterns", Source: "g/gp", Dependents: []string{"g/some-mold"}},
		},
	}
	manifestPath := filepath.Join(".ailloy", "installed.yaml")
	if err := foundry.WriteInstalledManifest(manifestPath, manifest); err != nil {
		t.Fatal(err)
	}
	ingotRemoveForce = true
	if err := runIngotRemove(nil, []string{"github-patterns"}); err != nil {
		t.Fatalf("runIngotRemove with --force: %v", err)
	}
	if _, err := os.Stat(filepath.Join(".ailloy", "ingots", "github-patterns")); !os.IsNotExist(err) {
		t.Errorf("ingot dir should be gone: %v", err)
	}
}

func TestRunIngotRemove_GlobalScope(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Cleanup(func() { ingotRemoveGlobal = false })

	globalDir := filepath.Join(tmp, ".ailloy", "ingots", "github-patterns")
	if err := os.MkdirAll(globalDir, 0750); err != nil {
		t.Fatal(err)
	}

	manifestPath := filepath.Join(tmp, ".ailloy", "installed.yaml")
	manifest := &foundry.InstalledManifest{
		APIVersion: "v1",
		Ingots: []foundry.ArtifactEntry{
			{Name: "github-patterns", Source: "g/gp", Dependents: []string{"user"}},
		},
	}
	if err := foundry.WriteInstalledManifest(manifestPath, manifest); err != nil {
		t.Fatal(err)
	}

	ingotRemoveGlobal = true
	if err := runIngotRemove(nil, []string{"github-patterns"}); err != nil {
		t.Fatalf("runIngotRemove --global: %v", err)
	}
	if _, err := os.Stat(globalDir); !os.IsNotExist(err) {
		t.Errorf("global ingot dir should be gone: %v", err)
	}
	got, _ := foundry.ReadInstalledManifest(manifestPath)
	if got != nil && len(got.Ingots) != 0 {
		t.Errorf("ingot entry should be gone: %+v", got.Ingots)
	}
}
