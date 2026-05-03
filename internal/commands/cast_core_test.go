package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nimble-giant/ailloy/pkg/blanks"
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
