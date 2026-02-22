package foundry

import (
	"archive/tar"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// makeTarball creates an in-memory tar archive from a map of path → content.
func makeTarball(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	for name, content := range files {
		hdr := &tar.Header{
			Name: name,
			Mode: 0644,
			Size: int64(len(content)),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestExtractTar(t *testing.T) {
	dest := t.TempDir()
	tarData := makeTarball(t, map[string]string{
		"mold.yaml":       "name: test",
		"commands/foo.md": "hello",
	})

	if err := extractTar(tarData, dest); err != nil {
		t.Fatalf("extractTar: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dest, "mold.yaml"))
	if err != nil {
		t.Fatalf("expected mold.yaml: %v", err)
	}
	if string(content) != "name: test" {
		t.Errorf("content = %q, want %q", content, "name: test")
	}

	content, err = os.ReadFile(filepath.Join(dest, "commands", "foo.md"))
	if err != nil {
		t.Fatalf("expected commands/foo.md: %v", err)
	}
	if string(content) != "hello" {
		t.Errorf("content = %q, want %q", content, "hello")
	}
}

func TestFetcher_Fetch(t *testing.T) {
	cacheDir := t.TempDir()

	tarData := makeTarball(t, map[string]string{
		"mold.yaml":     "name: test-mold\nversion: 1.0.0",
		"commands/a.md": "command a",
		"flux.yaml":     "key: value",
	})

	// Mock git runner that simulates clone, fetch, and archive.
	bareDir := BareCloneDir(cacheDir, &Reference{Host: "github.com", Owner: "owner", Repo: "repo"})

	git := func(args ...string) ([]byte, error) {
		key := fmt.Sprintf("%v", args)

		// clone --bare
		if len(args) >= 3 && args[0] == "clone" && args[1] == "--bare" {
			// Create fake bare clone structure.
			if err := os.MkdirAll(args[3], 0750); err != nil {
				return nil, err
			}
			if err := os.WriteFile(filepath.Join(args[3], "HEAD"), []byte("ref: refs/heads/main"), 0644); err != nil {
				return nil, err
			}
			return []byte("Cloning into bare repository..."), nil
		}

		// fetch --all
		if len(args) >= 3 && args[1] == bareDir && args[2] == "fetch" {
			return []byte(""), nil
		}

		// archive
		if len(args) >= 4 && args[2] == "archive" {
			return tarData, nil
		}

		return nil, fmt.Errorf("unexpected git call: %s", key)
	}

	fetcher := NewFetcherWithCacheDir(git, cacheDir)
	ref := &Reference{Host: "github.com", Owner: "owner", Repo: "repo"}
	resolved := &ResolvedVersion{Tag: "v1.0.0", Commit: "abc123"}

	fsys, err := fetcher.Fetch(ref, resolved)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}

	// Read mold.yaml from the returned fs.FS.
	data, err := fsys.Open("mold.yaml")
	if err != nil {
		t.Fatalf("opening mold.yaml: %v", err)
	}
	defer func() { _ = data.Close() }()

	buf := make([]byte, 1024)
	n, _ := data.Read(buf)
	if got := string(buf[:n]); got != "name: test-mold\nversion: 1.0.0" {
		t.Errorf("mold.yaml content = %q", got)
	}
}

func TestFetcher_Fetch_Subpath(t *testing.T) {
	cacheDir := t.TempDir()

	tarData := makeTarball(t, map[string]string{
		"molds/claude/mold.yaml":     "name: claude-mold",
		"molds/claude/commands/a.md": "command a",
		"README.md":                  "root readme",
	})

	bareDir := BareCloneDir(cacheDir, &Reference{Host: "github.com", Owner: "owner", Repo: "repo"})

	git := func(args ...string) ([]byte, error) {
		if len(args) >= 3 && args[0] == "clone" && args[1] == "--bare" {
			if err := os.MkdirAll(args[3], 0750); err != nil {
				return nil, err
			}
			if err := os.WriteFile(filepath.Join(args[3], "HEAD"), []byte("ref: refs/heads/main"), 0644); err != nil {
				return nil, err
			}
			return []byte("Cloning..."), nil
		}
		if len(args) >= 3 && args[1] == bareDir && args[2] == "fetch" {
			return []byte(""), nil
		}
		if len(args) >= 4 && args[2] == "archive" {
			return tarData, nil
		}
		return nil, fmt.Errorf("unexpected git call: %v", args)
	}

	fetcher := NewFetcherWithCacheDir(git, cacheDir)
	ref := &Reference{Host: "github.com", Owner: "owner", Repo: "repo", Subpath: "molds/claude"}
	resolved := &ResolvedVersion{Tag: "v1.0.0", Commit: "abc123"}

	fsys, err := fetcher.Fetch(ref, resolved)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}

	// Should be rooted at the subpath.
	data, err := fsys.Open("mold.yaml")
	if err != nil {
		t.Fatalf("opening mold.yaml at subpath: %v", err)
	}
	defer func() { _ = data.Close() }()

	buf := make([]byte, 1024)
	n, _ := data.Read(buf)
	if got := string(buf[:n]); got != "name: claude-mold" {
		t.Errorf("mold.yaml content = %q", got)
	}
}

func TestFetcher_Fetch_CacheHit(t *testing.T) {
	cacheDir := t.TempDir()
	ref := &Reference{Host: "github.com", Owner: "owner", Repo: "repo"}
	resolved := &ResolvedVersion{Tag: "v1.0.0", Commit: "abc123"}

	// Pre-populate cache.
	vDir := VersionDir(cacheDir, ref, "v1.0.0")
	if err := os.MkdirAll(vDir, 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(vDir, "mold.yaml"), []byte("cached: true"), 0644); err != nil {
		t.Fatal(err)
	}

	// Also need bare clone dir with HEAD for ensureBareClone to succeed.
	bareDir := BareCloneDir(cacheDir, ref)
	if err := os.MkdirAll(bareDir, 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bareDir, "HEAD"), []byte("ref: refs/heads/main"), 0644); err != nil {
		t.Fatal(err)
	}

	fetchCalled := false
	git := func(args ...string) ([]byte, error) {
		// Allow fetch --all for existing bare clone.
		if len(args) >= 3 && args[2] == "fetch" {
			fetchCalled = true
			return []byte(""), nil
		}
		// archive should NOT be called for cache hit.
		if len(args) >= 4 && args[2] == "archive" {
			t.Error("git archive should not be called for cache hit")
		}
		return nil, fmt.Errorf("unexpected: %v", args)
	}

	fetcher := NewFetcherWithCacheDir(git, cacheDir)
	fsys, err := fetcher.Fetch(ref, resolved)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}

	if !fetchCalled {
		t.Error("expected bare clone fetch to still be called")
	}

	// Read cached content.
	data, err := fsys.Open("mold.yaml")
	if err != nil {
		t.Fatalf("opening mold.yaml: %v", err)
	}
	defer func() { _ = data.Close() }()

	buf := make([]byte, 1024)
	n, _ := data.Read(buf)
	if got := string(buf[:n]); got != "cached: true" {
		t.Errorf("expected cached content, got %q", got)
	}
}
