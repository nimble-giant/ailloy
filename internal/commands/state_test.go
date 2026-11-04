package commands

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nimble-giant/ailloy/pkg/projectstate"
)

func stateTestProject(t *testing.T) string {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv(projectstate.EnvDir, "")
	dir := t.TempDir()
	t.Chdir(dir)
	return dir
}

// Given a project, When `ailloy state path` runs, Then it prints the hidden
// state dir so users can find the bookkeeping.
func TestStatePath_PrintsProjectStateDir(t *testing.T) {
	stateTestProject(t)
	var out bytes.Buffer
	if err := runStatePath(&out); err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(out.String()); got != projectStateDir() {
		t.Errorf("state path = %q, want %q", got, projectStateDir())
	}
}

// Given a project with legacy in-tree bookkeeping, When any ailloy command
// starts, Then the files move to the state dir with a one-line notice.
func TestMigrateProjectState_MovesLegacyAndNotifies(t *testing.T) {
	dir := stateTestProject(t)
	legacy := filepath.Join(dir, ".ailloy", "installed.yaml")
	if err := os.MkdirAll(filepath.Dir(legacy), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(legacy, []byte("apiVersion: v1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "ailloy.lock"), []byte("apiVersion: v1\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var errOut bytes.Buffer
	migrateProjectState(&errOut)

	if _, err := os.Stat(projectManifestPath()); err != nil {
		t.Errorf("manifest not migrated: %v", err)
	}
	if _, err := os.Stat(projectLockPath()); err != nil {
		t.Errorf("lock not migrated: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".ailloy")); !os.IsNotExist(err) {
		t.Errorf("legacy .ailloy/ should be gone, stat err = %v", err)
	}
	if !strings.Contains(errOut.String(), projectStateDir()) {
		t.Errorf("notice should name the new location; got %q", errOut.String())
	}

	// Second run is silent.
	errOut.Reset()
	migrateProjectState(&errOut)
	if errOut.Len() != 0 {
		t.Errorf("second migration should be silent; got %q", errOut.String())
	}
}

// Given state for this project, When `ailloy state list` runs, Then it shows
// the project root next to its state dir.
func TestStateList_ShowsRecordedProjects(t *testing.T) {
	dir := stateTestProject(t)
	if err := os.MkdirAll(projectStateDir(), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := projectstate.Record("."); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := runStateList(&out); err != nil {
		t.Fatal(err)
	}
	real, _ := filepath.EvalSymlinks(dir)
	if !strings.Contains(out.String(), real) || !strings.Contains(out.String(), projectStateDir()) {
		t.Errorf("state list missing project %q / dir %q:\n%s", real, projectStateDir(), out.String())
	}
}

func TestSkipsStateHooks(t *testing.T) {
	for name, want := range map[string]bool{"help": true, "completion": true, "__complete": true, "cast": false} {
		if got := skipsStateHooks(name); got != want {
			t.Errorf("skipsStateHooks(%q) = %v, want %v", name, got, want)
		}
	}
}
