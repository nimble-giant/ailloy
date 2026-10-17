package commands

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nimble-giant/ailloy/internal/server"
)

// writeServeTestMold builds a small mold exercising flux, an ingot, an
// append strategy, a workflow and an empty-rendering blank.
func writeServeTestMold(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, d := range []string{"commands", "ingots", ".github/workflows"} {
		if err := os.MkdirAll(filepath.Join(dir, d), 0o750); err != nil {
			t.Fatal(err)
		}
	}
	mustWrite(t, filepath.Join(dir, "mold.yaml"), "apiVersion: v1\nkind: mold\nname: serve-demo\nversion: 0.1.0\n")
	mustWrite(t, filepath.Join(dir, "flux.yaml"), `project:
  organization: default-org
agent:
  targets: []
output:
  commands: .claude/commands
  .github: .github
  NOTES.md:
    dest: CLAUDE.md
    strategy: append
`)
	mustWrite(t, filepath.Join(dir, "ingots", "sig.md"), "-- {{.project.organization}}")
	mustWrite(t, filepath.Join(dir, "commands", "hello.md"), "Hello {{.project.organization}} {{ingot \"sig\"}}\ntargets={{.agent.targets}}\n")
	mustWrite(t, filepath.Join(dir, "commands", "empty.md"), "{{if false}}never{{end}}\n")
	mustWrite(t, filepath.Join(dir, ".github", "workflows", "ci.yml"), "name: ci\n")
	mustWrite(t, filepath.Join(dir, "NOTES.md"), "notes for {{.project.organization}}\n")
	return dir
}

func filesByPath(res *server.CastResult) map[string]server.File {
	out := map[string]server.File{}
	for _, f := range res.Files {
		out[f.Path] = f
	}
	return out
}

func TestServeRenderer_RendersMold(t *testing.T) {
	moldDir := writeServeTestMold(t)
	r := &serveRenderer{allowLocal: true, logger: log.New(io.Discard, "", 0)}

	res, err := r.Render(context.Background(), server.CastRequest{
		Mold:   moldDir,
		Values: map[string]any{"project": map[string]any{"organization": "from-values"}},
		Set:    []string{"project.organization=acme", "agent.targets=[claude,codex]"},
	})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if res.Mold != "serve-demo" {
		t.Fatalf("Mold = %q", res.Mold)
	}
	files := filesByPath(res)
	hello, ok := files[".claude/commands/hello.md"]
	if !ok {
		t.Fatalf("hello.md missing; got %v", files)
	}
	want := "Hello acme -- acme\ntargets=[claude codex]\n"
	if string(hello.Content) != want {
		t.Fatalf("hello.md = %q, want %q", hello.Content, want)
	}
	if _, ok := files[".claude/commands/empty.md"]; ok {
		t.Fatal("blank that renders empty should be skipped, as in cast and forge")
	}
	if _, ok := files[".github/workflows/ci.yml"]; ok {
		t.Fatal("workflows must be excluded unless with_workflows is set")
	}
	if notes := files["CLAUDE.md"]; notes.Strategy != "append" || string(notes.Content) != "notes for acme\n" {
		t.Fatalf("CLAUDE.md = %+v", notes)
	}
}

func TestServeRenderer_ValuesApplyBelowSet(t *testing.T) {
	moldDir := writeServeTestMold(t)
	r := &serveRenderer{allowLocal: true, logger: log.New(io.Discard, "", 0)}
	res, err := r.Render(context.Background(), server.CastRequest{
		Mold:   moldDir,
		Values: map[string]any{"project": map[string]any{"organization": "from-values"}},
	})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if got := string(filesByPath(res)[".claude/commands/hello.md"].Content); !strings.HasPrefix(got, "Hello from-values") {
		t.Fatalf("hello.md = %q", got)
	}
}

