package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/nimble-giant/ailloy/pkg/mold"
)

// TestResolveDepsEphemeral_DoesNotWriteToDisk verifies that the ephemeral
// resolver loads ore schemas/defaults from a (possibly cached) source without
// copying anything into .ailloy/ores/. This is the central contract: forge
// and temper must be read-only previews.
func TestResolveDepsEphemeral_DoesNotWriteToDisk(t *testing.T) {
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

	resolver, err := ResolveDepsEphemeral(manifest, true)
	if err != nil {
		t.Fatalf("ResolveDepsEphemeral: %v", err)
	}
	if resolver == nil {
		t.Fatal("nil resolver")
	}

	// .ailloy/ores/ MUST NOT be created by an ephemeral resolve.
	if _, err := os.Stat(filepath.Join(moldDir, ".ailloy", "ores")); !os.IsNotExist(err) {
		t.Errorf(".ailloy/ores should not exist after ephemeral resolve: %v", err)
	}

	// Overlays should include ore.status.enabled.
	if len(resolver.Overlays()) == 0 {
		t.Fatal("no overlays produced")
	}
	found := false
	for _, ov := range resolver.Overlays() {
		for _, e := range ov.Entries {
			if e.Name == "ore.status.enabled" {
				found = true
			}
		}
	}
	if !found {
		t.Errorf("ore.status.enabled not in overlays: %+v", resolver.Overlays())
	}

	// Defaults should expose the ore namespace under "ore".
	oreNS, ok := resolver.Defaults()["ore"].(map[string]any)
	if !ok {
		t.Fatalf("expected defaults[\"ore\"] to be a map, got %T", resolver.Defaults()["ore"])
	}
	statusNS, ok := oreNS["status"].(map[string]any)
	if !ok {
		t.Fatalf("expected defaults[\"ore\"][\"status\"] to be a map, got %T", oreNS["status"])
	}
	if statusNS["enabled"] != false {
		t.Errorf("expected ore.status.enabled = false, got %v", statusNS["enabled"])
	}
}

// TestResolveDepsEphemeral_RejectsLocalDepFromRemoteMold verifies the
// local-dep sandbox: when allowLocalDeps=false (parent mold was remote), a
// local-path ore dep must be refused before any read.
func TestResolveDepsEphemeral_RejectsLocalDepFromRemoteMold(t *testing.T) {
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
		Dependencies: []mold.Dependency{{Ore: remoteOre, Version: "1.0.0"}},
	}
	if _, err := ResolveDepsEphemeral(manifest, false); err == nil {
		t.Fatal("expected error for local-path dep when allowLocalDeps=false")
	}
}

// TestForge_OreNeverWritesToDisk runs forge against a local mold that
// declares an ore dep and confirms the run does not touch .ailloy/ores/.
func TestForge_OreNeverWritesToDisk(t *testing.T) {
	tmp := t.TempDir()

	remoteOre := filepath.Join(tmp, "remote-ore")
	writeOreFiles(t, remoteOre, "status")

	moldDir := filepath.Join(tmp, "mold")
	if err := os.MkdirAll(moldDir, 0750); err != nil {
		t.Fatal(err)
	}
	moldYAML := fmt.Sprintf(`apiVersion: v1
kind: mold
name: test-mold
version: 1.0.0
dependencies:
  - ore: %s
    version: "1.0.0"
`, remoteOre)
	if err := os.WriteFile(filepath.Join(moldDir, "mold.yaml"), []byte(moldYAML), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(moldDir, "flux.yaml"), []byte(`output:
  ".":
    process: false
`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(moldDir, "README.md"), []byte("# test-mold\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cwd, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	if err := os.Chdir(moldDir); err != nil {
		t.Fatal(err)
	}

	if err := runForge(forgeCmd, []string{"."}); err != nil {
		t.Fatalf("runForge: %v", err)
	}

	if _, err := os.Stat(filepath.Join(moldDir, ".ailloy", "ores")); !os.IsNotExist(err) {
		t.Errorf(".ailloy/ores should not exist after forge: %v", err)
	}
}
