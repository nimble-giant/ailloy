package commands

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/goccy/go-yaml"
)

// Regression: writeInstallState must merge with the existing state.yaml
// instead of overwriting. The previous behavior wiped earlier mold install
// dirs every time a new mold was cast.
func TestWriteInstallState_MergesWithExisting(t *testing.T) {
	t.Chdir(t.TempDir())

	if err := writeInstallState([]string{".claude/skills/shortcut-cli", ".github/workflows"}); err != nil {
		t.Fatalf("first writeInstallState: %v", err)
	}
	if err := writeInstallState([]string{".claude/skills/linear-cli"}); err != nil {
		t.Fatalf("second writeInstallState: %v", err)
	}

	state, err := loadInstallStateForTest(installStatePath)
	if err != nil {
		t.Fatalf("read state.yaml: %v", err)
	}
	wantBlank := []string{".claude/skills/linear-cli", ".claude/skills/shortcut-cli"}
	if !reflect.DeepEqual(state.BlankDirs, wantBlank) {
		t.Errorf("BlankDirs = %v, want %v", state.BlankDirs, wantBlank)
	}
	wantWorkflows := []string{".github/workflows"}
	if !reflect.DeepEqual(state.WorkflowDirs, wantWorkflows) {
		t.Errorf("WorkflowDirs = %v, want %v", state.WorkflowDirs, wantWorkflows)
	}
}

func TestWriteInstallState_DedupesRepeatedDirs(t *testing.T) {
	t.Chdir(t.TempDir())

	if err := writeInstallState([]string{".claude/skills/foo"}); err != nil {
		t.Fatal(err)
	}
	if err := writeInstallState([]string{".claude/skills/foo"}); err != nil {
		t.Fatal(err)
	}

	state, err := loadInstallStateForTest(installStatePath)
	if err != nil {
		t.Fatalf("read state.yaml: %v", err)
	}
	if len(state.BlankDirs) != 1 || state.BlankDirs[0] != ".claude/skills/foo" {
		t.Errorf("expected 1 deduped dir, got %v", state.BlankDirs)
	}
}

func TestWriteInstallState_FreshDirCreatesFile(t *testing.T) {
	t.Chdir(t.TempDir())

	if err := writeInstallState([]string{".claude/skills/x"}); err != nil {
		t.Fatalf("writeInstallState: %v", err)
	}
	if _, err := os.Stat(filepath.Join(".ailloy", "state.yaml")); err != nil {
		t.Errorf("state.yaml not created: %v", err)
	}
}

func loadInstallStateForTest(path string) (*installState, error) {
	data, err := os.ReadFile(path) // #nosec G304 -- test path
	if err != nil {
		return nil, err
	}
	var s installState
	if err := yaml.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return &s, nil
}
