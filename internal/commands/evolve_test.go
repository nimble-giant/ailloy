package commands

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/ulikunitz/xz"
)

func TestEvolveAnimationArt(t *testing.T) {
	art := evolveAnimationArt()
	if art == "" {
		t.Fatal("expected non-empty evolution art")
	}
	if strings.HasPrefix(art, "\n") {
		t.Errorf("art should not start with a leading newline; cursor-up math depends on the rendered height matching the line count")
	}
	if lines := strings.Count(art, "\n") + 1; lines < 5 {
		t.Errorf("expected multi-line mascot art (>=5 lines), got %d", lines)
	}
}

func TestIsHomebrewPath(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		{"/opt/homebrew/bin/ailloy", true},
		{"/opt/homebrew/Cellar/ailloy/0.6.19/bin/ailloy", true},
		{"/usr/local/Cellar/ailloy/0.6.19/bin/ailloy", true},
		{"/usr/local/Homebrew/bin/ailloy", true},
		{"/home/linuxbrew/.linuxbrew/bin/ailloy", true},
		{"/usr/local/bin/ailloy", false},
		{"/home/alice/.local/bin/ailloy", false},
		{"/tmp/ailloy", false},
	}
	for _, tc := range cases {
		if got := isHomebrewPath(tc.path); got != tc.want {
			t.Errorf("isHomebrewPath(%q) = %v, want %v", tc.path, got, tc.want)
		}
	}
}

func TestAssetName(t *testing.T) {
	cases := []struct {
		goos, goarch, want string
	}{
		{"linux", "amd64", "ailloy-linux-amd64.xz"},
		{"linux", "arm64", "ailloy-linux-arm64.xz"},
		{"darwin", "amd64", "ailloy-darwin-amd64.xz"},
		{"darwin", "arm64", "ailloy-darwin-arm64.xz"},
		{"windows", "amd64", "ailloy-windows-amd64.exe"},
	}
	for _, tc := range cases {
		if got := assetName(tc.goos, tc.goarch); got != tc.want {
			t.Errorf("assetName(%q, %q) = %q, want %q", tc.goos, tc.goarch, got, tc.want)
		}
	}
}

func TestCompareSemver(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"0.6.19", "0.6.19", 0},
		{"v0.6.19", "0.6.19", 0},
		{"0.6.18", "0.6.19", -1},
		{"v0.7.0", "v0.6.19", 1},
	}
	for _, tc := range cases {
		got, err := compareSemver(tc.a, tc.b)
		if err != nil {
			t.Errorf("compareSemver(%q,%q) error: %v", tc.a, tc.b, err)
			continue
		}
		if got != tc.want {
			t.Errorf("compareSemver(%q,%q) = %d, want %d", tc.a, tc.b, got, tc.want)
		}
	}
	if _, err := compareSemver("not-a-version", "0.6.19"); err == nil {
		t.Errorf("expected error for invalid version")
	}
}

