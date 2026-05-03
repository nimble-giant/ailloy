package commands

import (
	"os"
	"strings"
	"testing"
)

func TestRunOreGet_NonRemoteRef_Errors(t *testing.T) {
	err := runOreGet(nil, []string{"./local/path"})
	if err == nil || !strings.Contains(err.Error(), "remote reference") {
		t.Errorf("expected remote-reference error, got %v", err)
	}
}

func TestRunOreAdd_NonRemoteRef_Errors(t *testing.T) {
	err := runOreAdd(nil, []string{"./local/path"})
	if err == nil || !strings.Contains(err.Error(), "remote reference") {
		t.Errorf("expected remote-reference error, got %v", err)
	}
}

func TestRunOreNew_RejectsBadName(t *testing.T) {
	for _, bad := range []string{"BadName", "bad-name", "9start", ""} {
		err := runOreNew(nil, []string{bad})
		if err == nil {
			t.Errorf("expected error for %q", bad)
		}
	}
}

func TestRunOreNew_RejectsExistingDirectory(t *testing.T) {
	dir := t.TempDir()
	cwd, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir("status", 0750); err != nil {
		t.Fatal(err)
	}
	if err := runOreNew(nil, []string{"status"}); err == nil {
		t.Error("expected error when directory exists")
	}
}

func TestRunOreNew_CreatesExpectedFiles(t *testing.T) {
	dir := t.TempDir()
	cwd, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	if err := runOreNew(nil, []string{"status"}); err != nil {
		t.Fatalf("runOreNew: %v", err)
	}
	for _, want := range []string{"status/ore.yaml", "status/flux.schema.yaml", "status/flux.yaml"} {
		if _, err := os.Stat(want); err != nil {
			t.Errorf("missing %s: %v", want, err)
		}
	}
}
