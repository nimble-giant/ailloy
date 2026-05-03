package mold

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFluxFileSlug(t *testing.T) {
	cases := map[string]string{
		"":                                  "mold",
		"agents":                            "agents",
		"nimble-giant/agents":               "nimble-giant_agents",
		"github.com/nimble-giant/agents":    "github.com_nimble-giant_agents",
		"github.com/nimble-giant/agents@v1": "github.com_nimble-giant_agents_v1",
		"github.com/kris/replicated-foundry/molds/launch":     "github.com_kris_replicated-foundry_molds_launch",
		"github.com/kris/replicated-foundry/molds/launch.git": "github.com_kris_replicated-foundry_molds_launch",
		"  github.com/kris/replicated-foundry/molds/launch  ": "github.com_kris_replicated-foundry_molds_launch",
	}
	for in, want := range cases {
		if got := FluxFileSlug(in); got != want {
			t.Errorf("FluxFileSlug(%q) = %q want %q", in, got, want)
		}
	}
}

func TestPersistedFluxPaths_EmptyRefReturnsNil(t *testing.T) {
	if got := PersistedFluxPaths(""); got != nil {
		t.Fatalf("expected nil for empty ref; got %v", got)
	}
	if got := PersistedFluxPaths("   "); got != nil {
		t.Fatalf("expected nil for whitespace ref; got %v", got)
	}
}

func TestPersistedFluxPaths_OrderingAndExistence(t *testing.T) {
	// Use a distinct project dir and HOME so project's relative path can't
	// accidentally resolve onto the global file.
	homeDir := t.TempDir()
	projectDir := t.TempDir()
	t.Chdir(projectDir)
	t.Setenv("HOME", homeDir)

	ref := "github.com/x/y"
	slug := FluxFileSlug(ref)

	if got := PersistedFluxPaths(ref); len(got) != 0 {
		t.Fatalf("expected no paths before files exist; got %v", got)
	}

	globalPath := filepath.Join(homeDir, ".ailloy", "flux", slug+".yaml")
	if err := os.MkdirAll(filepath.Dir(globalPath), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(globalPath, []byte("k: g\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	got := PersistedFluxPaths(ref)
	if len(got) != 1 || got[0] != globalPath {
		t.Fatalf("expected [%s]; got %v", globalPath, got)
	}

	projectPath := filepath.Join(".ailloy", "flux", slug+".yaml")
	if err := os.MkdirAll(filepath.Dir(projectPath), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(projectPath, []byte("k: p\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	// Both exist — global first, then project (project wins on merge).
	got = PersistedFluxPaths(ref)
	if len(got) != 2 || got[0] != globalPath || got[1] != projectPath {
		t.Fatalf("expected [global project]; got %v", got)
	}
}