func TestServeRenderer_WithWorkflows(t *testing.T) {
	moldDir := writeServeTestMold(t)
	r := &serveRenderer{allowLocal: true, logger: log.New(io.Discard, "", 0)}
	res, err := r.Render(context.Background(), server.CastRequest{Mold: moldDir, WithWorkflows: true})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if _, ok := filesByPath(res)[".github/workflows/ci.yml"]; !ok {
		t.Fatal("with_workflows should include .github/ blanks")
	}
}

// The server must never read ingots from its own working directory or home:
// only the requested mold's tree is in scope.
func TestServeRenderer_IgnoresServerLocalIngots(t *testing.T) {
	moldDir := t.TempDir()
	mustWrite(t, filepath.Join(moldDir, "mold.yaml"), "apiVersion: v1\nkind: mold\nname: leak\nversion: 0.1.0\n")
	mustWrite(t, filepath.Join(moldDir, "flux.yaml"), "output:\n  commands: .claude/commands\n")
	if err := os.MkdirAll(filepath.Join(moldDir, "commands"), 0o750); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(moldDir, "commands", "x.md"), `{{ingot "secret"}}`)

	cwd := t.TempDir()
	if err := os.MkdirAll(filepath.Join(cwd, "ingots"), 0o750); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(cwd, "ingots", "secret.md"), "server secret")
	t.Chdir(cwd)

	r := &serveRenderer{allowLocal: true, logger: log.New(io.Discard, "", 0)}
	_, err := r.Render(context.Background(), server.CastRequest{Mold: moldDir})
	var he *server.Error
	if !errors.As(err, &he) || he.Status != http.StatusUnprocessableEntity {
		t.Fatalf("err = %v, want 422 ingot-not-found", err)
	}
}

func TestServeRenderer_RejectsLocalRefs(t *testing.T) {
	r := &serveRenderer{logger: log.New(io.Discard, "", 0)}
	for _, ref := range []string{"/etc", "./mold", "~/mold", "mold"} {
		_, err := r.Render(context.Background(), server.CastRequest{Mold: ref})
		var he *server.Error
		if !errors.As(err, &he) || he.Status != http.StatusBadRequest {
			t.Fatalf("%s: err = %v, want 400", ref, err)
		}
	}
}

func TestServeRenderer_EnforcesAllowedSources(t *testing.T) {
	r := &serveRenderer{allowSources: []string{"github.com/acme"}, logger: log.New(io.Discard, "", 0)}
	_, err := r.Render(context.Background(), server.CastRequest{Mold: "github.com/evil/molds"})
	var he *server.Error
	if !errors.As(err, &he) || he.Status != http.StatusForbidden {
		t.Fatalf("err = %v, want 403", err)
	}
}

func TestSourceAllowed(t *testing.T) {
	cases := []struct {
		ref      string
		patterns []string
		want     bool
	}{
		{"github.com/acme/molds", nil, true},
		{"github.com/acme/molds", []string{"github.com/acme"}, true},
		{"https://github.com/acme/molds@v1.2.0", []string{"github.com/acme"}, true},
		{"git@github.com:acme/molds.git", []string{"github.com/acme/molds"}, true},
		{"github.com/acme/molds//sub/path@v1", []string{"github.com/acme/molds"}, true},
		{"GitHub.com/ACME/molds", []string{"github.com/acme"}, true},
		{"github.com/acme-evil/molds", []string{"github.com/acme"}, false},
		{"github.com/acme/molds-evil", []string{"github.com/acme/molds"}, false},
		{"gitlab.com/acme/molds", []string{"github.com/acme"}, false},
		{"github.com/other/x", []string{"github.com/acme", "https://github.com/other/"}, true},
	}
	for _, c := range cases {
		got, err := sourceAllowed(c.ref, c.patterns)
		if err != nil {
			t.Fatalf("%s: %v", c.ref, err)
		}
		if got != c.want {
			t.Errorf("sourceAllowed(%q, %v) = %v, want %v", c.ref, c.patterns, got, c.want)
		}
	}
}

