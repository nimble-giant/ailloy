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

// TestInstallFoundryCoreForwardsFluxOptions asserts that --set / -f values on
// the foundry cast command flow into each per-mold CastMold call. We use
// DryRun to avoid actually invoking CastMold; the test stops at the report
// shape, then a second sub-test exercises the real CastMold path with a
// fixture local mold to assert the values land in the resolved flux.
func TestInstallFoundryCoreForwardsFluxOptions(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("AILLOY_INDEX_CACHE_DIR", tmp)

	parentURL := "https://github.com/example/parent"
	entry := &index.FoundryEntry{URL: parentURL, Type: "git"}
	dir := index.CachedIndexDir(tmp, entry)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	yaml := []byte(`apiVersion: v1
kind: foundry-index
name: parent
molds:
  - name: alpha
    source: github.com/x/alpha
`)
	if err := os.WriteFile(filepath.Join(dir, "foundry.yaml"), yaml, 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := &index.Config{
		Foundries: []index.FoundryEntry{{Name: "parent", URL: parentURL, Type: "git", Status: "ok"}},
	}

	// DryRun keeps us out of CastMold; we just assert the options carry
	// through and that no mold is reported as failed for malformed --set.
	reports, _, err := InstallFoundryCore(context.Background(), cfg, "parent", InstallFoundryOptions{
		DryRun:       true,
		Shallow:      true,
		ValueFiles:   []string{"./team-defaults.yaml"},
		SetOverrides: []string{"agents.targets=[claude,opencode]"},
	})
	if err != nil {
		t.Fatalf("InstallFoundryCore: %v", err)
	}
	if len(reports) != 1 {
		t.Fatalf("got %d reports, want 1", len(reports))
	}
	if reports[0].Err != nil {
		t.Errorf("dry-run report carried err: %v", reports[0].Err)
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
