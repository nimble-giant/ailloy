package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nimble-giant/ailloy/pkg/foundry"
)

func TestRunOreGet_NonRemoteRef_Errors(t *testing.T) {
	err := runOreGet(nil, []string{"./local/path"})
	if err == nil || !strings.Contains(err.Error(), "remote reference") {
		t.Errorf("expected remote-reference error, got %v", err)
	}
}

func TestRunOreAdd_NonRemoteRef_Errors(t *testing.T) {
	err := runOreAdd(nil, []string{"./local/path"})
	if err == nil || !strings.Contains(err.Error(), "remote reference") {
		t.Errorf("expected remote-reference error, got %v", err)
	}
}

func TestRunOreNew_RejectsBadName(t *testing.T) {
	for _, bad := range []string{"BadName", "bad-name", "9start", ""} {
		err := runOreNew(nil, []string{bad})
		if err == nil {
			t.Errorf("expected error for %q", bad)
		}
	}
}

func TestRunOreNew_RejectsExistingDirectory(t *testing.T) {
	dir := t.TempDir()
	cwd, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir("status", 0750); err != nil {
		t.Fatal(err)
	}
	if err := runOreNew(nil, []string{"status"}); err == nil {
		t.Error("expected error when directory exists")
	}
}

func TestRunOreRemove_NotInstalled_Errors(t *testing.T) {
	dir := t.TempDir()
	cwd, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	// No .ailloy/installed.yaml at all.
	if err := runOreRemove(nil, []string{"status"}); err == nil {
		t.Error("expected error when no ores installed")
	}
}

func TestRunOreRemove_HasMoldDependents_ErrorsWithoutForce(t *testing.T) {
	dir := t.TempDir()
	cwd, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	t.Cleanup(func() { oreRemoveForce = false })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	manifest := &foundry.InstalledManifest{
		APIVersion: "v1",
		Ores: []foundry.ArtifactEntry{
			{Name: "status", Source: "g/status-ore", Dependents: []string{"g/some-mold"}},
		},
	}
	manifestPath := filepath.Join(".ailloy", "installed.yaml")
	if err := foundry.WriteInstalledManifest(manifestPath, manifest); err != nil {
		t.Fatal(err)
	}
	if err := runOreRemove(nil, []string{"status"}); err == nil {
		t.Error("expected error when mold depends on ore without --force")
	}
}

func TestRunOreRemove_OnlyUserDependent_Removes(t *testing.T) {
	dir := t.TempDir()
	cwd, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	// Create the ore install dir + manifest entry.
	if err := os.MkdirAll(filepath.Join(".ailloy", "ores", "status"), 0750); err != nil {
		t.Fatal(err)
	}
	manifest := &foundry.InstalledManifest{
		APIVersion: "v1",
		Ores: []foundry.ArtifactEntry{
			{Name: "status", Source: "g/status-ore", Dependents: []string{"user"}},
		},
	}
	manifestPath := filepath.Join(".ailloy", "installed.yaml")
	if err := foundry.WriteInstalledManifest(manifestPath, manifest); err != nil {
		t.Fatal(err)
	}
	if err := runOreRemove(nil, []string{"status"}); err != nil {
		t.Fatalf("runOreRemove: %v", err)
	}
	// Verify dir gone, manifest entry gone.
	if _, err := os.Stat(filepath.Join(".ailloy", "ores", "status")); !os.IsNotExist(err) {
		t.Errorf("ore dir should be gone: %v", err)
	}
	got, _ := foundry.ReadInstalledManifest(manifestPath)
	if got != nil && len(got.Ores) != 0 {
		t.Errorf("ore entry should be gone: %+v", got.Ores)
	}
}

func TestRunOreRemove_Force_OverridesMoldDependents(t *testing.T) {
	dir := t.TempDir()
	cwd, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	t.Cleanup(func() { oreRemoveForce = false })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(".ailloy", "ores", "status"), 0750); err != nil {
		t.Fatal(err)
	}
	manifest := &foundry.InstalledManifest{
		APIVersion: "v1",
		Ores: []foundry.ArtifactEntry{
			{Name: "status", Source: "g/status-ore", Dependents: []string{"g/some-mold"}},
		},
	}
	manifestPath := filepath.Join(".ailloy", "installed.yaml")
	if err := foundry.WriteInstalledManifest(manifestPath, manifest); err != nil {
		t.Fatal(err)
	}
	oreRemoveForce = true
	if err := runOreRemove(nil, []string{"status"}); err != nil {
		t.Fatalf("runOreRemove with --force: %v", err)
	}
	if _, err := os.Stat(filepath.Join(".ailloy", "ores", "status")); !os.IsNotExist(err) {
		t.Errorf("ore dir should be gone: %v", err)
	}
}

func TestRunOreRemove_GlobalScope(t *testing.T) {
	// Verify --global removes from ~/.ailloy/ores/<name> + ~/.ailloy/installed.yaml.
	tmp := t.TempDir()
	t.Setenv("HOME", tmp) // re-route ~/.ailloy/ to a temp dir so the test doesn't pollute real $HOME
	t.Cleanup(func() { oreRemoveGlobal = false })

	globalDir := filepath.Join(tmp, ".ailloy", "ores", "status")
	if err := os.MkdirAll(globalDir, 0750); err != nil {
		t.Fatal(err)
	}

	manifestPath := filepath.Join(tmp, ".ailloy", "installed.yaml")
	manifest := &foundry.InstalledManifest{
		APIVersion: "v1",
		Ores: []foundry.ArtifactEntry{
			{Name: "status", Source: "g/status-ore", Dependents: []string{"user"}},
		},
	}
	if err := foundry.WriteInstalledManifest(manifestPath, manifest); err != nil {
		t.Fatal(err)
	}

	oreRemoveGlobal = true
	if err := runOreRemove(nil, []string{"status"}); err != nil {
		t.Fatalf("runOreRemove --global: %v", err)
	}
	if _, err := os.Stat(globalDir); !os.IsNotExist(err) {
		t.Errorf("global ore dir should be gone: %v", err)
	}
	got, _ := foundry.ReadInstalledManifest(manifestPath)
	if got != nil && len(got.Ores) != 0 {
		t.Errorf("ore entry should be gone: %+v", got.Ores)
	}
}

func TestRunOreNew_CreatesExpectedFiles(t *testing.T) {
	dir := t.TempDir()
	cwd, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	if err := runOreNew(nil, []string{"status"}); err != nil {
		t.Fatalf("runOreNew: %v", err)
	}
	for _, want := range []string{"status/ore.yaml", "status/flux.schema.yaml", "status/flux.yaml"} {
		if _, err := os.Stat(want); err != nil {
			t.Errorf("missing %s: %v", want, err)
		}
	}
}
