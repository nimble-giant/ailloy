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
