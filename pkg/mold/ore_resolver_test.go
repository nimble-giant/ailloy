package mold

import (
	"testing"
	"testing/fstest"
)

func TestOreResolver_LoadsOreFromDir_PrefixesEntries(t *testing.T) {
	oreFS := fstest.MapFS{
		"ores/status/ore.yaml": &fstest.MapFile{Data: []byte(`apiVersion: v1
kind: ore
name: status
version: 1.0.0
`)},
		"ores/status/flux.schema.yaml": &fstest.MapFile{Data: []byte(`- name: enabled
  type: bool
  default: "false"
- name: field_id
  type: string
`)},
		"ores/status/flux.yaml": &fstest.MapFile{Data: []byte(`enabled: false
field_id: ""
`)},
	}

	overlays, defaults, err := LoadOreOverlaysFromFS(oreFS, "ores", nil)
	if err != nil {
		t.Fatalf("LoadOreOverlaysFromFS: %v", err)
	}
	if len(overlays) != 1 {
		t.Fatalf("len(overlays) = %d, want 1: %+v", len(overlays), overlays)
	}
	got := overlays[0].Entries
	if len(got) != 2 || got[0].Name != "ore.status.enabled" || got[1].Name != "ore.status.field_id" {
		t.Errorf("entries should be prefixed: %+v", got)
	}
	statusDefaults, ok := defaults["ore"].(map[string]any)["status"].(map[string]any)
	if !ok {
		t.Fatalf("defaults missing ore.status: %+v", defaults)
	}
	if statusDefaults["enabled"] != false {
		t.Errorf("defaults.enabled = %v", statusDefaults["enabled"])
	}
}

func TestOreResolver_AliasOverridesPrefix(t *testing.T) {
	oreFS := fstest.MapFS{
		"ores/foo/ore.yaml":         &fstest.MapFile{Data: []byte("apiVersion: v1\nkind: ore\nname: status\nversion: 1.0.0\n")},
		"ores/foo/flux.schema.yaml": &fstest.MapFile{Data: []byte("- name: enabled\n  type: bool\n")},
		"ores/foo/flux.yaml":        &fstest.MapFile{Data: []byte("enabled: false\n")},
	}
	// Install dir "foo" overrides manifest name "status".
	overlays, defaults, err := LoadOreOverlaysFromFS(oreFS, "ores", nil)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(overlays) != 1 || overlays[0].Entries[0].Name != "ore.foo.enabled" {
		t.Errorf("alias should override prefix: %+v", overlays)
	}
	if _, ok := defaults["ore"].(map[string]any)["foo"]; !ok {
		t.Errorf("defaults should be under ore.foo: %+v", defaults)
	}
}

func TestLoadOreOverlays_HigherPriorityWins(t *testing.T) {
	// Same ore name "status" in two FSes. Caller passes a `seen` map; the
	// first call (mold-local, higher priority) registers "status" into seen,
	// and the second call (project, lower priority) skips it.
	molLocal := fstest.MapFS{
		"ores/status/ore.yaml":         &fstest.MapFile{Data: []byte("apiVersion: v1\nkind: ore\nname: status\nversion: 1.0.0\n")},
		"ores/status/flux.schema.yaml": &fstest.MapFile{Data: []byte("- name: enabled\n  type: bool\n  description: from-mold-local\n")},
		"ores/status/flux.yaml":        &fstest.MapFile{Data: []byte("enabled: false\n")},
	}
	project := fstest.MapFS{
		"ores/status/ore.yaml":         &fstest.MapFile{Data: []byte("apiVersion: v1\nkind: ore\nname: status\nversion: 1.0.0\n")},
		"ores/status/flux.schema.yaml": &fstest.MapFile{Data: []byte("- name: enabled\n  type: bool\n  description: from-project\n")},
		"ores/status/flux.yaml":        &fstest.MapFile{Data: []byte("enabled: false\n")},
	}

	seen := map[string]struct{}{}
	moldLocalOverlays, _, err := LoadOreOverlaysFromFS(molLocal, "ores", seen)
	if err != nil {
		t.Fatal(err)
	}
	projectOverlays, _, err := LoadOreOverlaysFromFS(project, "ores", seen)
	if err != nil {
		t.Fatal(err)
	}

	if len(moldLocalOverlays) != 1 {
		t.Fatalf("mold-local should load: %+v", moldLocalOverlays)
	}
	if moldLocalOverlays[0].Entries[0].Description != "from-mold-local" {
		t.Errorf("expected mold-local description; got %+v", moldLocalOverlays[0].Entries[0])
	}
	if len(projectOverlays) != 0 {
		t.Errorf("project should be skipped because mold-local already loaded 'status': %+v", projectOverlays)
	}
}
