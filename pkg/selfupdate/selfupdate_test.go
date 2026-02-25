package selfupdate

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"testing"
)

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

func TestPlatformAssetName(t *testing.T) {
	name := platformAssetName()
	want := fmt.Sprintf("ailloy-%s-%s", runtime.GOOS, runtime.GOARCH)
	if name != want {
		t.Errorf("platformAssetName() = %q, want %q", name, want)
	}
}

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
}

func TestCheck_DevBuild(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"tag_name":"v1.0.0","html_url":"https://github.com/test","assets":[]}`)
	}))
	defer srv.Close()

	// Temporarily override the latestURL by using the test server.
	// Since latestURL is a const, we test via the Check function with "dev" version.
	// The dev build detection happens after the API call, so we test parseSemver path.
	_, err := parseSemver("dev")
	if err == nil {
		t.Fatal("expected error parsing 'dev' as semver")
	}
}

func TestVerifyChecksum(t *testing.T) {
	// Create a temp file with known content.
	content := []byte("hello ailloy")
	tmpFile, err := os.CreateTemp("", "ailloy-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write(content); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	// Compute expected checksum.
	h := sha256.Sum256(content)
	expected := hex.EncodeToString(h[:])

	assetName := "ailloy-linux-amd64"
	checksumBody := fmt.Sprintf("%s  %s\nabcdef1234567890  other-file\n", expected, assetName)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, checksumBody)
	}))
	defer srv.Close()

	t.Run("valid checksum passes", func(t *testing.T) {
		err := verifyChecksum(tmpFile.Name(), assetName, srv.URL)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("wrong asset name skips verification", func(t *testing.T) {
		err := verifyChecksum(tmpFile.Name(), "nonexistent-asset", srv.URL)
		if err != nil {
			t.Fatalf("expected nil error for missing checksum entry, got: %v", err)
		}
	})

	t.Run("bad checksum fails", func(t *testing.T) {
		badBody := fmt.Sprintf("%s  %s\n", "0000000000000000000000000000000000000000000000000000000000000000", assetName)
		badSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, badBody)
		}))
		defer badSrv.Close()

		err := verifyChecksum(tmpFile.Name(), assetName, badSrv.URL)
		if err == nil {
			t.Fatal("expected checksum mismatch error")
		}
	})
}

func TestCheckResult_Outdated(t *testing.T) {
	// Test the semver comparison logic directly.
	cur, _ := parseSemver("v0.5.0")
	lat, _ := parseSemver("v0.6.7")

	if !lat.GreaterThan(cur) {
		t.Error("v0.6.7 should be greater than v0.5.0")
	}
}

func TestCheckResult_UpToDate(t *testing.T) {
	cur, _ := parseSemver("v0.6.7")
	lat, _ := parseSemver("v0.6.7")

	if lat.GreaterThan(cur) {
		t.Error("same versions should not be outdated")
	}
}

func TestCheckResult_Ahead(t *testing.T) {
	cur, _ := parseSemver("v1.0.0")
	lat, _ := parseSemver("v0.6.7")

	if lat.GreaterThan(cur) {
		t.Error("current ahead of latest should not be outdated")
	}
}
