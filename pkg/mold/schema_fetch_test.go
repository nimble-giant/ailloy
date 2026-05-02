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
	f.Close()

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
