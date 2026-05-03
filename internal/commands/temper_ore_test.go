package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nimble-giant/ailloy/pkg/mold"
)

// TestTemper_OnMold_ResolvesOreEphemerally verifies that running temper on
// a mold with an ore dep (a) merges the ore schema into the mold's view and
// (b) does not write anything to .ailloy/ores/.
func TestTemper_OnMold_ResolvesOreEphemerally(t *testing.T) {
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
	if err := os.WriteFile(filepath.Join(moldDir, "flux.yaml"), []byte("output:\n  \".\":\n    process: false\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(moldDir, "flux.schema.yaml"), []byte("- name: name\n  type: string\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cwd, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	if err := os.Chdir(moldDir); err != nil {
		t.Fatal(err)
	}

	if err := runTemper(temperCmd, []string{"."}); err != nil {
		t.Fatalf("runTemper: %v", err)
	}

	if _, err := os.Stat(filepath.Join(moldDir, ".ailloy", "ores")); !os.IsNotExist(err) {
		t.Errorf(".ailloy/ores should not exist after temper: %v", err)
	}
}

// TestTemper_OnOreDir_PrefixedEntry_Errors verifies that an ore directory
// whose flux.schema.yaml contains a pre-prefixed entry name (`ore.x` or
// `<oreName>.x`) is reported as an error — entries must be unprefixed.
func TestTemper_OnOreDir_PrefixedEntry_Errors(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "ore.yaml"), []byte(`apiVersion: v1
kind: ore
name: status
version: 1.0.0
`), 0644); err != nil {
		t.Fatal(err)
	}
	// Schema with a pre-prefixed entry (forbidden).
	if err := os.WriteFile(filepath.Join(dir, "flux.schema.yaml"), []byte(`- name: enabled
  type: bool
- name: ore.bad
  type: string
- name: status.also_bad
  type: string
`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "flux.yaml"), []byte("enabled: false\n"), 0644); err != nil {
		t.Fatal(err)
	}

	result := mold.Temper(os.DirFS(dir))
	errs := result.Errors()
	if len(errs) == 0 {
		t.Fatalf("expected errors for pre-prefixed entries, got %+v", result.Diagnostics)
	}
	var sawOrePrefix, sawNamePrefix bool
	for _, e := range errs {
		if strings.Contains(e.Message, "ore.bad") {
			sawOrePrefix = true
		}
		if strings.Contains(e.Message, "status.also_bad") {
			sawNamePrefix = true
		}
	}
	if !sawOrePrefix || !sawNamePrefix {
		t.Errorf("expected errors flagging both ore.bad and status.also_bad, got %+v", errs)
	}
}

// TestTemper_OnOreDir_MissingEnabled_Errors verifies that an ore directory
// without an `enabled: bool` schema entry is reported as an error.
func TestTemper_OnOreDir_MissingEnabled_Errors(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "ore.yaml"), []byte(`apiVersion: v1
kind: ore
name: status
version: 1.0.0
`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "flux.schema.yaml"), []byte(`- name: foo
  type: string
`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "flux.yaml"), []byte("foo: bar\n"), 0644); err != nil {
		t.Fatal(err)
	}

	result := mold.Temper(os.DirFS(dir))
	if !result.HasErrors() {
		t.Fatalf("expected error for missing enabled, got %+v", result.Diagnostics)
	}
	var sawEnabledMissing bool
	for _, e := range result.Errors() {
		if strings.Contains(strings.ToLower(e.Message), "enabled") {
			sawEnabledMissing = true
		}
	}
	if !sawEnabledMissing {
		t.Errorf("expected error mentioning 'enabled', got %+v", result.Errors())
	}
}

// TestTemper_OnOreDir_OrphanDefaults_Warn verifies that flux.yaml leaves
// without matching schema entries are surfaced as warnings (not errors).
func TestTemper_OnOreDir_OrphanDefaults_Warn(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "ore.yaml"), []byte(`apiVersion: v1
kind: ore
name: status
version: 1.0.0
`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "flux.schema.yaml"), []byte(`- name: enabled
  type: bool
`), 0644); err != nil {
		t.Fatal(err)
	}
	// orphan_key has no schema entry → should warn.
	if err := os.WriteFile(filepath.Join(dir, "flux.yaml"), []byte("enabled: false\norphan_key: hello\n"), 0644); err != nil {
		t.Fatal(err)
	}

	result := mold.Temper(os.DirFS(dir))
	if result.HasErrors() {
		t.Fatalf("expected no errors, got %+v", result.Errors())
	}
	var sawOrphan bool
	for _, w := range result.Warnings() {
		if strings.Contains(w.Message, "orphan_key") {
			sawOrphan = true
		}
	}
	if !sawOrphan {
		t.Errorf("expected warning mentioning orphan_key, got %+v", result.Warnings())
	}
}

// TestTemper_OnOreDir_TopLevelOreKey_Errors verifies that flux.yaml with
// a top-level "ore" key is flagged — the wrap-prefix should be absent in
// source files.
func TestTemper_OnOreDir_TopLevelOreKey_Errors(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "ore.yaml"), []byte(`apiVersion: v1
kind: ore
name: status
version: 1.0.0
`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "flux.schema.yaml"), []byte(`- name: enabled
  type: bool
`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "flux.yaml"), []byte("enabled: false\nore:\n  status:\n    enabled: true\n"), 0644); err != nil {
		t.Fatal(err)
	}

	result := mold.Temper(os.DirFS(dir))
	if !result.HasErrors() {
		t.Fatalf("expected error for top-level ore key, got %+v", result.Diagnostics)
	}
	var sawOreKey bool
	for _, e := range result.Errors() {
		if strings.Contains(e.Message, "ore") && strings.Contains(strings.ToLower(e.Message), "top-level") {
			sawOreKey = true
		}
	}
	if !sawOreKey {
		// Accept any message mentioning "ore" as long as it errored.
		for _, e := range result.Errors() {
			if strings.Contains(e.Message, "ore") {
				sawOreKey = true
			}
		}
	}
	if !sawOreKey {
		t.Errorf("expected error mentioning ore wrap key, got %+v", result.Errors())
	}
}
