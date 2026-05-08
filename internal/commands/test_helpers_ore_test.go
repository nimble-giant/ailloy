package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// fakeFoundryHelpers — Phase 12 test helpers for ore packaging.
//
// IMPLEMENTATION NOTE (fallback)
// ------------------------------
// The original Phase 12 plan called for a "real" fake-foundry helper that
// runs `git init` + `git tag` and produces a reference resolvable through
// `foundry.ResolveWithMetadata`. The public foundry resolver hard-codes the
// HTTPS clone URL (`<host>/<owner>/<repo>.git`) and exposes no GitRunner
// override on its public entry points (only ResolveOption + WithLockPath).
// Wiring a local file-backed git remote through ResolveWithMetadata would
// require either (a) a refactor of foundry.go to inject GitRunner / clone-URL,
// or (b) an HTTP/git server stood up per test. Both are larger than the
// 30-60 minute budget the plan allows.
//
// As permitted by the plan, we fall back to local-path-based helpers that
// exercise the same install_deps → cascade-uninstall pipeline used by remote
// refs, just bypassing the foundry resolver. The helpers do still run
// `git init`, `git add`, `git commit`, and `git tag v<version>` so that
// tagged-commit-style ores are produced on disk for any future test that
// upgrades to a real foundry harness — the directory layout and tag name are
// already correct for that path.

// buildFakeOreFoundry creates a directory containing ore.yaml + flux.schema.yaml
// + flux.yaml, runs git init + tag v<version>, and returns the directory path
// (suitable as a local-path dependency reference) along with a cleanup func.
//
// Returned refURL is currently a filesystem path — the local-path branch of
// resolveDepFS handles it. When/if the foundry resolver gains a GitRunner
// hook, callers can pass `file://<path>` (or a synthesized host triple) and
// have the helper drive the resolver instead.
func buildFakeOreFoundry(t *testing.T, baseDir, name, version string) (refURL string, cleanup func()) {
	t.Helper()
	dir := filepath.Join(baseDir, "fake-ore-"+name)
	if err := os.MkdirAll(dir, 0750); err != nil {
		t.Fatalf("mkdir fake ore: %v", err)
	}

	files := map[string]string{
		"ore.yaml": fmt.Sprintf(
			"apiVersion: v1\nkind: ore\nname: %s\nversion: %s\n",
			name, trimV(version),
		),
		"flux.schema.yaml": "- name: enabled\n  type: bool\n  default: \"false\"\n" +
			"- name: greeting\n  type: string\n  default: \"hello\"\n",
		"flux.yaml": "enabled: false\ngreeting: \"hello\"\n",
	}
	for fn, body := range files {
		if err := os.WriteFile(filepath.Join(dir, fn), []byte(body), 0644); err != nil {
			t.Fatalf("write %s: %v", fn, err)
		}
	}

	// Run git init/add/commit/tag so the directory looks like a real
	// foundry-style source. Failures here are non-fatal: if git is missing
	// the directory is still a valid local-path ore.
	gitInitTagAndCommit(t, dir, version)

	cleanup = func() { _ = os.RemoveAll(dir) }
	return dir, cleanup
}

// buildFakeMoldFoundryWithOreDep creates a directory containing a minimal
// mold.yaml that declares a dependency on the given ore reference, plus a
// blank file that templates against the ore namespace. Returns the directory
// path and a cleanup func.
//
// The mold's flux schema is supplied entirely by the ore overlay — this is
// the realistic shape for a consumer mold that lifts its config out into a
// shared ore.
func buildFakeMoldFoundryWithOreDep(t *testing.T, baseDir, oreRefURL string) (moldRefURL string, cleanup func()) {
	t.Helper()
	dir := filepath.Join(baseDir, "fake-mold")
	if err := os.MkdirAll(filepath.Join(dir, "commands"), 0750); err != nil {
		t.Fatalf("mkdir fake mold: %v", err)
	}

	moldYAML := fmt.Sprintf(`apiVersion: v1
kind: Mold
name: status-consumer
version: 0.1.0
dependencies:
  - ore: %s
    version: 1.0.0
`, oreRefURL)

	fluxYAML := `output:
  commands/status.md:
    dest: .claude/commands/status.md
`
	statusBlank := `# Status command
{{- if .ore.status.enabled }}
Status reporting enabled.
{{- else }}
Status reporting disabled — greeting is "{{ index (index .ore "status") "greeting" }}".
{{- end }}
`

	files := map[string]string{
		"mold.yaml":          moldYAML,
		"flux.yaml":          fluxYAML,
		"commands/status.md": statusBlank,
	}
	for fn, body := range files {
		full := filepath.Join(dir, fn)
		if err := os.MkdirAll(filepath.Dir(full), 0750); err != nil {
			t.Fatalf("mkdir parent of %s: %v", fn, err)
		}
		if err := os.WriteFile(full, []byte(body), 0644); err != nil {
			t.Fatalf("write %s: %v", fn, err)
		}
	}

	gitInitTagAndCommit(t, dir, "v0.1.0")

	cleanup = func() { _ = os.RemoveAll(dir) }
	return dir, cleanup
}

// gitInitTagAndCommit runs `git init`, configures a local user, stages
// everything, commits, and tags the working tree. Non-fatal — if git isn't
// available the directory is still usable as a local-path foundry source.
//
// The local-path test pipeline doesn't actually consult the git tag; tagging
// is preserved so the same helper produces a foundry-shaped directory ready
// for a future fake-foundry harness.
func gitInitTagAndCommit(t *testing.T, dir, version string) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		return // git missing — skip; local-path branch still works
	}
	tag := version
	if len(tag) > 0 && tag[0] != 'v' {
		tag = "v" + tag
	}
	cmds := [][]string{
		{"git", "init", "-q"},
		{"git", "config", "user.email", "test@example.invalid"},
		{"git", "config", "user.name", "Phase 12 Test"},
		{"git", "config", "commit.gpgsign", "false"},
		{"git", "add", "-A"},
		{"git", "commit", "-q", "-m", "initial"},
		{"git", "tag", tag},
	}
	for _, c := range cmds {
		cmd := exec.Command(c[0], c[1:]...) //#nosec G204 -- arguments are constructed in-test
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Logf("git %v failed (non-fatal): %v\n%s", c[1:], err, out)
			return
		}
	}
}

// trimV strips a leading "v" from a version string so the on-disk ore.yaml
// records the bare semver (matching what `ailloy ore new` writes).
func trimV(v string) string {
	if len(v) > 0 && v[0] == 'v' {
		return v[1:]
	}
	return v
}
