package selfupdate

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// overrideLatestURL points the package at the given test server and returns a
// cleanup function that restores the original value.
func overrideLatestURL(t *testing.T, url string) {
	t.Helper()
	orig := latestURL
	latestURL = url
	t.Cleanup(func() { latestURL = orig })
}

// --- parseSemver ---

func TestParseSemver(t *testing.T) {
	tests := []struct {
		input   string
		wantErr bool
	}{
		{"v0.6.7", false},
		{"0.6.7", false},
		{"v1.0.0", false},
		{"1.2.3", false},
		{"dev", true},
		{"unknown", true},
		{"", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			_, err := parseSemver(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseSemver(%q): err=%v, wantErr=%v", tt.input, err, tt.wantErr)
			}
		})
	}
}

// --- platformAssetName ---

func TestPlatformAssetName(t *testing.T) {
	name := platformAssetName()
	want := fmt.Sprintf("ailloy-%s-%s", runtime.GOOS, runtime.GOARCH)
	if name != want {
		t.Errorf("platformAssetName() = %q, want %q", name, want)
	}
}

// --- findAssets ---

func TestFindAssets(t *testing.T) {
	wantBinary := platformAssetName()

	t.Run("finds binary and checksum", func(t *testing.T) {
		release := &ReleaseInfo{
			Assets: []Asset{
				{Name: wantBinary, BrowserDownloadURL: "https://example.com/binary"},
				{Name: "checksums.txt", BrowserDownloadURL: "https://example.com/checksums"},
				{Name: "ailloy-windows-amd64.exe", BrowserDownloadURL: "https://example.com/win"},
			},
		}

		binary, checksum, err := findAssets(release)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if binary.Name != wantBinary {
			t.Errorf("binary.Name = %q, want %q", binary.Name, wantBinary)
		}
		if checksum == nil || checksum.Name != "checksums.txt" {
			t.Errorf("checksum not found or wrong name")
		}
	})

	t.Run("missing binary returns error", func(t *testing.T) {
		release := &ReleaseInfo{
			Assets: []Asset{
				{Name: "checksums.txt"},
			},
		}

		_, _, err := findAssets(release)
		if err == nil {
			t.Fatal("expected error for missing binary asset")
		}
	})

	t.Run("missing checksum is ok", func(t *testing.T) {
		release := &ReleaseInfo{
			Assets: []Asset{
				{Name: wantBinary, BrowserDownloadURL: "https://example.com/binary"},
			},
		}

		binary, checksum, err := findAssets(release)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if binary == nil {
			t.Fatal("binary should not be nil")
		}
		if checksum != nil {
			t.Error("checksum should be nil when not present")
		}
	})

	t.Run("empty assets returns error", func(t *testing.T) {
		release := &ReleaseInfo{Assets: []Asset{}}
		_, _, err := findAssets(release)
		if err == nil {
			t.Fatal("expected error for empty assets")
		}
	})
}

// --- Check (integration via httptest) ---

func TestCheck_Outdated(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Accept") != "application/vnd.github+json" {
			t.Errorf("expected Accept header, got %q", r.Header.Get("Accept"))
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"tag_name":"v2.0.0","html_url":"https://github.com/test/release","assets":[]}`)
	}))
	defer srv.Close()
	overrideLatestURL(t, srv.URL)

	result, err := Check("v1.0.0")
	if err != nil {
		t.Fatalf("Check returned error: %v", err)
	}
	if !result.Outdated {
		t.Error("expected Outdated=true")
	}
	if result.UpToDate {
		t.Error("expected UpToDate=false")
	}
	if result.DevBuild {
		t.Error("expected DevBuild=false")
	}
	if result.Current != "v1.0.0" {
		t.Errorf("Current = %q, want %q", result.Current, "v1.0.0")
	}
	if result.Latest != "v2.0.0" {
		t.Errorf("Latest = %q, want %q", result.Latest, "v2.0.0")
	}
}

func TestCheck_UpToDate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"tag_name":"v1.0.0","html_url":"https://github.com/test/release","assets":[]}`)
	}))
	defer srv.Close()
	overrideLatestURL(t, srv.URL)

	result, err := Check("v1.0.0")
	if err != nil {
		t.Fatalf("Check returned error: %v", err)
	}
	if result.Outdated {
		t.Error("expected Outdated=false for same version")
	}
	if !result.UpToDate {
		t.Error("expected UpToDate=true for same version")
	}
}