func TestLoadServeTokens(t *testing.T) {
	file := filepath.Join(t.TempDir(), "tokens")
	mustWrite(t, file, "# comment\n alpha \n\nbeta\n")
	got, err := loadServeTokens(file, "gamma, delta,")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(got, ",") != "alpha,beta,gamma,delta" {
		t.Fatalf("tokens = %v", got)
	}
	if _, err := loadServeTokens(filepath.Join(t.TempDir(), "missing"), ""); err == nil {
		t.Fatal("expected error for a missing token file")
	}
}

// End to end: HTTP request → server → real render pipeline → tarball.
func TestServe_EndToEndTarball(t *testing.T) {
	moldDir := writeServeTestMold(t)
	h, err := server.New(server.Config{
		Renderer: &serveRenderer{allowLocal: true, logger: log.New(io.Discard, "", 0)},
		Tokens:   []string{"tok"},
		Logger:   log.New(io.Discard, "", 0),
	})
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(h)
	defer ts.Close()

	body := `{"mold":"` + moldDir + `","set":["project.organization=acme"],"targets":["claude","opencode"]}`
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/v1/cast", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer tok")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d: %s", resp.StatusCode, b)
	}
	gz, err := gzip.NewReader(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	tr := tar.NewReader(gz)
	got := map[string]string{}
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		b, _ := io.ReadAll(tr)
		got[hdr.Name] = string(b)
	}
	if got[".claude/commands/hello.md"] != "Hello acme -- acme\ntargets=[claude opencode]\n" {
		t.Fatalf("tar = %v", got)
	}
}

// writeIngotDepMold builds a mold that declares two ingot dependencies: a
// single-package ingot repo and a multi-package one (ingots/<name>/), plus a
// bundled ingot whose name collides with a dependency's.
func writeIngotDepMold(t *testing.T) string {
	t.Helper()
	single := t.TempDir()
	mustWrite(t, filepath.Join(single, "ingot.yaml"), "apiVersion: v1\nkind: ingot\nname: greeting\nversion: 1.0.0\nfiles: [body.md]\n")
	mustWrite(t, filepath.Join(single, "body.md"), "Hi from {{.project.organization}}")

	multi := t.TempDir()
	writeMultiIngotFixture(t, multi, "alpha", "shadowed")

	moldDir := t.TempDir()
	for _, d := range []string{"commands", "ingots"} {
		if err := os.MkdirAll(filepath.Join(moldDir, d), 0o750); err != nil {
			t.Fatal(err)
		}
	}
	mustWrite(t, filepath.Join(moldDir, "mold.yaml"), `apiVersion: v1
kind: mold
name: deps-demo
version: 0.1.0
dependencies:
  - ingot: `+single+`
    version: 1.0.0
  - ingot: `+multi+`
    version: 1.0.0
`)
	mustWrite(t, filepath.Join(moldDir, "flux.yaml"), "project:\n  organization: acme\noutput:\n  commands: .claude/commands\n")
	mustWrite(t, filepath.Join(moldDir, "ingots", "shadowed.md"), "bundled wins")
	mustWrite(t, filepath.Join(moldDir, "commands", "x.md"), `{{ingot "greeting"}} | {{ingot "alpha"}} | {{ingot "shadowed"}}`+"\n")
	return moldDir
}

// Ingots a mold declares as dependencies resolve in memory, like cast's
// installed .ailloy/ingots/, with the mold's own ingots taking precedence.
func TestServeRenderer_ResolvesIngotDependencies(t *testing.T) {
	moldDir := writeIngotDepMold(t)
	t.Chdir(t.TempDir())
	r := &serveRenderer{allowLocal: true, logger: log.New(io.Discard, "", 0)}
	res, err := r.Render(context.Background(), server.CastRequest{Mold: moldDir})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	got := string(filesByPath(res)[".claude/commands/x.md"].Content)
	want := "Hi from acme | # alpha\n | bundled wins\n"
	if got != want {
		t.Fatalf("x.md = %q, want %q", got, want)
	}
	if _, err := os.Stat(".ailloy"); !os.IsNotExist(err) {
		t.Fatal("serve must not install ingot deps into .ailloy/")
	}
}
