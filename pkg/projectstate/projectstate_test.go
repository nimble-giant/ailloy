package projectstate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// isolate points HOME at a fresh temp dir, clears the override env var, and
// chdirs into a fresh project dir. Returns (home, project).
func isolate(t *testing.T) (string, string) {
	t.Helper()
	home := t.TempDir()
	project := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv(EnvDir, "")
	t.Chdir(project)
	return home, project
}

func mustWrite(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func mustRead(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path) // #nosec G304 -- test path
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// Given a project, When the state dir is resolved, Then it lives under
// ~/.ailloy/projects/ and never inside the project tree.
func TestDir_LivesUnderHomeNotProject(t *testing.T) {
	home, project := isolate(t)
	got := Dir()
	wantPrefix := filepath.Join(home, ".ailloy", "projects") + string(filepath.Separator)
	if !strings.HasPrefix(got, wantPrefix) {
		t.Fatalf("Dir() = %q, want under %q", got, wantPrefix)
	}
	realProject, _ := filepath.EvalSymlinks(project)
	if strings.HasPrefix(got, realProject) || strings.HasPrefix(got, project) {
		t.Fatalf("Dir() = %q must not live inside the project %q", got, project)
	}
	if !strings.HasPrefix(filepath.Base(got), filepath.Base(realProject)+"-") {
		t.Errorf("Dir() basename %q should start with the project name %q for readability", filepath.Base(got), filepath.Base(realProject))
	}
}

// Given two different projects with the same basename, Then their state dirs
// differ; and the same project always maps to the same dir.
func TestDirFor_StableAndUnique(t *testing.T) {
	isolate(t)
	a := filepath.Join(t.TempDir(), "app")
	b := filepath.Join(t.TempDir(), "app")
	for _, d := range []string{a, b} {
		if err := os.MkdirAll(d, 0o750); err != nil {
			t.Fatal(err)
		}
	}
	if DirFor(a) == DirFor(b) {
		t.Fatalf("distinct projects share a state dir: %q", DirFor(a))
	}
	if first, again := DirFor(a), DirFor(a); first != again {
		t.Fatal("DirFor is not stable")
	}
	t.Chdir(a)
	if Dir() != DirFor(a) {
		t.Errorf("Dir() in %q = %q, DirFor = %q", a, Dir(), DirFor(a))
	}
}

// Given AILLOY_PROJECT_STATE_DIR is relative, Then it resolves against the
// project root (".ailloy" keeps state inside the project).
func TestDirFor_EnvOverride(t *testing.T) {
	isolate(t)
	t.Setenv(EnvDir, ".ailloy")
	if got := Dir(); got != ".ailloy" {
		t.Errorf("Dir() = %q, want .ailloy", got)
	}
	if got := DirFor("/some/mold"); got != filepath.Join("/some/mold", ".ailloy") {
		t.Errorf("DirFor = %q", got)
	}
	abs := filepath.Join(t.TempDir(), "state")
	t.Setenv(EnvDir, abs)
	if got := DirFor("/anything"); got != abs {
		t.Errorf("absolute override: got %q want %q", got, abs)
	}
}

// Given a project that still has legacy in-project bookkeeping, When
// MigrateLegacy runs, Then every known artifact moves into the state dir,
// unknown files are left alone, and the project tree is cleaned up.
func TestMigrateLegacy_MovesKnownArtifacts(t *testing.T) {
	_, project := isolate(t)
	mustWrite(t, filepath.Join(project, ".ailloy", "installed.yaml"), "molds: []\n")
	mustWrite(t, filepath.Join(project, ".ailloy", "state.yaml"), "blankDirs: []\n")
	mustWrite(t, filepath.Join(project, ".ailloy", "flux", "m.yaml"), "a: 1\n")
	mustWrite(t, filepath.Join(project, ".ailloy", "ores", "o", "ore.yaml"), "kind: ore\n")
	mustWrite(t, filepath.Join(project, ".ailloy", "ingots", "i", "ingot.yaml"), "name: i\n")
	mustWrite(t, filepath.Join(project, "ailloy.lock"), "apiVersion: v1\n")

	res, err := MigrateLegacy(".")
	if err != nil {
		t.Fatalf("MigrateLegacy: %v", err)
	}
	if len(res.Moved) != 6 {
		t.Errorf("moved %d items, want 6: %v", len(res.Moved), res.Moved)
	}
	dir := Dir()
	for rel, body := range map[string]string{
		"installed.yaml":      "molds: []\n",
		"state.yaml":          "blankDirs: []\n",
		"flux/m.yaml":         "a: 1\n",
		"ores/o/ore.yaml":     "kind: ore\n",
		"ingots/i/ingot.yaml": "name: i\n",
		LegacyLockFileName:    "apiVersion: v1\n",
	} {
		if got := mustRead(t, filepath.Join(dir, rel)); got != body {
			t.Errorf("%s = %q, want %q", rel, got, body)
		}
	}
	if exists(filepath.Join(project, ".ailloy")) {
		t.Error("empty legacy .ailloy/ dir should be removed")
	}
	if exists(filepath.Join(project, "ailloy.lock")) {
		t.Error("legacy ailloy.lock should be moved out of the project")
	}
	if root, ok := RootFor(dir); !ok || root == "" {
		t.Errorf("RootFor(state dir) = %q, %v; want the project root recorded", root, ok)
	}
}

// Given a legacy file AND an existing file at the destination, Then the
// legacy copy is left in place (never clobber) and reported as a conflict.
func TestMigrateLegacy_DoesNotClobber(t *testing.T) {
	_, project := isolate(t)
	mustWrite(t, filepath.Join(project, ".ailloy", "installed.yaml"), "legacy\n")
	mustWrite(t, filepath.Join(project, ".ailloy", "plugin.json"), "{}\n")
	mustWrite(t, filepath.Join(Dir(), "installed.yaml"), "current\n")

	res, err := MigrateLegacy(".")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Conflicts) != 1 {
		t.Errorf("conflicts = %v, want 1", res.Conflicts)
	}
	if got := mustRead(t, filepath.Join(Dir(), "installed.yaml")); got != "current\n" {
		t.Errorf("destination clobbered: %q", got)
	}
	if !exists(filepath.Join(project, ".ailloy", "installed.yaml")) {
		t.Error("conflicting legacy file should be left in place")
	}
	if !exists(filepath.Join(project, ".ailloy", "plugin.json")) {
		t.Error("unknown files in .ailloy/ must be left alone")
	}
}