func TestCheck_Ahead(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"tag_name":"v1.0.0","html_url":"https://github.com/test/release","assets":[]}`)
	}))
	defer srv.Close()
	overrideLatestURL(t, srv.URL)

	result, err := Check("v2.0.0")
	if err != nil {
		t.Fatalf("Check returned error: %v", err)
	}
	if result.Outdated {
		t.Error("expected Outdated=false when current is ahead")
	}
	if !result.UpToDate {
		t.Error("expected UpToDate=true when current is ahead")
	}
}

func TestCheck_DevBuild(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"tag_name":"v1.0.0","html_url":"https://github.com/test/release","assets":[]}`)
	}))
	defer srv.Close()
	overrideLatestURL(t, srv.URL)

	result, err := Check("dev")
	if err != nil {
		t.Fatalf("Check returned error: %v", err)
	}
	if !result.DevBuild {
		t.Error("expected DevBuild=true for 'dev' version")
	}
	if result.Outdated || result.UpToDate {
		t.Error("dev builds should not set Outdated or UpToDate")
	}
}

func TestCheck_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	overrideLatestURL(t, srv.URL)

	_, err := Check("v1.0.0")
	if err == nil {
		t.Fatal("expected error when API returns 500")
	}
}

func TestCheck_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{not json}`)
	}))
	defer srv.Close()
	overrideLatestURL(t, srv.URL)

	_, err := Check("v1.0.0")
	if err == nil {
		t.Fatal("expected error when API returns invalid JSON")
	}
}

func TestCheck_InvalidLatestTag(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"tag_name":"not-a-version","html_url":"https://github.com/test","assets":[]}`)
	}))
	defer srv.Close()
	overrideLatestURL(t, srv.URL)

	_, err := Check("v1.0.0")
	if err == nil {
		t.Fatal("expected error when latest tag is not valid semver")
	}
}

// --- download ---

func TestDownload_Success(t *testing.T) {
	content := "binary-data-here"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, content)
	}))
	defer srv.Close()

	tmpFile, err := os.CreateTemp(t.TempDir(), "dl-test-*")
	if err != nil {
		t.Fatal(err)
	}

	if err := download(srv.URL, tmpFile); err != nil {
		t.Fatalf("download returned error: %v", err)
	}
	tmpFile.Close()

	got, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != content {
		t.Errorf("downloaded content = %q, want %q", string(got), content)
	}
}

func TestDownload_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	tmpFile, err := os.CreateTemp(t.TempDir(), "dl-err-*")
	if err != nil {
		t.Fatal(err)
	}
	defer tmpFile.Close()

	err = download(srv.URL, tmpFile)
	if err == nil {
		t.Fatal("expected error for 404 response")
	}
}

// --- verifyChecksum ---

