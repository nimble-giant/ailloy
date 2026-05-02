package commands

import (
	"strings"
	"testing"
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
		{"linux", "amd64", "ailloy-linux-amd64"},
		{"linux", "arm64", "ailloy-linux-arm64"},
		{"darwin", "amd64", "ailloy-darwin-amd64"},
		{"darwin", "arm64", "ailloy-darwin-arm64"},
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
