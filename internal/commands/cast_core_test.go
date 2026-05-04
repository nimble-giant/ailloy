package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nimble-giant/ailloy/pkg/blanks"
	"github.com/nimble-giant/ailloy/pkg/foundry"
	"github.com/nimble-giant/ailloy/pkg/mold"
)

// TestLayerFluxForCore_AutoLoadsPersistedFluxFiles asserts that
// ./.ailloy/flux/<slug>.yaml and ~/.ailloy/flux/<slug>.yaml are layered into
// the cast pipeline between mold defaults and explicit -f files. This is the
// fix for the bug where the foundries TUI's project/global save wrote files
// that nothing read.
func TestLayerFluxForCore_AutoLoadsPersistedFluxFiles(t *testing.T) {
	homeDir := t.TempDir()
	projectDir := t.TempDir()
	t.Chdir(projectDir)
	t.Setenv("HOME", homeDir)

	// Build a minimal mold dir on disk so MoldReader can load it.
	moldDir := filepath.Join(projectDir, "mold")
	if err := os.MkdirAll(moldDir, 0o750); err != nil {
		t.Fatal(err)
	}
	manifest := []byte(`name: launch
flux:
  - name: target
    type: string
    default: claude
`)
	if err := os.WriteFile(filepath.Join(moldDir, "mold.yaml"), manifest, 0o600); err != nil {
		t.Fatal(err)
	}

	reader, err := blanks.NewMoldReaderFromPath(moldDir)
	if err != nil {
		t.Fatalf("MoldReaderFromPath: %v", err)
	}

	source := "github.com/kris/replicated-foundry/molds/launch"
	slug := mold.FluxFileSlug(source)

	// Without any persisted file: target == default.
	flux, err := layerFluxForCore(reader, source, nil, nil)
	if err != nil {
		t.Fatalf("layerFluxForCore: %v", err)
	}
	if got := flux["target"]; got != "claude" {
		t.Fatalf("expected default target=claude; got %v", got)
	}

	// Write project-scoped persisted file with target: opencode.
	projectFlux := filepath.Join(".ailloy", "flux", slug+".yaml")
	if err := os.MkdirAll(filepath.Dir(projectFlux), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(projectFlux, []byte("target: opencode\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	flux, err = layerFluxForCore(reader, source, nil, nil)
	if err != nil {
		t.Fatalf("layerFluxForCore: %v", err)
	}
	if got := flux["target"]; got != "opencode" {
		t.Fatalf("expected project-saved target=opencode; got %v", got)
	}

	// Explicit --set still wins over persisted file (Helm-style precedence).
	flux, err = layerFluxForCore(reader, source, nil, []string{"target=zed"})
	if err != nil {
		t.Fatalf("layerFluxForCore: %v", err)
	}
	if got := flux["target"]; got != "zed" {
		t.Fatalf("expected --set to win over persisted; got %v", got)
	}

	// Empty source skips persisted-file lookup (local mold dirs).
	flux, err = layerFluxForCore(reader, "", nil, nil)
	if err != nil {
		t.Fatalf("layerFluxForCore: %v", err)
	}
	if got := flux["target"]; got != "claude" {
		t.Fatalf("expected default when source empty; got %v", got)
	}
}

// TestLayerFluxForCore_SubpathOverridesAreFound is the regression test for
// issue #196: when a foundry hosts multiple molds at distinct subpaths, the
// saved override file must be picked up during the cast. Previously the load
// side keyed lookups on Reference.CacheKey() (host/owner/repo only) while the
// fluxpicker saved keyed on the full mold source (with subpath), so the two
// disagreed and per-mold customization silently no-op'd.
func TestLayerFluxForCore_SubpathOverridesAreFound(t *testing.T) {
	homeDir := t.TempDir()
	projectDir := t.TempDir()
	t.Chdir(projectDir)
	t.Setenv("HOME", homeDir)

	moldDir := filepath.Join(projectDir, "mold")
	if err := os.MkdirAll(moldDir, 0o750); err != nil {
		t.Fatal(err)
	}
	manifest := []byte(`name: launch
flux:
  - name: target
    type: string
    default: claude
`)
	if err := os.WriteFile(filepath.Join(moldDir, "mold.yaml"), manifest, 0o600); err != nil {
		t.Fatal(err)
	}
	reader, err := blanks.NewMoldReaderFromPath(moldDir)
	if err != nil {
		t.Fatalf("MoldReaderFromPath: %v", err)
	}

	// Mirror the foundries TUI's save path: the picker slugs the raw mold
	// source string (no Reference parsing) and writes to .ailloy/flux/.
	rawSource := "github.com/replicated-collab/foundry//molds/launch"
	saveSlug := mold.FluxFileSlug(rawSource)
	projectFlux := filepath.Join(".ailloy", "flux", saveSlug+".yaml")
	if err := os.MkdirAll(filepath.Dir(projectFlux), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(projectFlux, []byte("target: opencode\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	// Mirror the cast pipeline: parse the same ref and pass OverrideKey() in
	// as `source`. The lookup must find the file that was saved above.
	ref, err := foundry.ParseReference(rawSource)
	if err != nil {
		t.Fatalf("ParseReference: %v", err)
	}
	flux, err := layerFluxForCore(reader, ref.OverrideKey(), nil, nil)
	if err != nil {
		t.Fatalf("layerFluxForCore: %v", err)
	}
	if got := flux["target"]; got != "opencode" {
		t.Fatalf("expected subpath override target=opencode; got %v (slug saved=%q lookup-key=%q)",
			got, saveSlug, ref.OverrideKey())
	}

	// CacheKey() — the old, buggy lookup — must NOT find the override.
	// Pinning this prevents a future refactor from silently regressing.
	flux, err = layerFluxForCore(reader, ref.CacheKey(), nil, nil)
	if err != nil {
		t.Fatalf("layerFluxForCore (cache key): %v", err)
	}
	if got := flux["target"]; got != "claude" {
		t.Fatalf("CacheKey() lookup unexpectedly matched; subpath molds in the same repo would collide. got=%v", got)
	}
}
