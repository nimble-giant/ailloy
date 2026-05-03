package mold

import (
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
