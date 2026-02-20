package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunTemper_NoManifest(t *testing.T) {
	dir := t.TempDir()
	err := runTemper(temperCmd, []string{dir})
	if err == nil {
		t.Fatal("expected error when no manifest exists")
	}
	if !strings.Contains(err.Error(), "no mold.yaml or ingot.yaml") {
		t.Errorf("expected 'no mold.yaml or ingot.yaml' error, got: %v", err)
	}
}

func TestRunTemper_ValidMold(t *testing.T) {
	dir := t.TempDir()

	moldYAML := `apiVersion: v1
kind: mold
name: test-mold
version: 1.0.0
commands:
  - hello.md
`
	if err := os.WriteFile(filepath.Join(dir, "mold.yaml"), []byte(moldYAML), 0644); err != nil {
		t.Fatal(err)
	}

	cmdDir := filepath.Join(dir, "claude", "commands")
	if err := os.MkdirAll(cmdDir, 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cmdDir, "hello.md"), []byte("# Hello {{.name}}"), 0644); err != nil {
		t.Fatal(err)
	}

	err := runTemper(temperCmd, []string{dir})
	if err != nil {
		t.Errorf("expected no error for valid mold, got: %v", err)
	}
}

func TestRunTemper_InvalidMold(t *testing.T) {
	dir := t.TempDir()

	// Mold with missing required fields
	moldYAML := `kind: mold
`
	if err := os.WriteFile(filepath.Join(dir, "mold.yaml"), []byte(moldYAML), 0644); err != nil {
		t.Fatal(err)
	}

	err := runTemper(temperCmd, []string{dir})
	if err == nil {
		t.Fatal("expected error for invalid mold")
	}
	if !strings.Contains(err.Error(), "validation failed") {
		t.Errorf("expected 'validation failed' error, got: %v", err)
	}
}

func TestRunTemper_ValidIngot(t *testing.T) {
	dir := t.TempDir()

	ingotYAML := `apiVersion: v1
kind: ingot
name: test-ingot
version: 1.0.0
files:
  - templates/partial.md
`
	if err := os.WriteFile(filepath.Join(dir, "ingot.yaml"), []byte(ingotYAML), 0644); err != nil {
		t.Fatal(err)
	}

	tmplDir := filepath.Join(dir, "templates")
	if err := os.MkdirAll(tmplDir, 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmplDir, "partial.md"), []byte("# Partial"), 0644); err != nil {
		t.Fatal(err)
	}

	err := runTemper(temperCmd, []string{dir})
	if err != nil {
		t.Errorf("expected no error for valid ingot, got: %v", err)
	}
}

func TestRunTemper_DefaultsToCurrentDir(t *testing.T) {
	// Should fail since the test working dir won't have mold.yaml/ingot.yaml at temp path
	dir := t.TempDir()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(origDir) }) //nolint:errcheck
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	err = runTemper(temperCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no manifest in current dir")
	}
}
