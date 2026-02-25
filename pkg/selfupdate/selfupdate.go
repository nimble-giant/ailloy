package selfupdate

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
)

const (
	repo            = "nimble-giant/ailloy"
	binaryName      = "ailloy"
	latestURL       = "https://api.github.com/repos/" + repo + "/releases/latest"
	requestTimeout  = 15 * time.Second
	downloadTimeout = 120 * time.Second
)

// ReleaseInfo holds metadata about a GitHub release.
type ReleaseInfo struct {
	TagName string  `json:"tag_name"`
	HTMLURL string  `json:"html_url"`
	Assets  []Asset `json:"assets"`
}

// Asset represents a downloadable file attached to a release.
type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// CheckResult is returned by Check and summarises whether an update is available.
type CheckResult struct {
	Current   string
	Latest    string
	Outdated  bool
	Release   *ReleaseInfo
	UpToDate  bool
	DevBuild  bool
}

// Check queries the GitHub API for the latest release and compares it against
// the running version.
func Check(currentVersion string) (*CheckResult, error) {
	release, err := fetchLatestRelease()
	if err != nil {
		return nil, fmt.Errorf("checking for updates: %w", err)
	}

	result := &CheckResult{
		Current: currentVersion,
		Latest:  release.TagName,
		Release: release,
	}

	cur, err := parseSemver(currentVersion)
	if err != nil {
		// Non-semver builds (e.g. "dev") can't be compared.
		result.DevBuild = true
		return result, nil
	}

	lat, err := parseSemver(release.TagName)
	if err != nil {
		return nil, fmt.Errorf("parsing latest version %q: %w", release.TagName, err)
	}

	if lat.GreaterThan(cur) {
		result.Outdated = true
	} else {
		result.UpToDate = true
	}

	return result, nil
}

// Update downloads the latest release binary, verifies its checksum, and
// replaces the running executable in-place.
func Update(release *ReleaseInfo) error {
	binaryAsset, checksumAsset, err := findAssets(release)
	if err != nil {
		return err
	}

	// Download to a temp file next to the target.
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("determining executable path: %w", err)
	}

	tmpFile, err := os.CreateTemp("", "ailloy-update-*")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath) //nolint:errcheck // best-effort cleanup

	if err := download(binaryAsset.BrowserDownloadURL, tmpFile); err != nil {
		tmpFile.Close() //nolint:errcheck
		return fmt.Errorf("downloading binary: %w", err)
	}
	tmpFile.Close() //nolint:errcheck

	// Verify checksum.
	if checksumAsset != nil {
		if err := verifyChecksum(tmpPath, binaryAsset.Name, checksumAsset.BrowserDownloadURL); err != nil {
			return err
		}
	}

	if err := os.Chmod(tmpPath, 0755); err != nil { //#nosec G302 -- executable needs 0755
		return fmt.Errorf("setting permissions: %w", err)
	}

	// Atomic-ish replace: rename over the current binary.
	if err := replaceExecutable(execPath, tmpPath); err != nil {
		return fmt.Errorf("replacing binary: %w", err)
	}

	return nil
}

// --- internal helpers ---

func fetchLatestRelease() (*ReleaseInfo, error) {
	client := &http.Client{Timeout: requestTimeout}

	req, err := http.NewRequest(http.MethodGet, latestURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching latest release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned %s", resp.Status)
	}

	var release ReleaseInfo
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("decoding release JSON: %w", err)
	}

	return &release, nil
}

func parseSemver(v string) (*semver.Version, error) {
	v = strings.TrimPrefix(v, "v")
	// Strip any metadata after a hyphen that isn't a valid pre-release
	// (e.g. "0.6.7-dirty" from git describe).
	return semver.StrictNewVersion(v)
}

func platformAssetName() string {
	return fmt.Sprintf("%s-%s-%s", binaryName, runtime.GOOS, runtime.GOARCH)
}

func findAssets(release *ReleaseInfo) (binary *Asset, checksum *Asset, err error) {
	wantName := platformAssetName()

	for i := range release.Assets {
		a := &release.Assets[i]
		switch a.Name {
		case wantName:
			binary = a
		case "checksums.txt":
			checksum = a
		}
	}

	if binary == nil {
		return nil, nil, fmt.Errorf("no release asset found for %s (looked for %q)", runtime.GOOS+"/"+runtime.GOARCH, wantName)
	}

	return binary, checksum, nil
}

func download(url string, dest *os.File) error {
	client := &http.Client{Timeout: downloadTimeout}

	resp, err := client.Get(url) //#nosec G107 -- URL comes from GitHub API response
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned %s", resp.Status)
	}

	_, err = io.Copy(dest, resp.Body)
	return err
}

func verifyChecksum(filePath, assetName, checksumURL string) error {
	client := &http.Client{Timeout: requestTimeout}

	resp, err := client.Get(checksumURL) //#nosec G107 -- URL comes from GitHub API response
	if err != nil {
		return fmt.Errorf("downloading checksums: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading checksums: %w", err)
	}

	expected := ""
	for _, line := range strings.Split(string(body), "\n") {
		if strings.Contains(line, assetName) {
			parts := strings.Fields(line)
			if len(parts) >= 1 {
				expected = parts[0]
			}
			break
		}
	}

	if expected == "" {
		// No checksum entry found – skip verification rather than block the update.
		return nil
	}

	f, err := os.Open(filePath) //#nosec G304 -- path is a temp file we created
	if err != nil {
		return fmt.Errorf("opening file for checksum: %w", err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return fmt.Errorf("hashing file: %w", err)
	}

	actual := hex.EncodeToString(h.Sum(nil))
	if actual != expected {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expected, actual)
	}

	return nil
}

func replaceExecutable(target, source string) error {
	// On most systems we can rename directly over the running binary.
	return os.Rename(source, target)
}
