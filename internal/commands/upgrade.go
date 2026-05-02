package commands

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/nimble-giant/ailloy/pkg/styles"
	"github.com/spf13/cobra"
)

const (
	upgradeRepoOwner = "nimble-giant"
	upgradeRepoName  = "ailloy"
)

var (
	upgradeCheck bool
	upgradeForce bool
	upgradePin   string
)

var (
	upgradeReleaseAPIBase = "https://api.github.com"
	upgradeReleaseDLBase  = "https://github.com"
	upgradeHTTPClient     = &http.Client{Timeout: 30 * time.Second}
	upgradeCurrentVersion = ""
)

var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade the ailloy CLI itself to the latest release",
	Long: `Download and install the latest ailloy release in place.

Skips Homebrew installs by default — those should run
'brew upgrade nimble-giant/tap/ailloy' instead. Use --force to override.

Use --version to install or downgrade to a specific release tag.`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          runUpgrade,
}

func init() {
	rootCmd.AddCommand(upgradeCmd)
	upgradeCmd.Flags().BoolVar(&upgradeCheck, "check", false, "print available version without installing")
	upgradeCmd.Flags().BoolVar(&upgradeForce, "force", false, "upgrade even if installed via Homebrew")
	upgradeCmd.Flags().StringVar(&upgradePin, "version", "", "install a specific release tag (e.g. v0.6.19)")
}

func runUpgrade(_ *cobra.Command, _ []string) error {
	current := strings.TrimSpace(upgradeCurrentVersion)
	if current == "" {
		current = "dev"
	}

	target := strings.TrimSpace(upgradePin)
	if target == "" {
		latest, err := fetchLatestTag()
		if err != nil {
			return fmt.Errorf("look up latest release: %w", err)
		}
		target = latest
	}
	if !strings.HasPrefix(target, "v") {
		target = "v" + target
	}

	if upgradePin == "" {
		if cmp, err := compareSemver(current, strings.TrimPrefix(target, "v")); err == nil && cmp == 0 {
			fmt.Println(styles.SuccessStyle.Render("✓ ") + fmt.Sprintf("ailloy is already at %s", target))
			return nil
		}
	}

	if upgradeCheck {
		fmt.Printf("current: %s\n", current)
		fmt.Printf("latest:  %s\n", target)
		return nil
	}

	exePath, err := resolveExecutable()
	if err != nil {
		return fmt.Errorf("locate current executable: %w", err)
	}

	if runtime.GOOS == "windows" {
		fmt.Println(styles.WarningStyle.Render("⚠️  ") +
			"Self-upgrade is not supported on Windows.")
		fmt.Println(styles.SubtleStyle.Render(
			"    Download " + target + " from https://github.com/" +
				upgradeRepoOwner + "/" + upgradeRepoName + "/releases"))
		return errors.New("windows self-upgrade unsupported")
	}

	if isHomebrewPath(exePath) && !upgradeForce {
		fmt.Println(styles.WarningStyle.Render("⚠️  ") +
			"ailloy was installed via Homebrew.")
		fmt.Println(styles.SubtleStyle.Render(
			"    Run: brew upgrade nimble-giant/tap/ailloy"))
		fmt.Println(styles.SubtleStyle.Render(
			"    (or pass --force to swap the binary anyway)"))
		return errors.New("managed by Homebrew")
	}

	if err := installRelease(target, exePath); err != nil {
		return err
	}

	fmt.Println(styles.SuccessStyle.Render("✓ ") + "Upgraded ailloy to " + target)
	if out, err := exec.Command(exePath, "--version").CombinedOutput(); err == nil { // #nosec G204 -- exePath is the resolved path of our own executable
		fmt.Println(strings.TrimSpace(string(out)))
	}
	return nil
}

func resolveExecutable() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		return resolved, nil
	}
	return exe, nil
}

