package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nimble-giant/ailloy/pkg/mold"
)

func TestNewMold_CreatesStructure(t *testing.T) {
	dir := t.TempDir()
	name := "test-mold"

	newMoldOutput = dir
	newMoldNoAgents = false

	err := runNewMold(nil, []string{name})
	if err != nil {
		t.Fatalf("runNewMold returned error: %v", err)
	}

	moldDir := filepath.Join(dir, name)

	expected := []string{
		"mold.yaml",
		"flux.yaml",
		"AGENTS.md",
		"commands/hello.md",
		"skills/helper.md",
	}

	for _, rel := range expected {
		path := filepath.Join(moldDir, rel)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file %s to exist", rel)
		}
	}
}

func TestNewMold_MoldYamlContent(t *testing.T) {
	dir := t.TempDir()
	name := "my-cool-mold"

	newMoldOutput = dir
	newMoldNoAgents = false

	err := runNewMold(nil, []string{name})
	if err != nil {
		t.Fatalf("runNewMold returned error: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, name, "mold.yaml"))
	if err != nil {
		t.Fatalf("failed to read mold.yaml: %v", err)
	}

	yaml := string(content)
	if !strings.Contains(yaml, "name: "+name) {
		t.Errorf("mold.yaml should contain name: %s, got:\n%s", name, yaml)
	}
	if !strings.Contains(yaml, "apiVersion: v1") {
		t.Errorf("mold.yaml should contain apiVersion: v1, got:\n%s", yaml)
	}
	if !strings.Contains(yaml, "kind: mold") {
		t.Errorf("mold.yaml should contain kind: mold, got:\n%s", yaml)
	}
	if !strings.Contains(yaml, "version: 0.1.0") {
		t.Errorf("mold.yaml should contain version: 0.1.0, got:\n%s", yaml)
	}
}

func TestNewMold_AgentsMdIncluded(t *testing.T) {
	dir := t.TempDir()
	name := "agents-test"

	newMoldOutput = dir
	newMoldNoAgents = false

	err := runNewMold(nil, []string{name})
	if err != nil {
		t.Fatalf("runNewMold returned error: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, name, "AGENTS.md"))
	if err != nil {
		t.Fatalf("AGENTS.md should exist: %v", err)
	}

	md := string(content)
	if !strings.Contains(md, "{{project_name}}") {
		t.Errorf("AGENTS.md should contain {{project_name}} template variable, got:\n%s", md)
	}
}

func TestNewMold_NoAgentsFlag(t *testing.T) {
	dir := t.TempDir()
	name := "no-agents-test"

	newMoldOutput = dir
	newMoldNoAgents = true

	err := runNewMold(nil, []string{name})
	if err != nil {
		t.Fatalf("runNewMold returned error: %v", err)
	}

	agentsPath := filepath.Join(dir, name, "AGENTS.md")
	if _, err := os.Stat(agentsPath); !os.IsNotExist(err) {
		t.Errorf("AGENTS.md should not exist when --no-agents is set")
	}

	// Other files should still exist
	moldYaml := filepath.Join(dir, name, "mold.yaml")
	if _, err := os.Stat(moldYaml); os.IsNotExist(err) {
		t.Errorf("mold.yaml should still exist")
	}
}

func TestNewMold_ExistingDirErrors(t *testing.T) {
	dir := t.TempDir()
	name := "existing-dir"

	// Pre-create the directory
	if err := os.MkdirAll(filepath.Join(dir, name), 0750); err != nil {
		t.Fatalf("failed to create test directory: %v", err)
	}

	newMoldOutput = dir
	newMoldNoAgents = false

	err := runNewMold(nil, []string{name})
	if err == nil {
		t.Fatal("expected error when target directory already exists")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("error should mention 'already exists', got: %v", err)
	}
}

func TestNewMold_Temperable(t *testing.T) {
	dir := t.TempDir()
	name := "temper-test"

	newMoldOutput = dir
	newMoldNoAgents = false

	err := runNewMold(nil, []string{name})
	if err != nil {
		t.Fatalf("runNewMold returned error: %v", err)
	}

	moldDir := filepath.Join(dir, name)
	fsys := os.DirFS(moldDir)
	result := mold.Temper(fsys)

	if result.HasErrors() {
		for _, d := range result.Errors() {
			t.Errorf("temper error: %s: %s", d.File, d.Message)
		}
		t.Fatalf("scaffolded mold should pass temper validation")
	}

	if result.Name != name {
		t.Errorf("temper should detect mold name %q, got %q", name, result.Name)
	}
}

func TestNewMold_InvalidName(t *testing.T) {
	dir := t.TempDir()

	newMoldOutput = dir
	newMoldNoAgents = false

	invalidNames := []string{
		"bad/name",
		"bad\\name",
		"bad:name",
		"bad*name",
	}

	for _, name := range invalidNames {
		t.Run(name, func(t *testing.T) {
			err := runNewMold(nil, []string{name})
			if err == nil {
				t.Errorf("expected error for invalid name %q", name)
			}
			if !strings.Contains(err.Error(), "special characters") {
				t.Errorf("error should mention special characters, got: %v", err)
			}
		})
	}
}
