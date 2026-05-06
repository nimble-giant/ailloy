package commands

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHumanizeBytes(t *testing.T) {
	cases := []struct {
		in   int64
		want string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1572864, "1.5 MB"},
		{1073741824, "1.0 GB"},
	}
	for _, tc := range cases {
		got := humanizeBytes(tc.in)
		if got != tc.want {
			t.Errorf("humanizeBytes(%d) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestGatherMoldStatsEmpty(t *testing.T) {
	dir := t.TempDir()
	stats, err := gatherMoldStats(dir, filepath.Join(dir, "indexes"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.Refs != 0 || stats.Versions != 0 || stats.Bytes != 0 {
		t.Errorf("expected zero stats, got %+v", stats)
	}
}

func TestGatherMoldStatsCountsAndSkipsIndexes(t *testing.T) {
	root := t.TempDir()
	mustMkdirAll(t, filepath.Join(root, "github.com", "foo", "bar", "v1"))
	mustWriteFile(t, filepath.Join(root, "github.com", "foo", "bar", "v1", "README.md"), make([]byte, 100))
	mustMkdirAll(t, filepath.Join(root, "github.com", "foo", "bar", "v2"))
	mustWriteFile(t, filepath.Join(root, "github.com", "foo", "bar", "v2", "README.md"), make([]byte, 200))
	mustMkdirAll(t, filepath.Join(root, "gitlab.com", "baz", "qux", "v1"))
	mustWriteFile(t, filepath.Join(root, "gitlab.com", "baz", "qux", "v1", "README.md"), make([]byte, 50))
	indexRoot := filepath.Join(root, "indexes")
	mustMkdirAll(t, indexRoot)
	mustWriteFile(t, filepath.Join(indexRoot, "foundry.yaml"), make([]byte, 9999))

	stats, err := gatherMoldStats(root, indexRoot)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.Refs != 2 {
		t.Errorf("Refs = %d, want 2", stats.Refs)
	}
	if stats.Versions != 3 {
		t.Errorf("Versions = %d, want 3", stats.Versions)
	}
	if stats.Bytes != 350 {
		t.Errorf("Bytes = %d, want 350 (indexes/ should be skipped)", stats.Bytes)
	}
}

func TestGatherIndexStatsEmpty(t *testing.T) {
	dir := t.TempDir()
	stats, err := gatherIndexStats(filepath.Join(dir, "does-not-exist"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.Indexes != 0 || stats.Bytes != 0 {
		t.Errorf("expected zero stats, got %+v", stats)
	}
}

func TestGatherIndexStatsCounts(t *testing.T) {
	root := t.TempDir()
	mustMkdirAll(t, filepath.Join(root, "github.com", "alice", "molds"))
	mustWriteFile(t, filepath.Join(root, "github.com", "alice", "molds", "foundry.yaml"), make([]byte, 1000))
	mustMkdirAll(t, filepath.Join(root, "abc123hashurl"))
	mustWriteFile(t, filepath.Join(root, "abc123hashurl", "foundry.yaml"), make([]byte, 500))
	// A directory without foundry.yaml should not be counted as an index.
	mustMkdirAll(t, filepath.Join(root, "stray"))
	mustWriteFile(t, filepath.Join(root, "stray", "other.txt"), make([]byte, 7))

	stats, err := gatherIndexStats(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.Indexes != 2 {
		t.Errorf("Indexes = %d, want 2", stats.Indexes)
	}
	if stats.Bytes != 1507 {
		t.Errorf("Bytes = %d, want 1507", stats.Bytes)
	}
}

func TestRemoveMoldsPreservesIndexes(t *testing.T) {
	root := t.TempDir()
	mustMkdirAll(t, filepath.Join(root, "github.com", "foo", "bar", "v1"))
	mustWriteFile(t, filepath.Join(root, "github.com", "foo", "bar", "v1", "x"), []byte("x"))
	mustMkdirAll(t, filepath.Join(root, "gitlab.com", "baz", "qux", "v1"))
	mustWriteFile(t, filepath.Join(root, "gitlab.com", "baz", "qux", "v1", "y"), []byte("y"))
	mustMkdirAll(t, filepath.Join(root, "indexes", "github.com", "alice", "molds"))
	mustWriteFile(t, filepath.Join(root, "indexes", "github.com", "alice", "molds", "foundry.yaml"), []byte("z"))

	removed, errs := removeMolds(root)
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if removed != 2 {
		t.Errorf("removed = %d, want 2 (top-level non-indexes entries)", removed)
	}
	if _, err := os.Stat(filepath.Join(root, "github.com")); !os.IsNotExist(err) {
		t.Errorf("github.com should have been removed, err = %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "gitlab.com")); !os.IsNotExist(err) {
		t.Errorf("gitlab.com should have been removed, err = %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "indexes", "github.com", "alice", "molds", "foundry.yaml")); err != nil {
		t.Errorf("indexes/ should have been preserved, err = %v", err)
	}
}

func TestRemoveMoldsMissingDirIsOK(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "nope")
	removed, errs := removeMolds(missing)
	if len(errs) != 0 {
		t.Errorf("expected no errors, got %v", errs)
	}
	if removed != 0 {
		t.Errorf("removed = %d, want 0", removed)
	}
}

func TestRemoveIndexes(t *testing.T) {
	root := t.TempDir()
	mustMkdirAll(t, filepath.Join(root, "github.com", "alice", "molds"))
	mustWriteFile(t, filepath.Join(root, "github.com", "alice", "molds", "foundry.yaml"), []byte("z"))

	if err := removeIndexes(root); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Errorf("indexRoot should have been removed, err = %v", err)
	}
}

func TestRemoveIndexesMissingDirIsOK(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "nope")
	if err := removeIndexes(missing); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestRenderCachePreviewBoth(t *testing.T) {
	out := renderCachePreview(
		"~/.ailloy/cache",
		&moldStats{Refs: 12, Versions: 47, Bytes: 327600000}, // ~312.4 MB
		&indexStats{Indexes: 3, Bytes: 1258291},              // ~1.2 MB
	)
	for _, want := range []string{
		"Ailloy cache:",
		"~/.ailloy/cache",
		"Molds",
		"12 refs, 47 versions",
		"312.4 MB",
		"Indexes",
		"3 indexes",
		"1.2 MB",
		"Total:",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("preview missing %q in:\n%s", want, out)
		}
	}
}

func TestRenderCachePreviewMoldsOnly(t *testing.T) {
	out := renderCachePreview(
		"~/.ailloy/cache",
		&moldStats{Refs: 1, Versions: 1, Bytes: 100},
		nil,
	)
	if strings.Contains(out, "Indexes") {
		t.Errorf("expected no Indexes row, got:\n%s", out)
	}
	if !strings.Contains(out, "Molds") {
		t.Errorf("expected Molds row, got:\n%s", out)
	}
}

func TestRenderCachePreviewIndexesOnly(t *testing.T) {
	out := renderCachePreview(
		"~/.ailloy/cache",
		nil,
		&indexStats{Indexes: 1, Bytes: 100},
	)
	if strings.Contains(out, "Molds") {
		t.Errorf("expected no Molds row, got:\n%s", out)
	}
	if !strings.Contains(out, "Indexes") {
		t.Errorf("expected Indexes row, got:\n%s", out)
	}
}

func TestConfirmInteractiveYes(t *testing.T) {
	for _, in := range []string{"y\n", "Y\n", "yes\n"} {
		var out bytes.Buffer
		ok, err := confirmInteractive(strings.NewReader(in), &out, "Proceed? [y/N] ")
		if err != nil {
			t.Errorf("input %q: unexpected error %v", in, err)
		}
		if !ok {
			t.Errorf("input %q: expected ok=true", in)
		}
		if !strings.Contains(out.String(), "Proceed?") {
			t.Errorf("input %q: prompt not written, got %q", in, out.String())
		}
	}
}

func TestConfirmInteractiveNo(t *testing.T) {
	for _, in := range []string{"n\n", "no\n", "\n", "anything\n"} {
		var out bytes.Buffer
		ok, err := confirmInteractive(strings.NewReader(in), &out, "Proceed? ")
		if err != nil {
			t.Errorf("input %q: unexpected error %v", in, err)
		}
		if ok {
			t.Errorf("input %q: expected ok=false", in)
		}
	}
}

func TestConfirmInteractiveEOF(t *testing.T) {
	var out bytes.Buffer
	ok, err := confirmInteractive(strings.NewReader(""), &out, "Proceed? ")
	if err != nil && err != io.EOF {
		t.Errorf("expected nil or EOF, got %v", err)
	}
	if ok {
		t.Errorf("expected ok=false on EOF")
	}
}

func newTempCacheDirs(t *testing.T) (moldRoot, indexRoot string) {
	t.Helper()
	moldRoot = t.TempDir()
	indexRoot = filepath.Join(moldRoot, "indexes")
	mustMkdirAll(t, filepath.Join(moldRoot, "github.com", "foo", "bar", "v1"))
	mustWriteFile(t, filepath.Join(moldRoot, "github.com", "foo", "bar", "v1", "x"), []byte("xx"))
	mustMkdirAll(t, filepath.Join(indexRoot, "github.com", "alice", "molds"))
	mustWriteFile(t, filepath.Join(indexRoot, "github.com", "alice", "molds", "foundry.yaml"), []byte("yy"))
	return moldRoot, indexRoot
}

func TestExecuteCacheClearEmpty(t *testing.T) {
	moldRoot := t.TempDir()
	indexRoot := filepath.Join(moldRoot, "indexes")
	var out bytes.Buffer
	exit, err := executeCacheClear(cacheClearOptions{
		MoldRoot: moldRoot, IndexRoot: indexRoot,
		Molds: false, Indexes: false,
		Yes:    true,
		Stdout: &out, Stdin: strings.NewReader(""), IsTTY: func() bool { return false },
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exit != 0 {
		t.Errorf("exit = %d, want 0", exit)
	}
	if !strings.Contains(out.String(), "Cache is already empty") {
		t.Errorf("output missing 'Cache is already empty', got:\n%s", out.String())
	}
}

func TestExecuteCacheClearDefaultWipesBoth(t *testing.T) {
	moldRoot, indexRoot := newTempCacheDirs(t)
	var out bytes.Buffer
	exit, err := executeCacheClear(cacheClearOptions{
		MoldRoot: moldRoot, IndexRoot: indexRoot,
		Yes:    true,
		Stdout: &out, Stdin: strings.NewReader(""), IsTTY: func() bool { return false },
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exit != 0 {
		t.Errorf("exit = %d, want 0", exit)
	}
	if _, err := os.Stat(filepath.Join(moldRoot, "github.com")); !os.IsNotExist(err) {
		t.Errorf("github.com should be gone, err = %v", err)
	}
	if _, err := os.Stat(indexRoot); !os.IsNotExist(err) {
		t.Errorf("indexRoot should be gone, err = %v", err)
	}
	if !strings.Contains(out.String(), "Cleared") {
		t.Errorf("output missing 'Cleared', got:\n%s", out.String())
	}
}

func TestExecuteCacheClearMoldsOnly(t *testing.T) {
	moldRoot, indexRoot := newTempCacheDirs(t)
	var out bytes.Buffer
	exit, err := executeCacheClear(cacheClearOptions{
		MoldRoot: moldRoot, IndexRoot: indexRoot,
		Molds:  true,
		Yes:    true,
		Stdout: &out, Stdin: strings.NewReader(""), IsTTY: func() bool { return false },
	})
	if err != nil || exit != 0 {
		t.Fatalf("exit=%d err=%v", exit, err)
	}
	if _, err := os.Stat(filepath.Join(moldRoot, "github.com")); !os.IsNotExist(err) {
		t.Errorf("molds should be gone")
	}
	if _, err := os.Stat(filepath.Join(indexRoot, "github.com", "alice", "molds", "foundry.yaml")); err != nil {
		t.Errorf("indexes should be preserved, err = %v", err)
	}
	if strings.Contains(out.String(), "Indexes") {
		t.Errorf("output should not mention Indexes, got:\n%s", out.String())
	}
}

func TestExecuteCacheClearIndexesOnly(t *testing.T) {
	moldRoot, indexRoot := newTempCacheDirs(t)
	var out bytes.Buffer
	exit, err := executeCacheClear(cacheClearOptions{
		MoldRoot: moldRoot, IndexRoot: indexRoot,
		Indexes: true,
		Yes:     true,
		Stdout:  &out, Stdin: strings.NewReader(""), IsTTY: func() bool { return false },
	})
	if err != nil || exit != 0 {
		t.Fatalf("exit=%d err=%v", exit, err)
	}
	if _, err := os.Stat(filepath.Join(moldRoot, "github.com", "foo", "bar", "v1", "x")); err != nil {
		t.Errorf("molds should be preserved, err = %v", err)
	}
	if _, err := os.Stat(indexRoot); !os.IsNotExist(err) {
		t.Errorf("indexRoot should be gone, err = %v", err)
	}
}

func TestExecuteCacheClearDryRunDoesNotDelete(t *testing.T) {
	moldRoot, indexRoot := newTempCacheDirs(t)
	var out bytes.Buffer
	exit, err := executeCacheClear(cacheClearOptions{
		MoldRoot: moldRoot, IndexRoot: indexRoot,
		DryRun: true,
		Stdout: &out, Stdin: strings.NewReader(""), IsTTY: func() bool { return false },
	})
	if err != nil || exit != 0 {
		t.Fatalf("exit=%d err=%v", exit, err)
	}
	if _, err := os.Stat(filepath.Join(moldRoot, "github.com", "foo", "bar", "v1", "x")); err != nil {
		t.Errorf("molds should be preserved on dry-run")
	}
	if _, err := os.Stat(indexRoot); err != nil {
		t.Errorf("indexes should be preserved on dry-run")
	}
}

func TestExecuteCacheClearNonTTYNoYesErrors(t *testing.T) {
	moldRoot, indexRoot := newTempCacheDirs(t)
	var out bytes.Buffer
	exit, err := executeCacheClear(cacheClearOptions{
		MoldRoot: moldRoot, IndexRoot: indexRoot,
		Stdout: &out, Stdin: strings.NewReader(""), IsTTY: func() bool { return false },
	})
	if err == nil {
		t.Errorf("expected error, got nil")
	}
	if err != nil && !strings.Contains(err.Error(), "--yes") {
		t.Errorf("error message should mention --yes, got: %v", err)
	}
	if exit != 1 {
		t.Errorf("exit = %d, want 1", exit)
	}
	if _, statErr := os.Stat(filepath.Join(moldRoot, "github.com")); statErr != nil {
		t.Errorf("nothing should have been deleted")
	}
}

func TestExecuteCacheClearTTYPromptDecline(t *testing.T) {
	moldRoot, indexRoot := newTempCacheDirs(t)
	var out bytes.Buffer
	exit, err := executeCacheClear(cacheClearOptions{
		MoldRoot: moldRoot, IndexRoot: indexRoot,
		Stdout: &out, Stdin: strings.NewReader("n\n"), IsTTY: func() bool { return true },
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exit != 0 {
		t.Errorf("exit = %d, want 0", exit)
	}
	if _, statErr := os.Stat(filepath.Join(moldRoot, "github.com")); statErr != nil {
		t.Errorf("decline should preserve files")
	}
	if !strings.Contains(out.String(), "Cancelled") {
		t.Errorf("output should say Cancelled, got:\n%s", out.String())
	}
}

func TestExecuteCacheClearTTYPromptAccept(t *testing.T) {
	moldRoot, indexRoot := newTempCacheDirs(t)
	var out bytes.Buffer
	exit, err := executeCacheClear(cacheClearOptions{
		MoldRoot: moldRoot, IndexRoot: indexRoot,
		Stdout: &out, Stdin: strings.NewReader("y\n"), IsTTY: func() bool { return true },
	})
	if err != nil || exit != 0 {
		t.Fatalf("exit=%d err=%v", exit, err)
	}
	if _, statErr := os.Stat(filepath.Join(moldRoot, "github.com")); !os.IsNotExist(statErr) {
		t.Errorf("accept should delete files")
	}
}

func mustMkdirAll(t *testing.T, p string) {
	t.Helper()
	if err := os.MkdirAll(p, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", p, err)
	}
}

func mustWriteFile(t *testing.T, p string, data []byte) {
	t.Helper()
	if err := os.WriteFile(p, data, 0o644); err != nil {
		t.Fatalf("WriteFile(%s): %v", p, err)
	}
}
