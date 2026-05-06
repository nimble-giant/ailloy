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

// TestInstallFoundryCoreForwardsFluxOptions is a compile-time guard for the
// new ValueFiles / SetOverrides fields on InstallFoundryOptions. It uses
// DryRun to short-circuit before CastMold runs (no real cast happens).
// Behavioral coverage of the actual forwarding lands in Task 3, which
// asserts per-mold flux apply/skip results from the same call path.
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

	// DryRun short-circuits before CastMold; we just verify the options
	// struct accepts the new fields and the run completes cleanly.
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

// TestInstallFoundryCoreFluxApplyResults verifies that for each mold we
// record which user-supplied flux keys are present in that mold's schema
// (Applied) vs. absent (Skipped). DryRun keeps us out of CastMold but the
// schema lookup still runs.
func TestInstallFoundryCoreFluxApplyResults(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("AILLOY_INDEX_CACHE_DIR", tmp)

	// Build a real local mold for "alpha" with an agents.targets schema, and a
	// second mold "beta" without it. Source paths in the foundry index point
	// at on-disk paths so FetchSchemaFromSource can find them — DryRun skips
	// the actual cast.
	moldDir := func(name string, schemaYAML string) string {
		d := filepath.Join(tmp, name)
		if err := os.MkdirAll(d, 0o750); err != nil {
			t.Fatal(err)
		}
		manifest := []byte("apiVersion: v1\nkind: mold\nname: " + name + "\nversion: 0.0.1\n")
		if err := os.WriteFile(filepath.Join(d, "mold.yaml"), manifest, 0o644); err != nil {
			t.Fatal(err)
		}
		if schemaYAML != "" {
			if err := os.WriteFile(filepath.Join(d, "flux.schema.yaml"), []byte(schemaYAML), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		return d
	}
	alphaPath := moldDir("alpha", `- name: agents.targets
  type: list
  default: "[claude]"
`)
	betaPath := moldDir("beta", "")

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
    source: ` + alphaPath + `
  - name: beta
    source: ` + betaPath + `
`)
	if err := os.WriteFile(filepath.Join(dir, "foundry.yaml"), yaml, 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := &index.Config{
		Foundries: []index.FoundryEntry{{Name: "parent", URL: parentURL, Type: "git", Status: "ok"}},
	}

	reports, _, err := InstallFoundryCore(context.Background(), cfg, "parent", InstallFoundryOptions{
		DryRun:       true,
		Shallow:      true,
		SetOverrides: []string{"agents.targets=[claude,opencode]"},
	})
	if err != nil {
		t.Fatalf("InstallFoundryCore: %v", err)
	}
	if len(reports) != 2 {
		t.Fatalf("got %d reports, want 2", len(reports))
	}

	byName := map[string]InstallFoundryReport{}
	for _, r := range reports {
		byName[r.Name] = r
	}

	if got := byName["alpha"].FluxApplied; !reflect.DeepEqual(got, []string{"agents.targets"}) {
		t.Errorf("alpha.FluxApplied = %v, want [agents.targets]", got)
	}
	if got := byName["alpha"].FluxSkipped; len(got) != 0 {
		t.Errorf("alpha.FluxSkipped = %v, want empty", got)
	}
	if got := byName["beta"].FluxApplied; len(got) != 0 {
		t.Errorf("beta.FluxApplied = %v, want empty", got)
	}
	if got := byName["beta"].FluxSkipped; !reflect.DeepEqual(got, []string{"agents.targets"}) {
		t.Errorf("beta.FluxSkipped = %v, want [agents.targets]", got)
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