func TestVerifyChecksum(t *testing.T) {
	content := []byte("hello ailloy")
	tmpFile, err := os.CreateTemp(t.TempDir(), "ailloy-test-*")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := tmpFile.Write(content); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	h := sha256.Sum256(content)
	expected := hex.EncodeToString(h[:])

	assetName := "ailloy-linux-amd64"
	checksumBody := fmt.Sprintf("%s  %s\nabcdef1234567890  other-file\n", expected, assetName)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, checksumBody)
	}))
	defer srv.Close()

	t.Run("valid checksum passes", func(t *testing.T) {
		if err := verifyChecksum(tmpFile.Name(), assetName, srv.URL); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("wrong asset name skips verification", func(t *testing.T) {
		if err := verifyChecksum(tmpFile.Name(), "nonexistent-asset", srv.URL); err != nil {
			t.Fatalf("expected nil error for missing checksum entry, got: %v", err)
		}
	})

	t.Run("bad checksum fails", func(t *testing.T) {
		badBody := fmt.Sprintf("%s  %s\n", "0000000000000000000000000000000000000000000000000000000000000000", assetName)
		badSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			fmt.Fprint(w, badBody)
		}))
		defer badSrv.Close()

		if err := verifyChecksum(tmpFile.Name(), assetName, badSrv.URL); err == nil {
			t.Fatal("expected checksum mismatch error")
		}
	})

	t.Run("checksum server error", func(t *testing.T) {
		errSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer errSrv.Close()

		// ReadAll still succeeds (it reads the empty body), and there is no
		// matching entry, so verification is skipped.
		if err := verifyChecksum(tmpFile.Name(), assetName, errSrv.URL); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

// --- replaceExecutable ---

func TestReplaceExecutable(t *testing.T) {
	dir := t.TempDir()

	target := filepath.Join(dir, "old-binary")
	if err := os.WriteFile(target, []byte("old"), 0755); err != nil {
		t.Fatal(err)
	}

	source := filepath.Join(dir, "new-binary")
	if err := os.WriteFile(source, []byte("new"), 0755); err != nil {
		t.Fatal(err)
	}

	if err := replaceExecutable(target, source); err != nil {
		t.Fatalf("replaceExecutable returned error: %v", err)
	}

	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "new" {
		t.Errorf("target content = %q, want %q", string(got), "new")
	}

	// Source should no longer exist (it was renamed).
	if _, err := os.Stat(source); !os.IsNotExist(err) {
		t.Error("source file should be removed after rename")
	}
}

func TestReplaceExecutable_MissingSource(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	if err := os.WriteFile(target, []byte("keep"), 0755); err != nil {
		t.Fatal(err)
	}

	err := replaceExecutable(target, filepath.Join(dir, "nonexistent"))
	if err == nil {
		t.Fatal("expected error when source does not exist")
	}
}

// --- semver comparison helpers ---

func TestSemverComparison_Outdated(t *testing.T) {
	cur, _ := parseSemver("v0.5.0")
	lat, _ := parseSemver("v0.6.7")
	if !lat.GreaterThan(cur) {
		t.Error("v0.6.7 should be greater than v0.5.0")
	}
}

func TestSemverComparison_UpToDate(t *testing.T) {
	cur, _ := parseSemver("v0.6.7")
	lat, _ := parseSemver("v0.6.7")
	if lat.GreaterThan(cur) {
		t.Error("same versions should not be outdated")
	}
}

func TestSemverComparison_Ahead(t *testing.T) {
	cur, _ := parseSemver("v1.0.0")
	lat, _ := parseSemver("v0.6.7")
	if lat.GreaterThan(cur) {
		t.Error("current ahead of latest should not be outdated")
	}
}

func TestSemverComparison_MajorMinorPatch(t *testing.T) {
	tests := []struct {
		current string
		latest  string
		wantGt  bool
	}{
		{"v1.0.0", "v1.0.1", true},  // patch bump
		{"v1.0.0", "v1.1.0", true},  // minor bump
		{"v1.0.0", "v2.0.0", true},  // major bump
		{"v1.1.0", "v1.0.9", false}, // minor > patch
		{"v2.0.0", "v1.9.9", false}, // major wins
	}

	for _, tt := range tests {
		t.Run(tt.current+"_vs_"+tt.latest, func(t *testing.T) {
			cur, _ := parseSemver(tt.current)
			lat, _ := parseSemver(tt.latest)
			got := lat.GreaterThan(cur)
			if got != tt.wantGt {
				t.Errorf("(%s).GreaterThan(%s) = %v, want %v", tt.latest, tt.current, got, tt.wantGt)
			}
		})
	}
}
