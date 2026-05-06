package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nimble-giant/ailloy/pkg/blanks"
	"github.com/nimble-giant/ailloy/pkg/foundry"
	"github.com/nimble-giant/ailloy/pkg/mold"
)

// TestCastMold_CleansEmptyMultiDestDirs reproduces issue #195: when a
// multi-destination output mapping has per-dest `set` context and the
// template renders to empty for one destination, the foundry-TUI install
// path (CastMold) eagerly created the destination dir but never cleaned
// it up after the empty render was skipped. The CLI cast path already
// cleans up via cleanupEmptyDirs (#145); CastMold needs the same.
func TestCastMold_CleansEmptyMultiDestDirs(t *testing.T) {
	projectDir := t.TempDir()
	t.Chdir(projectDir)
	t.Setenv("HOME", t.TempDir())

	moldDir := filepath.Join(projectDir, "mold")
	if err := os.MkdirAll(filepath.Join(moldDir, "agents"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(moldDir, "mold.yaml"),
		[]byte("apiVersion: v1\nkind: Mold\nname: launch\nversion: 0.1.0\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(moldDir, "flux.yaml"), []byte(`agent:
  targets:
    - opencode

output:
  agents:
    - dest: .claude/agents
      set:
        agent.current_target: claude
    - dest: .opencode/agents
      set:
        agent.current_target: opencode
`), 0o600); err != nil {
		t.Fatal(err)
	}
	template := `{{- if and (eq .agent.current_target "claude") (has "claude" .agent.targets) -}}
---
name: coding-agent
---
claude body
{{- else if and (eq .agent.current_target "opencode") (has "opencode" .agent.targets) -}}
---
description: opencode
---
opencode body
{{- end -}}`
	if err := os.WriteFile(filepath.Join(moldDir, "agents", "coding-agent.md"), []byte(template), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := CastMold(t.Context(), moldDir, CastOptions{}); err != nil {
		t.Fatalf("CastMold: %v", err)
	}

	if _, err := os.Stat(filepath.Join(projectDir, ".opencode", "agents", "coding-agent.md")); err != nil {
		t.Fatalf("expected .opencode/agents/coding-agent.md to exist: %v", err)
	}
	if _, err := os.Stat(filepath.Join(projectDir, ".claude", "agents")); err == nil {
		t.Errorf("regression #195: .claude/agents/ leftover after CastMold (template rendered empty for inactive target)")
	}
}

// TestRecordCastedFiles_SubpathMold reproduces the bug where monorepo-foundry
// molds (those with a non-empty Subpath) installed via the foundries TUI's
// batch install end up with `Files == nil` in the installed manifest, which
// makes the TUI label them "legacy" and `UninstallMold` refuse to remove
// them.
//
// Cause: cast.go and cast_core.go each open-coded the manifest write
// (recordInstalled, keyed by Ref.CacheKey()) and the Files backfill
// (RecordInstalledFiles, called with Ref.OverrideKey() as the lookup source).
// For subpath molds CacheKey != OverrideKey, FindBySource returned nil, and
// the error was silently swallowed (cast_core.go) or only logged into a
// discard sink (CastMold runs with log output redirected). Net effect:
// entry.Files stays nil.
//
// recordCastedFiles centralizes both writes so the source key cannot drift
// between them. This test pins the contract.
func TestRecordCastedFiles_SubpathMold(t *testing.T) {
	projectDir := t.TempDir()
	t.Chdir(projectDir)
	t.Setenv("HOME", t.TempDir())

	result := &foundry.ResolveResult{
		Ref: &foundry.Reference{
			Host:    "github.com",
			Owner:   "replicated-collab",
			Repo:    "foundry",
			Subpath: "molds/shortcut",
			Version: "v0.1.0",
		},
		Resolved: foundry.ResolvedVersion{Tag: "v0.1.0", Commit: "abc1234"},
	}
	files := []foundry.InstalledFile{
		{RelPath: ".claude/agents/shortcut.md", SHA256: "deadbeef"},
	}

	if err := recordCastedFiles(result, files, false); err != nil {
		t.Fatalf("recordCastedFiles: %v", err)
	}

	m, err := foundry.ReadInstalledManifest(manifestPathFor(false))
	if err != nil {
		t.Fatalf("ReadInstalledManifest: %v", err)
	}
	entry := m.FindBySource(result.Ref.CacheKey(), result.Ref.Subpath)
	if entry == nil {
		t.Fatalf("entry not found in manifest at (CacheKey=%q, Subpath=%q)", result.Ref.CacheKey(), result.Ref.Subpath)
	}
	if entry.Files == nil {
		t.Fatalf("entry.Files is nil — Files backfill silently dropped (legacy bug); manifest=%+v", entry)
	}
	if len(entry.Files) != 1 || entry.Files[0] != ".claude/agents/shortcut.md" {
		t.Fatalf("unexpected Files: %+v", entry.Files)
	}
}

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
