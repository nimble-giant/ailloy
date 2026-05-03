package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nimble-giant/ailloy/pkg/foundry"
	"github.com/nimble-giant/ailloy/pkg/mold"
)

// writeOreFiles drops a minimal ore (ore.yaml + flux.schema.yaml + flux.yaml)
// at the given directory.
func writeOreFiles(t *testing.T, dir, name string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0750); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"ore.yaml":         "apiVersion: v1\nkind: ore\nname: " + name + "\nversion: 1.0.0\n",
		"flux.schema.yaml": "- name: enabled\n  type: bool\n  default: \"false\"\n",
		"flux.yaml":        "enabled: false\n",
	}
	for fn, body := range files {
		if err := os.WriteFile(filepath.Join(dir, fn), []byte(body), 0644); err != nil {
			t.Fatal(err)
		}
	}
}

// TestInstallDeclaredDeps_InstallsMissingOre exercises the dep-walker's
// local-path branch (the e2e fake-foundry helper lands in Phase 12). It
// builds a synthetic ore on disk, declares it as a dependency, and verifies
// the installer copies files into .ailloy/ores/<name>/ and records the mold
// as a dependent in installed.yaml.
func TestInstallDeclaredDeps_InstallsMissingOre(t *testing.T) {
	tmp := t.TempDir()

	remoteOre := filepath.Join(tmp, "remote-ore")
	writeOreFiles(t, remoteOre, "status")

	moldDir := filepath.Join(tmp, "mold")
	if err := os.MkdirAll(moldDir, 0750); err != nil {
		t.Fatal(err)
	}
	cwd, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	if err := os.Chdir(moldDir); err != nil {
		t.Fatal(err)
	}

	manifest := &mold.Mold{
		Name:    "test-mold",
		Version: "1.0.0",
		Dependencies: []mold.Dependency{
			{Ore: remoteOre, Version: "1.0.0"},
		},
	}

	if err := installDeclaredDeps(manifest, "g/test-mold", false); err != nil {
		t.Fatalf("installDeclaredDeps: %v", err)
	}

	if _, err := os.Stat(filepath.Join(moldDir, ".ailloy", "ores", "status", "ore.yaml")); err != nil {
		t.Errorf("ore not installed: %v", err)
	}

	m, err := foundry.ReadInstalledManifest(filepath.Join(moldDir, ".ailloy", "installed.yaml"))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	if m == nil {
		t.Fatal("installed manifest missing")
	}
	entry := m.FindArtifact("ore", "status")
	if entry == nil {
		t.Fatalf("expected status ore in manifest; got %+v", m.Ores)
	}
	if len(entry.Dependents) != 1 || entry.Dependents[0] != "g/test-mold" {
		t.Errorf("dependents = %+v, want [g/test-mold]", entry.Dependents)
	}
}

// TestInstallDeclaredDeps_AppendsDependent verifies that a second mold
// declaring the same ore appends to the Dependents list rather than
// reinstalling.
func TestInstallDeclaredDeps_AppendsDependent(t *testing.T) {
	tmp := t.TempDir()

	remoteOre := filepath.Join(tmp, "remote-ore")
	writeOreFiles(t, remoteOre, "status")

	moldDir := filepath.Join(tmp, "mold")
	if err := os.MkdirAll(moldDir, 0750); err != nil {
		t.Fatal(err)
	}
	cwd, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	if err := os.Chdir(moldDir); err != nil {
		t.Fatal(err)
	}

	manifest := &mold.Mold{
		Name:    "first",
		Version: "1.0.0",
		Dependencies: []mold.Dependency{
			{Ore: remoteOre, Version: "1.0.0"},
		},
	}
	if err := installDeclaredDeps(manifest, "g/first", false); err != nil {
		t.Fatalf("first install: %v", err)
	}
	if err := installDeclaredDeps(manifest, "g/second", false); err != nil {
		t.Fatalf("second install: %v", err)
	}

	m, _ := foundry.ReadInstalledManifest(filepath.Join(moldDir, ".ailloy", "installed.yaml"))
	if m == nil {
		t.Fatal("manifest missing")
	}
	entry := m.FindArtifact("ore", "status")
	if entry == nil {
		t.Fatal("status ore missing")
	}
	if len(entry.Dependents) != 2 {
		t.Errorf("expected 2 dependents, got %v", entry.Dependents)
	}
}

