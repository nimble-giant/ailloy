package fluxpicker

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	yaml "github.com/goccy/go-yaml"

	"github.com/nimble-giant/ailloy/pkg/mold"
)

func TestWriteFluxFile_CreatesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "flux.yaml")
	overrides := map[string]any{"agents.targets": []string{"opencode"}}
	if err := writeFluxFile(path, overrides); err != nil {
		t.Fatalf("writeFluxFile: %v", err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	var got map[string]any
	if err := yaml.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	agents, _ := got["agents"].(map[string]any)
	if agents == nil {
		t.Fatalf("expected nested agents map, got %+v", got)
	}
	targets, _ := agents["targets"].([]any)
	if len(targets) != 1 || targets[0] != "opencode" {
		t.Fatalf("targets = %v want [opencode]", targets)
	}
}

func TestWriteFluxFile_MergesExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "flux.yaml")
	if err := os.WriteFile(path, []byte("agents:\n  parallel: true\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	overrides := map[string]any{"agents.targets": []string{"opencode"}}
	if err := writeFluxFile(path, overrides); err != nil {
		t.Fatalf("writeFluxFile: %v", err)
	}
	b, _ := os.ReadFile(path)
	var got map[string]any
	_ = yaml.Unmarshal(b, &got)
	agents := got["agents"].(map[string]any)
	if agents["parallel"] != true {
		t.Fatalf("expected existing parallel:true preserved, got %+v", agents)
	}
	if _, ok := agents["targets"]; !ok {
		t.Fatalf("expected targets to be merged in, got %+v", agents)
	}
}

func TestPersistOverrides_Session(t *testing.T) {
	path, err := persistOverrides("agents", SaveTargetSession, map[string]any{"k": "v"})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if path != "" {
		t.Fatalf("expected empty path for session target, got %q", path)
	}
}

func TestPersistOverrides_ProjectWritesFile(t *testing.T) {
	t.Chdir(t.TempDir())
	path, err := persistOverrides("agents", SaveTargetProject, map[string]any{"agents.targets": []string{"opencode"}})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if path != ".ailloy/flux/agents.yaml" {
		t.Fatalf("path = %q want .ailloy/flux/agents.yaml", path)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected file at %s, got %v", path, err)
	}
}

func TestPersistOverrides_GlobalWritesFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	path, err := persistOverrides("agents", SaveTargetGlobal, map[string]any{"k": "v"})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	want := filepath.Join(home, ".ailloy", "flux", "agents.yaml")
	if path != want {
		t.Fatalf("path = %q want %q", path, want)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected file at %s, got %v", path, err)
	}
}

func TestFluxFileSlug(t *testing.T) {
	cases := map[string]string{
		"":                                   "mold",
		"agents":                             "agents",
		"official/agents":                    "official_agents",
		"github.com/nimble-giant/agents":     "github.com_nimble-giant_agents",
		"github.com/nimble-giant/agents@v1":  "github.com_nimble-giant_agents_v1",
		"github.com/nimble-giant/agents.git": "github.com_nimble-giant_agents",
		"github.com/foo/bar//sub/path":       "github.com_foo_bar_sub_path",
		"trailing/":                          "trailing",
		"  spaces around  ":                  "spaces_around",
	}
	for in, want := range cases {
		if got := fluxFileSlug(in); got != want {
			t.Errorf("fluxFileSlug(%q) = %q want %q", in, got, want)
		}
	}
}

// Sibling foundries that re-export a same-named mold must produce distinct
// slugs so neither user's saved overrides clobber the other.
func TestFluxFileSlug_AvoidsCrossFoundryCollision(t *testing.T) {
	a := fluxFileSlug("github.com/nimble-giant/agents")
	b := fluxFileSlug("github.com/replicated/agents")
	if a == b {
		t.Fatalf("expected distinct slugs across foundries; both = %q", a)
	}
}

func TestPersistFoundryOverridesFansOutToProject(t *testing.T) {
	tmp := t.TempDir()
	t.Chdir(tmp)

	per := map[string][]mold.FluxVar{
		"alpha": {{Name: "agents.targets", Type: "list"}},
		"beta":  {{Name: "agents.targets", Type: "list"}, {Name: "theme", Type: "string"}},
		"gamma": {{Name: "theme", Type: "string"}},
	}
	refs := map[string]string{
		"alpha": "github.com/example/alpha",
		"beta":  "github.com/example/beta",
		"gamma": "github.com/example/gamma",
	}
	unified := map[string]any{
		"agents.targets": []any{"claude", "opencode"},
	}
	perMold := map[string]map[string]any{
		"beta":  {"theme": "noon"},
		"gamma": {"theme": "midnight"},
	}

	written, err := persistFoundryOverrides(SaveTargetProject, per, refs, unified, perMold)
	if err != nil {
		t.Fatalf("persistFoundryOverrides: %v", err)
	}

	// alpha and beta have agents.targets, so they get the unified value.
	// beta also gets its per-mold theme. gamma gets only per-mold theme.
	for _, name := range []string{"alpha", "beta", "gamma"} {
		slug := mold.FluxFileSlug(refs[name])
		path := filepath.Join(".ailloy", "flux", slug+".yaml")
		if _, err := os.Stat(path); err != nil {
			t.Errorf("expected %s to exist (%s flux file): %v", path, name, err)
		}
		if !contains(written, path) {
			t.Errorf("written paths missing %s: %v", path, written)
		}
	}

	// Spot-check beta's content.
	betaSlug := mold.FluxFileSlug(refs["beta"])
	b, _ := os.ReadFile(filepath.Join(".ailloy", "flux", betaSlug+".yaml"))
	if !strings.Contains(string(b), "claude") || !strings.Contains(string(b), "noon") {
		t.Errorf("beta flux file missing expected content:\n%s", b)
	}
}

func TestPersistFoundryOverridesSessionIsNoOp(t *testing.T) {
	tmp := t.TempDir()
	t.Chdir(tmp)
	written, err := persistFoundryOverrides(
		SaveTargetSession,
		map[string][]mold.FluxVar{"alpha": {{Name: "k", Type: "string"}}},
		map[string]string{"alpha": "ref"},
		map[string]any{"k": "v"},
		nil,
	)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if len(written) != 0 {
		t.Errorf("session save wrote files: %v", written)
	}
}

func contains(ss []string, s string) bool {
	for _, x := range ss {
		if x == s {
			return true
		}
	}
	return false
}
