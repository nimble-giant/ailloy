package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nimble-giant/ailloy/pkg/foundry/index"
)

func TestNewFoundry_CreatesStructure(t *testing.T) {
	dir := t.TempDir()
	name := "test-foundry"

	newFoundryOutput = dir

	err := runNewFoundry(nil, []string{name})
	if err != nil {
		t.Fatalf("runNewFoundry returned error: %v", err)
	}

	foundryDir := filepath.Join(dir, name)

	expected := []string{
		"foundry.yaml",
		"README.md",
	}

	for _, rel := range expected {
		path := filepath.Join(foundryDir, rel)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file %s to exist", rel)
		}
	}
}

func TestNewFoundry_FoundryYamlContent(t *testing.T) {
	dir := t.TempDir()
	name := "my-cool-foundry"

	newFoundryOutput = dir

	err := runNewFoundry(nil, []string{name})
	if err != nil {
		t.Fatalf("runNewFoundry returned error: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, name, "foundry.yaml"))
	if err != nil {
		t.Fatalf("failed to read foundry.yaml: %v", err)
	}

	yaml := string(content)
	if !strings.Contains(yaml, "name: "+name) {
		t.Errorf("foundry.yaml should contain name: %s, got:\n%s", name, yaml)
	}
	if !strings.Contains(yaml, "apiVersion: v1") {
		t.Errorf("foundry.yaml should contain apiVersion: v1, got:\n%s", yaml)
	}
	if !strings.Contains(yaml, "kind: foundry-index") {
		t.Errorf("foundry.yaml should contain kind: foundry-index, got:\n%s", yaml)
	}
}

func TestNewFoundry_ExistingDirErrors(t *testing.T) {
	dir := t.TempDir()
	name := "existing-dir"

	if err := os.MkdirAll(filepath.Join(dir, name), 0750); err != nil {
		t.Fatalf("failed to create test directory: %v", err)
	}

	newFoundryOutput = dir

	err := runNewFoundry(nil, []string{name})
	if err == nil {
		t.Fatal("expected error when target directory already exists")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("error should mention 'already exists', got: %v", err)
	}
}

func TestNewFoundry_InvalidName(t *testing.T) {
	dir := t.TempDir()

	newFoundryOutput = dir

	invalidNames := []string{
		"bad/name",
		"bad\\name",
		"bad:name",
		"bad*name",
	}

	for _, name := range invalidNames {
		t.Run(name, func(t *testing.T) {
			err := runNewFoundry(nil, []string{name})
			if err == nil {
				t.Errorf("expected error for invalid name %q", name)
			}
			if !strings.Contains(err.Error(), "special characters") {
				t.Errorf("error should mention special characters, got: %v", err)
			}
		})
	}
}

func TestNewFoundry_Parseable(t *testing.T) {
	dir := t.TempDir()
	name := "parse-test"

	newFoundryOutput = dir

	err := runNewFoundry(nil, []string{name})
	if err != nil {
		t.Fatalf("runNewFoundry returned error: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, name, "foundry.yaml"))
	if err != nil {
		t.Fatalf("failed to read foundry.yaml: %v", err)
	}

	idx, err := index.ParseIndex(content)
	if err != nil {
		t.Fatalf("ParseIndex failed: %v", err)
	}
	if err := idx.Validate(); err != nil {
		t.Fatalf("Validate failed: %v", err)
	}
	if idx.Name != name {
		t.Errorf("Name = %q, want %q", idx.Name, name)
	}
}
