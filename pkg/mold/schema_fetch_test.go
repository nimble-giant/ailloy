package mold_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

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
