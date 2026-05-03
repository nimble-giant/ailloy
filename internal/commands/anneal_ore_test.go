package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nimble-giant/ailloy/pkg/blanks"
)

// TestAnneal_LoadsMergedFluxWithOreOverlays verifies that the schema-loading
// path used by anneal includes ore overlay entries. After Phase 10, anneal
// resolves its effective schema via mold.LoadMoldFluxWithOres so the wizard
// prompts cover both the mold's own flux vars and any ore-prefixed keys.
func TestAnneal_LoadsMergedFluxWithOreOverlays(t *testing.T) {
	tmp := t.TempDir()
	cwd, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}

	// Mold's own files.
	if err := os.WriteFile("mold.yaml", []byte(`apiVersion: v1
kind: mold
name: test-mold
version: 1.0.0
`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("flux.schema.yaml", []byte(`- name: project.organization
  type: string
`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("flux.yaml", []byte(``), 0644); err != nil {
		t.Fatal(err)
	}

	// Project-installed ore.
	oreDir := filepath.Join(".ailloy", "ores", "status")
	if err := os.MkdirAll(oreDir, 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(oreDir, "ore.yaml"),
		[]byte("apiVersion: v1\nkind: ore\nname: status\nversion: 1.0.0\n"),
		0644,
	); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(oreDir, "flux.schema.yaml"),
		[]byte("- name: enabled\n  type: bool\n"),
		0644,
	); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(oreDir, "flux.yaml"),
		[]byte("enabled: false\n"),
		0644,
	); err != nil {
		t.Fatal(err)
	}

	reader, err := blanks.NewMoldReaderFromPath(".")
	if err != nil {
		t.Fatal(err)
	}

	schema, _, err := resolveAnnealSchema(reader, false)
	if err != nil {
		t.Fatal(err)
	}

	names := map[string]bool{}
	for _, e := range schema {
		names[e.Name] = true
	}
	if !names["project.organization"] || !names["ore.status.enabled"] {
		t.Errorf("merged schema should include both mold and ore entries: %+v", names)
	}
}
