package commands

import (
	"os"
	"path/filepath"
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