// TestInstallDeclaredDeps_AliasCollisionPreCheck verifies the pre-resolution
// alias collision detection — two ore deps with the same `as:` alias must
// fail before any download.
func TestInstallDeclaredDeps_AliasCollisionPreCheck(t *testing.T) {
	tmp := t.TempDir()

	oreA := filepath.Join(tmp, "ore-a")
	writeOreFiles(t, oreA, "alpha")
	oreB := filepath.Join(tmp, "ore-b")
	writeOreFiles(t, oreB, "beta")

	moldDir := filepath.Join(tmp, "mold")
	if err := os.MkdirAll(moldDir, 0750); err != nil {
		t.Fatal(err)
	}
	cwd, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	if err := os.Chdir(moldDir); err != nil {
		t.Fatal(err)
	}

	manifest := &mold.Mold{
		Name:    "m",
		Version: "1.0.0",
		Dependencies: []mold.Dependency{
			{Ore: oreA, Version: "1.0.0", As: "shared"},
			{Ore: oreB, Version: "1.0.0", As: "shared"},
		},
	}
	err := installDeclaredDeps(manifest, "g/m", false)
	if err == nil {
		t.Fatal("expected alias collision error, got nil")
	}
}

// TestCast_AutoInstallsOreFromMoldYAML covers the cast→install→merge pipeline:
// install an ore from manifest.Dependencies, then load merged schema/defaults
// via LoadMoldFluxWithOres and confirm the ore's namespace is present.
func TestCast_AutoInstallsOreFromMoldYAML(t *testing.T) {
	tmp := t.TempDir()

	remoteOre := filepath.Join(tmp, "remote-ore")
	writeOreFiles(t, remoteOre, "status")

	moldDir := filepath.Join(tmp, "mold")
	if err := os.MkdirAll(moldDir, 0750); err != nil {
		t.Fatal(err)
	}
	cwd, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	if err := os.Chdir(moldDir); err != nil {
		t.Fatal(err)
	}

	// Write a mold.yaml without flux entries — schema comes entirely from
	// the ore overlay.
	moldYAML := []byte("apiVersion: v1\nkind: Mold\nname: m\nversion: 0.1.0\n")
	if err := os.WriteFile("mold.yaml", moldYAML, 0644); err != nil {
		t.Fatal(err)
	}

	manifest := &mold.Mold{
		Name:    "m",
		Version: "0.1.0",
		Dependencies: []mold.Dependency{
			{Ore: remoteOre, Version: "1.0.0"},
		},
	}

	if err := installDeclaredDeps(manifest, "g/m", false); err != nil {
		t.Fatalf("installDeclaredDeps: %v", err)
	}
	if _, err := os.Stat(filepath.Join(".ailloy", "ores", "status", "ore.yaml")); err != nil {
		t.Fatalf("ore not installed: %v", err)
	}

	moldFS := os.DirFS(".")
	paths := buildOreSearchPaths(moldFS, false)

	schema, defaults, _, err := mold.LoadMoldFluxWithOres(moldFS, paths)
	if err != nil {
		t.Fatalf("LoadMoldFluxWithOres: %v", err)
	}

	// Ore namespace should be present in the merged schema.
	foundEnabled := false
	for _, fv := range schema {
		if fv.Name == "ore.status.enabled" {
			foundEnabled = true
			break
		}
	}
	if !foundEnabled {
		t.Errorf("merged schema missing ore.status.enabled; got %+v", schema)
	}

	// Defaults should expose the ore namespace under "ore".
	oreNS, ok := defaults["ore"].(map[string]any)
	if !ok {
		t.Fatalf("expected defaults[\"ore\"] to be a map, got %T", defaults["ore"])
	}
	statusNS, ok := oreNS["status"].(map[string]any)
	if !ok {
		t.Fatalf("expected defaults[\"ore\"][\"status\"] to be a map, got %T", oreNS["status"])
	}
	if statusNS["enabled"] != false {
		t.Errorf("expected ore.status.enabled = false, got %v", statusNS["enabled"])
	}
}
