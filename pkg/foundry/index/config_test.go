package index

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfigFrom_NewFormat(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	content := `foundries:
  - name: test-foundry
    url: https://github.com/test/foundry-index
    type: git
    status: ok
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfigFrom(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cfg.Foundries) != 1 {
		t.Fatalf("len(Foundries) = %d, want 1", len(cfg.Foundries))
	}
	if cfg.Foundries[0].Name != "test-foundry" {
		t.Errorf("Name = %q, want %q", cfg.Foundries[0].Name, "test-foundry")
	}
	if cfg.Foundries[0].URL != "https://github.com/test/foundry-index" {
		t.Errorf("URL = %q, want %q", cfg.Foundries[0].URL, "https://github.com/test/foundry-index")
	}
	if cfg.Foundries[0].Type != "git" {
		t.Errorf("Type = %q, want %q", cfg.Foundries[0].Type, "git")
	}
}

func TestLoadConfigFrom_LegacyFormat(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	content := `foundries:
  - https://github.com/test/foundry-index
  - https://example.com/foundry.yaml
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfigFrom(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cfg.Foundries) != 2 {
		t.Fatalf("len(Foundries) = %d, want 2", len(cfg.Foundries))
	}

	// First entry: git type
	if cfg.Foundries[0].URL != "https://github.com/test/foundry-index" {
		t.Errorf("URL = %q", cfg.Foundries[0].URL)
	}
	if cfg.Foundries[0].Type != "git" {
		t.Errorf("Type = %q, want git", cfg.Foundries[0].Type)
	}
	if cfg.Foundries[0].Name != "foundry-index" {
		t.Errorf("Name = %q, want foundry-index", cfg.Foundries[0].Name)
	}

	// Second entry: url type (ends in .yaml)
	if cfg.Foundries[1].Type != "url" {
		t.Errorf("Type = %q, want url", cfg.Foundries[1].Type)
	}
	if cfg.Foundries[1].Name != "foundry" {
		t.Errorf("Name = %q, want foundry", cfg.Foundries[1].Name)
	}
}

func TestLoadConfigFrom_NotFound(t *testing.T) {
	cfg, err := LoadConfigFrom("/nonexistent/config.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Foundries) != 0 {
		t.Errorf("expected empty config, got %d foundries", len(cfg.Foundries))
	}
}

func TestSaveConfigTo_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	cfg := &Config{
		Foundries: []FoundryEntry{
			{Name: "test", URL: "https://github.com/test/index", Type: "git", Status: "ok"},
		},
	}

	if err := SaveConfigTo(cfg, path); err != nil {
		t.Fatalf("save error: %v", err)
	}

	loaded, err := LoadConfigFrom(path)
	if err != nil {
		t.Fatalf("load error: %v", err)
	}

	if len(loaded.Foundries) != 1 {
		t.Fatalf("len(Foundries) = %d, want 1", len(loaded.Foundries))
	}
	if loaded.Foundries[0].Name != "test" {
		t.Errorf("Name = %q, want test", loaded.Foundries[0].Name)
	}
	if loaded.Foundries[0].URL != "https://github.com/test/index" {
		t.Errorf("URL = %q", loaded.Foundries[0].URL)
	}
}

func TestAddFoundry(t *testing.T) {
	cfg := &Config{}

	added := cfg.AddFoundry(FoundryEntry{Name: "test", URL: "https://example.com/index"})
	if !added {
		t.Error("expected AddFoundry to return true")
	}
	if len(cfg.Foundries) != 1 {
		t.Fatalf("len = %d, want 1", len(cfg.Foundries))
	}

	// Duplicate
	added = cfg.AddFoundry(FoundryEntry{Name: "test2", URL: "https://example.com/index"})
	if added {
		t.Error("expected AddFoundry to return false for duplicate URL")
	}
	if len(cfg.Foundries) != 1 {
		t.Fatalf("len = %d, want 1", len(cfg.Foundries))
	}
}

func TestRemoveFoundry(t *testing.T) {
	cfg := &Config{
		Foundries: []FoundryEntry{
			{Name: "first", URL: "https://example.com/first"},
			{Name: "second", URL: "https://example.com/second"},
		},
	}

	// Remove by name
	removed := cfg.RemoveFoundry("first")
	if !removed {
		t.Error("expected RemoveFoundry to return true")
	}
	if len(cfg.Foundries) != 1 {
		t.Fatalf("len = %d, want 1", len(cfg.Foundries))
	}
	if cfg.Foundries[0].Name != "second" {
		t.Error("wrong entry remaining")
	}

	// Remove by URL
	removed = cfg.RemoveFoundry("https://example.com/second")
	if !removed {
		t.Error("expected RemoveFoundry to return true")
	}
	if len(cfg.Foundries) != 0 {
		t.Fatalf("len = %d, want 0", len(cfg.Foundries))
	}

	// Remove nonexistent
	removed = cfg.RemoveFoundry("nope")
	if removed {
		t.Error("expected RemoveFoundry to return false")
	}
}

