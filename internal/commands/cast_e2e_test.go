package commands

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// buildAilloyBinary compiles the ailloy CLI into a temp dir and returns the
// binary path. Skips on Windows (different path semantics) and on non-go
// environments.
func buildAilloyBinary(t *testing.T) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("e2e binary test currently skips Windows path conventions")
	}
	bin := filepath.Join(t.TempDir(), "ailloy")
	// Compile from the project root (assumed two levels up from this file).
	cmd := exec.Command("go", "build", "-o", bin, "./cmd/ailloy")
	// Walk up to the repo root by finding go.mod.
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	root := wd
	for {
		if _, err := os.Stat(filepath.Join(root, "go.mod")); err == nil {
			break
		}
		parent := filepath.Dir(root)
		if parent == root {
			t.Fatalf("could not find go.mod from %s", wd)
		}
		root = parent
	}
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go build failed: %v\n%s", err, out)
	}
	return bin
}

// TestE2E_MergeStrategy_FullCLI builds the actual ailloy binary and runs
// `ailloy cast` against an on-disk synthetic mold. Verifies that the merge
// pipeline works end-to-end, not just at the function-call level.
func TestE2E_MergeStrategy_FullCLI(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e binary build is slow; skipping in -short mode")
	}
	bin := buildAilloyBinary(t)

	// Set up a project dir to cast into.
	projectDir := t.TempDir()
	// Pre-seed an existing opencode.json with the "outline" MCP server.
	preExisting := []byte(`{"mcp":{"outline":{"url":"https://outline"}}}`)
	if err := os.WriteFile(filepath.Join(projectDir, "opencode.json"), preExisting, 0644); err != nil {
		t.Fatal(err)
	}

	// Build a real on-disk mold dir.
	moldDir := t.TempDir()
	mustWrite := func(rel, content string) {
		t.Helper()
		full := filepath.Join(moldDir, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	mustWrite("mold.yaml", "apiVersion: v1\nkind: Mold\nname: e2e-test\nversion: 0.1.0\n")
	mustWrite("flux.yaml", `output:
  config/opencode.json:
    dest: opencode.json
    strategy: merge
`)
	mustWrite("config/opencode.json", `{"mcp":{"replicated-docs":{"url":"https://docs"}}}`)

	// Run `ailloy cast <moldDir>` from the project dir.
	cmd := exec.Command(bin, "cast", moldDir)
	cmd.Dir = projectDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("ailloy cast failed: %v\noutput:\n%s", err, out)
	}

	// Verify the merged file on disk.
	got, err := os.ReadFile(filepath.Join(projectDir, "opencode.json"))
	if err != nil {
		t.Fatalf("readback: %v", err)
	}
	gs := string(got)
	if !strings.Contains(gs, "outline") {
		t.Errorf("outline missing after e2e merge:\n%s", gs)
	}
	if !strings.Contains(gs, "replicated-docs") {
		t.Errorf("replicated-docs missing after e2e merge:\n%s", gs)
	}
	if strings.Index(gs, "outline") > strings.Index(gs, "replicated-docs") {
		t.Errorf("expected outline before replicated-docs (base before overlay), got:\n%s", gs)
	}
}

// TestE2E_MergeStrategy_HandEditedFileErrors confirms the user-facing CLI
// error for an unparseable existing destination is actionable.
func TestE2E_MergeStrategy_HandEditedFileErrors(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e binary build is slow; skipping in -short mode")
	}
	bin := buildAilloyBinary(t)

	projectDir := t.TempDir()
	// Pre-seed unparseable JSON (simulates a hand-edit gone wrong).
	if err := os.WriteFile(filepath.Join(projectDir, "opencode.json"), []byte("not json{{"), 0644); err != nil {
		t.Fatal(err)
	}

	moldDir := t.TempDir()
	mustWrite := func(rel, content string) {
		t.Helper()
		full := filepath.Join(moldDir, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	mustWrite("mold.yaml", "apiVersion: v1\nkind: Mold\nname: e2e-test\nversion: 0.1.0\n")
	mustWrite("flux.yaml", `output:
  config/opencode.json:
    dest: opencode.json
    strategy: merge
`)
	mustWrite("config/opencode.json", `{"mcp":{"x":{}}}`)

	// First run without the flag — should fail with actionable message.
	cmd := exec.Command(bin, "cast", moldDir)
	cmd.Dir = projectDir
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected non-zero exit on unparseable dest, got success.\nstdout/stderr:\n%s", out)
	}
	combinedOut := string(out)
	for _, must := range []string{"opencode.json", "force-replace-on-parse-error"} {
		if !strings.Contains(combinedOut, must) {
			t.Errorf("CLI error must include %q for actionable UX; got:\n%s", must, combinedOut)
		}
	}

	// Verify the unparseable file was NOT clobbered.
	preserved, err := os.ReadFile(filepath.Join(projectDir, "opencode.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(preserved) != "not json{{" {
		t.Errorf("hand-edited file was clobbered without --force; got:\n%s", preserved)
	}

	// Second run with the flag — should succeed and clobber.
	cmd2 := exec.Command(bin, "cast", moldDir, "--force-replace-on-parse-error")
	cmd2.Dir = projectDir
	out2, err := cmd2.CombinedOutput()
	if err != nil {
		t.Fatalf("--force-replace-on-parse-error should succeed: %v\noutput:\n%s", err, out2)
	}
	finalBytes, _ := os.ReadFile(filepath.Join(projectDir, "opencode.json"))
	// After --force-replace-on-parse-error, the unparseable existing file is
	// discarded and the mold output is written through.
	if !strings.Contains(string(finalBytes), `"x"`) {
		t.Errorf("expected mold content (with mcp.x) after force-replace; got:\n%s", finalBytes)
	}
	if string(finalBytes) == "not json{{" {
		t.Errorf("expected hand-edited content to be replaced after --force-replace-on-parse-error; still got:\n%s", finalBytes)
	}
}

// TestE2E_MergeStrategy_HelpOutput verifies the new flag appears in
// `ailloy cast --help` and `ailloy forge --help`.
func TestE2E_MergeStrategy_HelpOutput(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e binary build is slow; skipping in -short mode")
	}
	bin := buildAilloyBinary(t)

	for _, sub := range []string{"cast", "forge"} {
		t.Run(sub, func(t *testing.T) {
			out, err := exec.Command(bin, sub, "--help").CombinedOutput()
			if err != nil {
				t.Fatalf("%s --help failed: %v\n%s", sub, err, out)
			}
			if !strings.Contains(string(out), "force-replace-on-parse-error") {
				t.Errorf("ailloy %s --help should mention --force-replace-on-parse-error; got:\n%s", sub, out)
			}
		})
	}
}
