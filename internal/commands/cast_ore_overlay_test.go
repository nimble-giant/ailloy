package commands

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/nimble-giant/ailloy/pkg/mold"
)

// TestAgentTargetsOreOverlay_EndToEndRendering verifies the issue #215
// acceptance criterion: a consumer mold whose only content is a one-line
// `dependencies: ore: agent_targets` declaration gets the full multi-target
// rendering produced by the ore's `output:` mappings + `blanks/` directory.
//
// This test exercises the resolution + render layer directly (mold.Temper
// for validation, EphemeralOreResolver for source extraction,
// ResolveFilesWithOreSources + ProcessTemplate for rendering). It does NOT
// go through the cast command — see cast_ore_e2e_test.go for why the full
// CastMold path is currently skipped under -race.
func TestAgentTargetsOreOverlay_EndToEndRendering(t *testing.T) {
	repoRoot := findRepoRoot(t)
	oreDir := filepath.Join(repoRoot, "testdata", "agent_targets", "ore")
	consumerDir := filepath.Join(repoRoot, "testdata", "agent_targets", "consumer")

	// 1. Temper validates: the ore + consumer testdata pass mold.Temper.
	if res := mold.Temper(os.DirFS(oreDir)); res.HasErrors() {
		t.Fatalf("ore testdata fails temper: %+v", res.Errors())
	}
	if res := mold.Temper(os.DirFS(consumerDir)); res.HasErrors() {
		t.Fatalf("consumer testdata fails temper: %+v", res.Errors())
	}

	// 2. Build the consumer manifest + its dependency-resolved OreSources.
	// We load the static mold.yaml for structural inspection, then construct
	// the Mold value with the ore dep's absolute path so ResolveDepsEphemeral
	// can stat it regardless of the test binary's working directory.
	manifest, err := mold.LoadMold(filepath.Join(consumerDir, "mold.yaml"))
	if err != nil {
		t.Fatalf("load consumer mold: %v", err)
	}
	// Patch the ore dep path to an absolute path so the ephemeral resolver can
	// stat it irrespective of the test binary's working directory.
	for i := range manifest.Dependencies {
		if manifest.Dependencies[i].Ore != "" {
			manifest.Dependencies[i].Ore = oreDir
		}
	}
	resolver, err := ResolveDepsEphemeral(manifest, true)
	if err != nil {
		t.Fatalf("resolve deps: %v", err)
	}
	sources := resolver.OreSources()
	if len(sources) != 1 {
		t.Fatalf("expected 1 ore source, got %d", len(sources))
	}

	// 3. Resolve outputs via the composite-FS resolver.
	consumerFS := os.DirFS(consumerDir)
	resolved, err := mold.ResolveFilesWithOreSources(nil, consumerFS, sources)
	if err != nil {
		t.Fatalf("ResolveFilesWithOreSources: %v", err)
	}

	gotDests := destPaths(resolved)
	wantDests := []string{
		".claude/agents/example.md",
		".opencode/agents/example.md",
		"AGENTS.md",
	}
	if !sliceEqual(gotDests, wantDests) {
		t.Fatalf("dest paths mismatch:\n  got:  %v\n  want: %v", gotDests, wantDests)
	}

	// 4. Each resolved file must carry the ore namespace + a non-nil SrcFS
	// (every output entry in this fixture comes from the ore overlay).
	for _, rf := range resolved {
		if rf.Origin != "agent_targets" {
			t.Errorf("%s: Origin = %q, want agent_targets", rf.DestPath, rf.Origin)
		}
		if rf.SrcFS == nil {
			t.Errorf("%s: SrcFS is nil", rf.DestPath)
		}
	}

	// 5. Render and verify per-target context flows through `set:` correctly.
	rendered := renderResolvedFiles(t, resolved)
	claude := rendered[".claude/agents/example.md"]
	opencode := rendered[".opencode/agents/example.md"]
	if !strings.Contains(claude, "target: claude") {
		t.Errorf(".claude/agents/example.md missing claude target token; got:\n%s", claude)
	}
	if !strings.Contains(opencode, "target: opencode") {
		t.Errorf(".opencode/agents/example.md missing opencode target token; got:\n%s", opencode)
	}
	// Sanity: the two renders are NOT identical (different set: context).
	if claude == opencode {
		t.Errorf("per-target renders should differ but were identical:\n%s", claude)
	}
}

// findRepoRoot walks up from the test's working directory until it finds a
// go.mod, returning that directory.
func findRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find repo root (go.mod) from " + dir)
		}
		dir = parent
	}
}

func destPaths(resolved []mold.ResolvedFile) []string {
	out := make([]string, 0, len(resolved))
	for _, rf := range resolved {
		out = append(out, rf.DestPath)
	}
	sort.Strings(out)
	return out
}

func sliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func renderResolvedFiles(t *testing.T, resolved []mold.ResolvedFile) map[string]string {
	t.Helper()
	out := map[string]string{}
	for _, rf := range resolved {
		fsys := rf.SrcFS
		content, err := fs.ReadFile(fsys, rf.SrcPath)
		if err != nil {
			t.Fatalf("read %s from origin %q: %v", rf.SrcPath, rf.Origin, err)
		}
		// Build a minimal flux context plus the per-destination set overlay.
		flux := map[string]any{}
		if len(rf.Set) > 0 {
			flux = mold.MergeSet(flux, rf.Set)
		}
		rendered, err := mold.ProcessTemplate(string(content), flux)
		if err != nil {
			t.Fatalf("render %s: %v", rf.SrcPath, err)
		}
		out[rf.DestPath] = rendered
	}
	return out
}