func TestFindFoundry(t *testing.T) {
	cfg := &Config{
		Foundries: []FoundryEntry{
			{Name: "test", URL: "https://example.com/test"},
		},
	}

	// By name
	entry := cfg.FindFoundry("test")
	if entry == nil {
		t.Fatal("expected to find by name")
	}

	// By URL
	entry = cfg.FindFoundry("https://example.com/test")
	if entry == nil {
		t.Fatal("expected to find by URL")
	}

	// Not found
	entry = cfg.FindFoundry("nope")
	if entry != nil {
		t.Error("expected nil for nonexistent")
	}
}

func TestEffectiveFoundries_PrependsOfficialWhenAbsent(t *testing.T) {
	cfg := &Config{
		Foundries: []FoundryEntry{
			{Name: "community", URL: "https://github.com/someone/foundry"},
		},
	}

	effective := cfg.EffectiveFoundries()
	if len(effective) != 2 {
		t.Fatalf("len = %d, want 2", len(effective))
	}
	if !IsOfficialFoundry(effective[0].URL) {
		t.Errorf("effective[0] URL = %q, want official", effective[0].URL)
	}
	if effective[1].Name != "community" {
		t.Errorf("effective[1].Name = %q, want community", effective[1].Name)
	}

	// The user's config slice must not be mutated.
	if len(cfg.Foundries) != 1 {
		t.Errorf("cfg.Foundries mutated: len = %d, want 1", len(cfg.Foundries))
	}
}

func TestEffectiveFoundries_NoDuplicateWhenRegistered(t *testing.T) {
	cfg := &Config{
		Foundries: []FoundryEntry{
			{Name: "foundry", URL: OfficialFoundryURL, Type: "git", Status: "ok"},
			{Name: "community", URL: "https://github.com/someone/foundry"},
		},
	}

	effective := cfg.EffectiveFoundries()
	if len(effective) != 2 {
		t.Fatalf("len = %d, want 2", len(effective))
	}
	if effective[0].Status != "ok" {
		t.Errorf("expected persisted official entry to win, got Status %q", effective[0].Status)
	}
}

func TestEffectiveFoundries_EmptyConfig(t *testing.T) {
	cfg := &Config{}
	effective := cfg.EffectiveFoundries()
	if len(effective) != 1 {
		t.Fatalf("len = %d, want 1", len(effective))
	}
	if !IsOfficialFoundry(effective[0].URL) {
		t.Errorf("expected official foundry, got %q", effective[0].URL)
	}
}

func TestFoundryForSource(t *testing.T) {
	cacheDir := t.TempDir()
	t.Setenv("AILLOY_INDEX_CACHE_DIR", cacheDir)

	official := FoundryEntry{Name: "foundry", URL: OfficialFoundryURL, Type: "git"}
	community := FoundryEntry{Name: "community", URL: "https://github.com/x/y", Type: "git"}

	mustWrite := func(path, content string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	mustWrite(CachedIndexPath(cacheDir, &official), `apiVersion: v1
kind: foundry-index
name: foundry
molds:
  - name: nimble-mold
    source: github.com/nimble-giant/nimble-mold
`)
	mustWrite(CachedIndexPath(cacheDir, &community), `apiVersion: v1
kind: foundry-index
name: community
molds:
  - name: their-mold
    source: github.com/x/their-mold
`)

	cfg := &Config{Foundries: []FoundryEntry{community}}

	if got := cfg.FoundryForSource("github.com/nimble-giant/nimble-mold"); got == nil || got.Name != "foundry" {
		t.Errorf("expected official foundry, got %+v", got)
	}
	if got := cfg.FoundryForSource("github.com/x/their-mold"); got == nil || got.Name != "community" {
		t.Errorf("expected community foundry, got %+v", got)
	}
	if got := cfg.FoundryForSource("github.com/unknown/mold"); got != nil {
		t.Errorf("expected nil for unknown source, got %+v", got)
	}
}

func TestHasOfficialFoundry(t *testing.T) {
	cfg := &Config{}
	if cfg.HasOfficialFoundry() {
		t.Error("empty config should not report official foundry")
	}

	cfg.Foundries = append(cfg.Foundries, FoundryEntry{URL: OfficialFoundryURL})
	if !cfg.HasOfficialFoundry() {
		t.Error("config with official URL should report true")
	}
}

func TestDetectType(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{"https://github.com/test/index", "git"},
		{"https://example.com/foundry.yaml", "url"},
		{"https://example.com/foundry.yml", "url"},
		{"https://example.com/FOUNDRY.YAML", "url"},
		{"github.com/test/index", "git"},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			got := DetectType(tt.url)
			if got != tt.want {
				t.Errorf("DetectType(%q) = %q, want %q", tt.url, got, tt.want)
			}
		})
	}
}

func TestNameFromURL(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{"https://github.com/nimble-giant/ailloy-foundry-index", "ailloy-foundry-index"},
		{"https://example.com/foundry.yaml", "foundry"},
		{"https://example.com/my-index.yml", "my-index"},
		{"github.com/test/my-foundry", "my-foundry"},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			got := nameFromURL(tt.url)
			if !strings.Contains(got, tt.want) {
				t.Errorf("nameFromURL(%q) = %q, want to contain %q", tt.url, got, tt.want)
			}
		})
	}
}
