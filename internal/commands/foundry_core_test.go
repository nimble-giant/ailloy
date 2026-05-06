package commands

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/nimble-giant/ailloy/pkg/foundry/index"
)

// TestInstallFoundryCoreShallow verifies --shallow restricts to root molds.
// It uses --dry-run to avoid invoking CastMold, and seeds a temp cache with
// a parent foundry that references a (non-existent) child URL — the child
// fetch fails and the resolver downgrades it to a warning, so the parent
// still resolves successfully and we only see the parent's own molds.
func TestInstallFoundryCoreShallow(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("AILLOY_INDEX_CACHE_DIR", tmp)

	parentURL := "https://github.com/example/parent"
	parentEntry := &index.FoundryEntry{URL: parentURL, Type: "git"}
	parentDir := index.CachedIndexDir(tmp, parentEntry)
	if err := os.MkdirAll(parentDir, 0o750); err != nil {
		t.Fatal(err)
	}
	parentYAML := []byte(`apiVersion: v1
kind: foundry-index
name: parent
molds:
  - name: alpha
    source: github.com/x/alpha
foundries:
  - name: missing
    source: github.com/example/missing
`)
	if err := os.WriteFile(filepath.Join(parentDir, "foundry.yaml"), parentYAML, 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := &index.Config{
		Foundries: []index.FoundryEntry{{Name: "parent", URL: parentURL, Type: "git", Status: "ok"}},
	}

	reports, warnings, err := InstallFoundryCore(context.Background(), cfg, "parent", InstallFoundryOptions{
		DryRun:  true,
		Shallow: true,
	})
	if err != nil {
		t.Fatalf("InstallFoundryCore: %v", err)
	}
	if len(reports) != 1 {
		t.Fatalf("got %d reports, want 1", len(reports))
	}
	if reports[0].Name != "alpha" {
		t.Errorf("reports[0].Name = %q, want alpha", reports[0].Name)
	}
	if reports[0].Foundry != "parent" {
		t.Errorf("reports[0].Foundry = %q, want parent", reports[0].Foundry)
	}
	if len(reports[0].Chain) != 0 {
		t.Errorf("reports[0].Chain = %v, want empty (root-owned)", reports[0].Chain)
	}
	// The unreachable child foundry should surface as a warning so the CLI
	// can explain why nested molds are missing.
	if len(warnings) != 1 {
		t.Fatalf("got %d warnings, want 1", len(warnings))
	}
	if warnings[0].Source != "github.com/example/missing" {
		t.Errorf("warnings[0].Source = %q, want github.com/example/missing", warnings[0].Source)
	}
}

func TestFoundryCastCommandShape(t *testing.T) {
	if foundryCastCmd.Use == "" || !strings.HasPrefix(foundryCastCmd.Use, "cast") {
		t.Fatalf("foundryCastCmd.Use = %q, want it to start with %q", foundryCastCmd.Use, "cast")
	}
	gotAliases := foundryCastCmd.Aliases
	wantAliases := []string{"install"}
	if !reflect.DeepEqual(gotAliases, wantAliases) {
		t.Fatalf("foundryCastCmd.Aliases = %v, want %v", gotAliases, wantAliases)
	}
}
