package merge

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAppendFile_NonexistentDest(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "AGENTS.md")
	if err := AppendFile(dest, []byte("# Wiki\n\nDocs here."), AppendOptions{MoldName: "wiki"}); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(dest)
	want := "<!-- ailloy:mold=wiki:start -->\n# Wiki\n\nDocs here.\n<!-- ailloy:mold=wiki:end -->\n"
	if string(got) != want {
		t.Errorf("nonexistent dest output mismatch.\nwant:\n%q\ngot:\n%q", want, string(got))
	}
}

func TestAppendFile_ExistingNoBlock_AppendsBlock(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "AGENTS.md")
	if err := os.WriteFile(dest, []byte("# Project AGENTS\n\nHand-written content."), 0644); err != nil {
		t.Fatal(err)
	}
	if err := AppendFile(dest, []byte("From wiki."), AppendOptions{MoldName: "wiki"}); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(dest)
	gs := string(got)
	if !strings.HasPrefix(gs, "# Project AGENTS\n\nHand-written content.") {
		t.Errorf("hand-written content corrupted:\n%s", gs)
	}
	if !strings.Contains(gs, "<!-- ailloy:mold=wiki:start -->") {
		t.Errorf("sentinel start missing:\n%s", gs)
	}
	if !strings.Contains(gs, "<!-- ailloy:mold=wiki:end -->") {
		t.Errorf("sentinel end missing:\n%s", gs)
	}
}

func TestAppendFile_TwoMolds_BothBlocksPresent(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "AGENTS.md")
	if err := AppendFile(dest, []byte("from wiki"), AppendOptions{MoldName: "wiki"}); err != nil {
		t.Fatal(err)
	}
	if err := AppendFile(dest, []byte("from docs"), AppendOptions{MoldName: "docs"}); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(dest)
	gs := string(got)
	for _, want := range []string{"<!-- ailloy:mold=wiki:start -->", "<!-- ailloy:mold=wiki:end -->",
		"<!-- ailloy:mold=docs:start -->", "<!-- ailloy:mold=docs:end -->",
		"from wiki", "from docs"} {
		if !strings.Contains(gs, want) {
			t.Errorf("missing %q in:\n%s", want, gs)
		}
	}
	// Order: wiki before docs (cast order).
	if strings.Index(gs, "wiki:start") > strings.Index(gs, "docs:start") {
		t.Errorf("expected wiki before docs:\n%s", gs)
	}
}

func TestAppendFile_RecastIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "AGENTS.md")
	if err := AppendFile(dest, []byte("v1 content"), AppendOptions{MoldName: "wiki"}); err != nil {
		t.Fatal(err)
	}
	if err := AppendFile(dest, []byte("v2 content"), AppendOptions{MoldName: "wiki"}); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(dest)
	gs := string(got)
	if strings.Count(gs, "wiki:start") != 1 {
		t.Errorf("re-cast should not duplicate sentinel; got:\n%s", gs)
	}
	if strings.Contains(gs, "v1 content") {
		t.Errorf("re-cast should replace v1 content with v2; got:\n%s", gs)
	}
	if !strings.Contains(gs, "v2 content") {
		t.Errorf("v2 content missing after re-cast:\n%s", gs)
	}
}

func TestAppendFile_RecastPreservesOtherMoldsAndForeignContent(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "AGENTS.md")
	// Set up: hand-written content + wiki block + foreign content + docs block + trailing hand-written
	initial := `# Project AGENTS

Hand-edited intro.

<!-- ailloy:mold=wiki:start -->
old wiki content
<!-- ailloy:mold=wiki:end -->

Some hand-written notes between blocks.

<!-- ailloy:mold=docs:start -->
docs content
<!-- ailloy:mold=docs:end -->

Trailing hand-written content.
`
	if err := os.WriteFile(dest, []byte(initial), 0644); err != nil {
		t.Fatal(err)
	}
	// Re-cast wiki only.
	if err := AppendFile(dest, []byte("new wiki content"), AppendOptions{MoldName: "wiki"}); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(dest)
	gs := string(got)
	for _, must := range []string{
		"# Project AGENTS",
		"Hand-edited intro.",
		"new wiki content",
		"Some hand-written notes between blocks.",
		"docs content",
		"Trailing hand-written content.",
	} {
		if !strings.Contains(gs, must) {
			t.Errorf("missing %q after wiki re-cast:\n%s", must, gs)
		}
	}
	if strings.Contains(gs, "old wiki content") {
		t.Errorf("old wiki content should have been replaced:\n%s", gs)
	}
	// Sentinels for both molds still present, exactly once.
	if strings.Count(gs, "wiki:start") != 1 || strings.Count(gs, "docs:start") != 1 {
		t.Errorf("expected exactly one sentinel per mold; got:\n%s", gs)
	}
}

func TestAppendFile_UnsupportedExtension(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "config.json")
	err := AppendFile(dest, []byte("x"), AppendOptions{MoldName: "wiki"})
	if err == nil {
		t.Fatal("expected error for non-markdown ext, got nil")
	}
	if !errors.Is(err, ErrUnsupportedAppendExt) {
		t.Errorf("expected ErrUnsupportedAppendExt, got: %v", err)
	}
}

func TestAppendFile_RequiresMoldName(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "AGENTS.md")
	err := AppendFile(dest, []byte("x"), AppendOptions{})
	if err == nil {
		t.Fatal("expected error when MoldName empty, got nil")
	}
	if !strings.Contains(err.Error(), "MoldName") {
		t.Errorf("error should mention MoldName; got: %v", err)
	}
}

func TestAppendFile_MarkdownExtVariants(t *testing.T) {
	for _, ext := range []string{".md", ".MD", ".markdown", ".Markdown"} {
		dir := t.TempDir()
		dest := filepath.Join(dir, "AGENTS"+ext)
		if err := AppendFile(dest, []byte("x"), AppendOptions{MoldName: "wiki"}); err != nil {
			t.Errorf("ext %q: unexpected error: %v", ext, err)
		}
	}
}
