package mold_test

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/nimble-giant/ailloy/pkg/mold"
)

func TestFetchSchemaFromLocalDir(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "mold.yaml"), []byte("name: demo\nversion: 0.0.1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "flux.schema.yaml"), []byte(`- name: agents.targets
  type: list
  description: which agents to install
- name: agents.parallel
  type: bool
  default: "true"
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "flux.yaml"), []byte("agents:\n  targets: []\n  parallel: true\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	schema, defaults, err := mold.FetchSchemaFromSource(context.Background(), dir)
	if err != nil {
		t.Fatalf("FetchSchemaFromSource: %v", err)
	}
	if len(schema) != 2 {
		t.Fatalf("schema len = %d want 2", len(schema))
	}
	if schema[0].Name != "agents.targets" || schema[0].Type != "list" {
		t.Fatalf("schema[0] = %+v", schema[0])
	}
	if defaults["agents"] == nil {
		t.Fatalf("expected defaults[agents] to be present, got %+v", defaults)
	}
}

func TestFetchSchemaFromSource_MissingFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "mold.yaml"), []byte("name: demo\nversion: 0.0.1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	schema, defaults, err := mold.FetchSchemaFromSource(context.Background(), dir)
	if err != nil {
		t.Fatalf("expected nil error when schema/defaults absent, got %v", err)
	}
	if len(schema) != 0 {
		t.Fatalf("expected empty schema, got %d entries", len(schema))
	}
	if len(defaults) != 0 {
		t.Fatalf("expected empty defaults, got %+v", defaults)
	}
}

func TestFetchSchemaFromSource_EmptySource(t *testing.T) {
	_, _, err := mold.FetchSchemaFromSource(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty source")
	}
}

func TestFetchSchemaFromSource_FileNotDir(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "not-a-dir-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	_ = f.Close()

	_, _, err = mold.FetchSchemaFromSource(context.Background(), f.Name())
	if err == nil {
		t.Fatal("expected error when source is a file, not a dir")
	}
}

func TestFetchSchemaFromSource_RemoteResolverDispatchAndCleanup(t *testing.T) {
	prev := mold.ResolveSchemaFunc
	t.Cleanup(func() { mold.ResolveSchemaFunc = prev })

	cleanupCalls := 0
	mold.ResolveSchemaFunc = func(ctx context.Context, source string) (fs.FS, func(), error) {
		if source != "official/agents" {
			t.Fatalf("unexpected source: %q", source)
		}
		return fstest.MapFS{
			"flux.schema.yaml": &fstest.MapFile{Data: []byte("- name: k\n  type: string\n")},
			"flux.yaml":        &fstest.MapFile{Data: []byte("k: v\n")},
		}, func() { cleanupCalls++ }, nil
	}

	schema, defaults, err := mold.FetchSchemaFromSource(context.Background(), "official/agents")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(schema) != 1 || schema[0].Name != "k" {
		t.Fatalf("schema = %+v", schema)
	}
	if defaults["k"] != "v" {
		t.Fatalf("defaults = %+v", defaults)
	}
	if cleanupCalls != 1 {
		t.Fatalf("cleanup calls = %d want 1", cleanupCalls)
	}
}

func TestLoadMoldFluxWithOres_MergesAcrossSearchPaths(t *testing.T) {
	moldFS := fstest.MapFS{
		"flux.schema.yaml":             &fstest.MapFile{Data: []byte("- name: project.organization\n  type: string\n")},
		"flux.yaml":                    &fstest.MapFile{Data: []byte("project:\n  organization: nimble-giant\n")},
		"ores/status/ore.yaml":         &fstest.MapFile{Data: []byte("apiVersion: v1\nkind: ore\nname: status\nversion: 1.0.0\n")},
		"ores/status/flux.schema.yaml": &fstest.MapFile{Data: []byte("- name: enabled\n  type: bool\n")},
		"ores/status/flux.yaml":        &fstest.MapFile{Data: []byte("enabled: false\n")},
	}
	schema, defaults, report, err := mold.LoadMoldFluxWithOres(moldFS, []mold.OreSearchPath{{Name: "mold-local", FS: moldFS, Root: "ores"}})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	// Should have the mold's own entry plus ore.status.enabled.
	names := map[string]bool{}
	for _, e := range schema {
		names[e.Name] = true
	}
	if !names["project.organization"] || !names["ore.status.enabled"] {
		t.Errorf("missing expected entries: %+v", names)
	}
	// Defaults should have ore.status.enabled wired through.
	if oreNs, ok := defaults["ore"].(map[string]any); !ok || oreNs["status"] == nil {
		t.Errorf("ore defaults not merged: %+v", defaults)
	}
	if report.Sources["ore.status.enabled"] != "ore:status" {
		t.Errorf("Sources should attribute to ore:status, got %v", report.Sources)
	}
}

func TestLoadMoldFluxWithOres_MoldDefaultsWinOverOreDefaults(t *testing.T) {
	// The mold's own flux.yaml sets a value under ore.status.enabled.
	// The installed ore's flux.yaml sets a different value for the same key.
	// Mold-wins: the merged defaults should reflect the mold's value.
	moldFS := fstest.MapFS{
		"flux.schema.yaml": &fstest.MapFile{Data: []byte(`- name: ore.status.enabled
  type: bool
`)},
		"flux.yaml": &fstest.MapFile{Data: []byte(`ore:
  status:
    enabled: true
`)},
		"ores/status/ore.yaml":         &fstest.MapFile{Data: []byte("apiVersion: v1\nkind: ore\nname: status\nversion: 1.0.0\n")},
		"ores/status/flux.schema.yaml": &fstest.MapFile{Data: []byte("- name: enabled\n  type: bool\n")},
		"ores/status/flux.yaml":        &fstest.MapFile{Data: []byte("enabled: false\n")},
	}
	_, defaults, _, err := mold.LoadMoldFluxWithOres(moldFS, []mold.OreSearchPath{{Name: "mold-local", FS: moldFS, Root: "ores"}})
	if err != nil {
		t.Fatalf("LoadMoldFluxWithOres: %v", err)
	}
	oreNs, ok := defaults["ore"].(map[string]any)
	if !ok {
		t.Fatalf("missing ore namespace: %+v", defaults)
	}
	statusNs, ok := oreNs["status"].(map[string]any)
	if !ok {
		t.Fatalf("missing ore.status: %+v", oreNs)
	}
	if statusNs["enabled"] != true {
		t.Errorf("mold should win on ore.status.enabled; got %v (want true)", statusNs["enabled"])
	}
}

func TestFetchSchemaFromSource_RemoteResolverError(t *testing.T) {
	prev := mold.ResolveSchemaFunc
	t.Cleanup(func() { mold.ResolveSchemaFunc = prev })

	sentinel := errors.New("resolver boom")
	mold.ResolveSchemaFunc = func(ctx context.Context, source string) (fs.FS, func(), error) {
		return nil, nil, sentinel
	}

	_, _, err := mold.FetchSchemaFromSource(context.Background(), "remote/ref")
	if !errors.Is(err, sentinel) {
		t.Fatalf("err = %v want wrap of sentinel", err)
	}
}