func TestInstallRelease_DecompressesXz(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("evolve does not run on windows; .xz path is non-windows only")
	}

	// Pretend the released binary is this little payload.
	want := []byte("#!/bin/sh\necho hello from ailloy\n")

	var compressed bytes.Buffer
	xw, err := xz.NewWriter(&compressed)
	if err != nil {
		t.Fatalf("xz.NewWriter: %v", err)
	}
	if _, err := xw.Write(want); err != nil {
		t.Fatalf("xz write: %v", err)
	}
	if err := xw.Close(); err != nil {
		t.Fatalf("xz close: %v", err)
	}
	compressedBytes := compressed.Bytes()

	asset := assetName(runtime.GOOS, runtime.GOARCH)
	if !strings.HasSuffix(asset, ".xz") {
		t.Fatalf("expected .xz asset on %s/%s, got %q", runtime.GOOS, runtime.GOARCH, asset)
	}

	sum := sha256.Sum256(compressedBytes)
	checksums := fmt.Sprintf("%s  %s\n", hex.EncodeToString(sum[:]), asset)

	tag := "v9.9.9"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/checksums.txt"):
			_, _ = w.Write([]byte(checksums))
		case strings.HasSuffix(r.URL.Path, "/"+asset):
			_, _ = w.Write(compressedBytes)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	origBase := evolveReleaseDLBase
	evolveReleaseDLBase = srv.URL
	defer func() { evolveReleaseDLBase = origBase }()

	dest := filepath.Join(t.TempDir(), "ailloy")
	if err := os.WriteFile(dest, []byte("old binary"), 0o755); err != nil { // #nosec G306 -- test fixture
		t.Fatalf("seed dest: %v", err)
	}

	if err := installRelease(tag, dest); err != nil {
		t.Fatalf("installRelease: %v", err)
	}

	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("read dest: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("dest content mismatch\n got: %q\nwant: %q", got, want)
	}

	info, err := os.Stat(dest)
	if err != nil {
		t.Fatalf("stat dest: %v", err)
	}
	if info.Mode().Perm()&0o100 == 0 {
		t.Errorf("dest not user-executable: mode=%v", info.Mode())
	}
}

func TestInstallRelease_ChecksumMismatch(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("evolve does not run on windows")
	}

	asset := assetName(runtime.GOOS, runtime.GOARCH)
	tag := "v9.9.9"

	// Wrong checksum on purpose.
	checksums := fmt.Sprintf("%s  %s\n", strings.Repeat("0", 64), asset)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/checksums.txt"):
			_, _ = w.Write([]byte(checksums))
		case strings.HasSuffix(r.URL.Path, "/"+asset):
			// Anything xz-decodable; won't reach the hash check otherwise.
			var b bytes.Buffer
			xw, _ := xz.NewWriter(&b)
			_, _ = xw.Write([]byte("bytes"))
			_ = xw.Close()
			_, _ = w.Write(b.Bytes())
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	origBase := evolveReleaseDLBase
	evolveReleaseDLBase = srv.URL
	defer func() { evolveReleaseDLBase = origBase }()

	dest := filepath.Join(t.TempDir(), "ailloy")
	if err := os.WriteFile(dest, []byte("old"), 0o755); err != nil { // #nosec G306 -- test fixture
		t.Fatalf("seed: %v", err)
	}

	err := installRelease(tag, dest)
	if err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("expected checksum mismatch error, got: %v", err)
	}

	// Original binary must remain — atomic-rename guarantees no half-written swap.
	got, _ := os.ReadFile(dest)
	if string(got) != "old" {
		t.Errorf("dest was clobbered on checksum failure: %q", got)
	}
}

func TestLookupChecksum(t *testing.T) {
	content := `d410762b53f1111790328fb136c95dd77b73e37a795ea00b1754939a872427eb  ailloy-darwin-arm64
e99c75c95f88d37730d509ac48395d1420440cdf9a470a10167c71e92f7746d6 *ailloy-darwin-amd64
8857e704891c1f54531fdb5561f099001a01e1713c1884467fead35a132513e6  ailloy-linux-arm64

`
	cases := []struct {
		name      string
		want      string
		wantFound bool
	}{
		{"ailloy-darwin-arm64", "d410762b53f1111790328fb136c95dd77b73e37a795ea00b1754939a872427eb", true},
		{"ailloy-darwin-amd64", "e99c75c95f88d37730d509ac48395d1420440cdf9a470a10167c71e92f7746d6", true},
		{"ailloy-linux-arm64", "8857e704891c1f54531fdb5561f099001a01e1713c1884467fead35a132513e6", true},
		{"ailloy-linux-amd64", "", false},
	}
	for _, tc := range cases {
		got, ok := lookupChecksum(content, tc.name)
		if ok != tc.wantFound {
			t.Errorf("lookupChecksum(%q) found=%v, want %v", tc.name, ok, tc.wantFound)
		}
		if got != tc.want {
			t.Errorf("lookupChecksum(%q) = %q, want %q", tc.name, got, tc.want)
		}
	}
}
