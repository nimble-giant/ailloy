package mold

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
)

func TestParseOre_MalformedYAML(t *testing.T) {
	data := []byte(`{{{not valid yaml`)
	if _, err := ParseOre(data); err == nil {
		t.Fatal("expected error for malformed YAML, got nil")
	}
}

func TestLoadOreFromFS(t *testing.T) {
	yaml := `apiVersion: v1
kind: ore
name: status
version: 2.1.0
description: GitHub Project Status field tracking
`
	fsys := fstest.MapFS{
		"ore.yaml": &fstest.MapFile{Data: []byte(yaml)},
	}

	o, err := LoadOreFromFS(fsys, "ore.yaml")
	if err != nil {
		t.Fatalf("LoadOreFromFS: %v", err)
	}
	if o.Name != "status" {
		t.Errorf("Name = %q; want status", o.Name)
	}
	if o.Version != "2.1.0" {
		t.Errorf("Version = %q; want 2.1.0", o.Version)
	}
	if o.Kind != "ore" {
		t.Errorf("Kind = %q; want ore", o.Kind)
	}
}

func TestLoadOre_MissingFile(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "does-not-exist.yaml")

	_, err := LoadOre(missing)
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("expected wrapped os.ErrNotExist, got: %v", err)
	}
}

func TestOre_EffectiveNamespace_FallsBackToName(t *testing.T) {
	o := &Ore{Name: "status"}
	if got := o.EffectiveNamespace(); got != "status" {
		t.Errorf("EffectiveNamespace() = %q; want status", got)
	}
}

func TestOre_EffectiveNamespace_NamespaceWinsOverName(t *testing.T) {
	o := &Ore{Name: "status_ore", Namespace: "status"}
	if got := o.EffectiveNamespace(); got != "status" {
		t.Errorf("EffectiveNamespace() = %q; want status", got)
	}
}

func TestParseOre_NamespaceField(t *testing.T) {
	data := []byte(`apiVersion: v1
kind: ore
name: status_ore
namespace: status
version: 1.0.0
`)
	ore, err := ParseOre(data)
	if err != nil {
		t.Fatalf("ParseOre: %v", err)
	}
	if ore.Namespace != "status" {
		t.Errorf("Namespace = %q; want status", ore.Namespace)
	}
	if ore.Name != "status_ore" {
		t.Errorf("Name = %q; want status_ore", ore.Name)
	}
}

func TestResolveOreNamespace_Precedence(t *testing.T) {
	// (a) install-dir == ore.Name + no manifest namespace → fallback to name.
	if got := resolveOreNamespace("status", &Ore{Name: "status"}); got != "status" {
		t.Errorf("(a) got %q; want status", got)
	}
	// (b) install-dir == ore.Name + manifest namespace set → publisher namespace wins.
	if got := resolveOreNamespace("status_ore", &Ore{Name: "status_ore", Namespace: "status"}); got != "status" {
		t.Errorf("(b) got %q; want status", got)
	}
	// (c) install-dir != ore.Name → alias is in play, install-dir wins over manifest namespace.
	if got := resolveOreNamespace("github_status", &Ore{Name: "status_ore", Namespace: "status"}); got != "github_status" {
		t.Errorf("(c) got %q; want github_status (alias overrides namespace)", got)
	}
	// (d) install-dir != ore.Name with no namespace → install-dir still wins (alias).
	if got := resolveOreNamespace("foo", &Ore{Name: "status"}); got != "foo" {
		t.Errorf("(d) got %q; want foo", got)
	}
}

func TestParseOre_HappyPath(t *testing.T) {
	data := []byte(`apiVersion: v1
kind: ore
name: status
version: 1.0.0
description: GitHub Project Status field tracking
author:
  name: Nimble Giant
  url: https://github.com/nimble-giant
requires:
  ailloy: ">=0.7.0"
`)

	ore, err := ParseOre(data)
	if err != nil {
		t.Fatalf("ParseOre: %v", err)
	}
	if ore.APIVersion != "v1" {
		t.Errorf("APIVersion = %q; want v1", ore.APIVersion)
	}
	if ore.Kind != "ore" {
		t.Errorf("Kind = %q; want ore", ore.Kind)
	}
	if ore.Name != "status" {
		t.Errorf("Name = %q; want status", ore.Name)
	}
	if ore.Version != "1.0.0" {
		t.Errorf("Version = %q; want 1.0.0", ore.Version)
	}
	if ore.Author.Name != "Nimble Giant" {
		t.Errorf("Author.Name = %q", ore.Author.Name)
	}
	if ore.Requires.Ailloy != ">=0.7.0" {
		t.Errorf("Requires.Ailloy = %q", ore.Requires.Ailloy)
	}
}