// isHomebrewPath reports whether the given executable path looks like it lives
// under a Homebrew prefix.
func isHomebrewPath(path string) bool {
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	sep := string(os.PathSeparator)
	if strings.Contains(abs, sep+"Cellar"+sep) {
		return true
	}
	prefixes := []string{
		"/opt/homebrew/",
		"/usr/local/Homebrew/",
		"/usr/local/Cellar/",
		"/home/linuxbrew/.linuxbrew/",
	}
	for _, p := range prefixes {
		if strings.HasPrefix(abs, p) {
			return true
		}
	}
	return false
}

func fetchLatestTag() (string, error) {
	url := upgradeReleaseAPIBase + "/repos/" + upgradeRepoOwner + "/" + upgradeRepoName + "/releases/latest"
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := upgradeHTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("github api returned %s", resp.Status)
	}
	var payload struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", err
	}
	if payload.TagName == "" {
		return "", errors.New("github api returned empty tag_name")
	}
	return payload.TagName, nil
}

func compareSemver(a, b string) (int, error) {
	av, err := semver.NewVersion(strings.TrimPrefix(a, "v"))
	if err != nil {
		return 0, err
	}
	bv, err := semver.NewVersion(strings.TrimPrefix(b, "v"))
	if err != nil {
		return 0, err
	}
	return av.Compare(bv), nil
}

// assetName returns the release asset filename for a given platform, matching
// the naming used by .github/workflows/release.yml.
func assetName(goos, goarch string) string {
	if goos == "windows" {
		return fmt.Sprintf("ailloy-%s-%s.exe", goos, goarch)
	}
	return fmt.Sprintf("ailloy-%s-%s", goos, goarch)
}

func installRelease(tag, destPath string) error {
	asset := assetName(runtime.GOOS, runtime.GOARCH)
	releaseBase := fmt.Sprintf("%s/%s/%s/releases/download/%s",
		upgradeReleaseDLBase, upgradeRepoOwner, upgradeRepoName, tag)

	checksums, err := downloadString(releaseBase + "/checksums.txt")
	if err != nil {
		return fmt.Errorf("download checksums.txt: %w", err)
	}
	expected, ok := lookupChecksum(checksums, asset)
	if !ok {
		return fmt.Errorf("no checksum entry for %s in release %s", asset, tag)
	}

	dir := filepath.Dir(destPath)
	tmp, err := os.CreateTemp(dir, ".ailloy-upgrade-*")
	if err != nil {
		return fmt.Errorf("create temp file in %s: %w", dir, err)
	}
	tmpPath := tmp.Name()
	keepTmp := false
	defer func() {
		if !keepTmp {
			_ = os.Remove(tmpPath)
		}
	}()

	resp, err := upgradeHTTPClient.Get(releaseBase + "/" + asset)
	if err != nil {
		_ = tmp.Close()
		return fmt.Errorf("download %s: %w", asset, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		_ = tmp.Close()
		return fmt.Errorf("download %s: %s", asset, resp.Status)
	}

	hasher := sha256.New()
	if _, err := io.Copy(io.MultiWriter(tmp, hasher), resp.Body); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write %s: %w", asset, err)
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	got := hex.EncodeToString(hasher.Sum(nil))
	if !strings.EqualFold(got, expected) {
		return fmt.Errorf("checksum mismatch for %s: got %s, expected %s", asset, got, expected)
	}

	if err := os.Chmod(tmpPath, 0o755); err != nil { // #nosec G302 -- binary must be executable
		return fmt.Errorf("chmod new binary: %w", err)
	}
	if err := os.Rename(tmpPath, destPath); err != nil {
		return fmt.Errorf("replace binary at %s: %w", destPath, err)
	}
	keepTmp = true
	return nil
}

func downloadString(url string) (string, error) {
	resp, err := upgradeHTTPClient.Get(url)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GET %s: %s", url, resp.Status)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

// lookupChecksum returns the SHA256 entry for name in a sha256sum-style file
// (one "<sha>  <name>" per line). Leading "*" markers (binary mode) are
// tolerated.
func lookupChecksum(content, name string) (string, bool) {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		entry := strings.TrimPrefix(parts[len(parts)-1], "*")
		if entry == name {
			return parts[0], true
		}
	}
	return "", false
}