// Given nothing legacy, Then MigrateLegacy is a no-op and creates nothing.
func TestMigrateLegacy_NoopWhenClean(t *testing.T) {
	isolate(t)
	res, err := MigrateLegacy(".")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Moved)+len(res.Conflicts) != 0 {
		t.Errorf("unexpected result %+v", res)
	}
	if exists(Dir()) {
		t.Error("no-op migration must not create the state dir")
	}
}

// Given the override points at the legacy dir, Then .ailloy/ entries stay put
// and only the root-level lock moves in beside them.
func TestMigrateLegacy_SkippedWhenOverrideIsLegacy(t *testing.T) {
	_, project := isolate(t)
	t.Setenv(EnvDir, ".ailloy")
	mustWrite(t, filepath.Join(project, ".ailloy", "installed.yaml"), "x\n")
	mustWrite(t, filepath.Join(project, "ailloy.lock"), "y\n")
	res, err := MigrateLegacy(".")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Moved) != 1 || res.Moved[0] != "ailloy.lock" {
		t.Errorf("moved %v; want only ailloy.lock", res.Moved)
	}
	if got := mustRead(t, filepath.Join(project, ".ailloy", "installed.yaml")); got != "x\n" {
		t.Errorf("installed.yaml disturbed: %q", got)
	}
	if got := mustRead(t, filepath.Join(project, ".ailloy", "ailloy.lock")); got != "y\n" {
		t.Errorf("lock not moved under .ailloy/: %q", got)
	}
}

// Given state recorded for several projects, Then List reports each with its
// root so users can discover which hidden dir belongs to which project.
func TestRecordAndList(t *testing.T) {
	isolate(t)
	a, b := t.TempDir(), t.TempDir()
	for _, p := range []string{a, b} {
		mustWrite(t, filepath.Join(DirFor(p), "installed.yaml"), "molds: []\n")
		if err := Record(p); err != nil {
			t.Fatal(err)
		}
	}
	// Record on a project without state is a no-op.
	c := t.TempDir()
	if err := Record(c); err != nil {
		t.Fatal(err)
	}
	if exists(DirFor(c)) {
		t.Error("Record must not create a state dir for a project without state")
	}
	got, err := List()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("List() = %+v, want 2 projects", got)
	}
	roots := map[string]bool{}
	for _, p := range got {
		roots[p.Root] = true
		if p.Dir != DirFor(p.Root) {
			t.Errorf("project %q dir %q != DirFor %q", p.Root, p.Dir, DirFor(p.Root))
		}
	}
	for _, p := range []string{a, b} {
		if !roots[canonical(p)] {
			t.Errorf("List missing %q: %v", p, roots)
		}
	}
}

// Given ailloy runs from the home directory, Then ~/.ailloy is the GLOBAL
// state (global manifest, ores, ingots, ~/ailloy.lock) and must never be
// migrated into a per-project dir.
func TestMigrateLegacy_NeverTouchesHome(t *testing.T) {
	home, _ := isolate(t)
	t.Chdir(home)
	mustWrite(t, filepath.Join(home, ".ailloy", "installed.yaml"), "global\n")
	mustWrite(t, filepath.Join(home, "ailloy.lock"), "global\n")
	res, err := MigrateLegacy(".")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Moved) != 0 {
		t.Fatalf("migrated global state: %v", res.Moved)
	}
	if !exists(filepath.Join(home, ".ailloy", "installed.yaml")) || !exists(filepath.Join(home, "ailloy.lock")) {
		t.Fatal("global state was moved")
	}
}
