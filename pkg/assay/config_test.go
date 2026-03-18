package assay

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig_NoFile(t *testing.T) {
	dir := t.TempDir()
	cfg, err := LoadConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg == nil {
		t.Fatal("expected default config, got nil")
	}
	// All rules should be enabled by default
	if !cfg.IsRuleEnabled("line-count") {
		t.Error("expected line-count to be enabled by default")
	}
}

func TestLoadConfig_ValidFile(t *testing.T) {
	dir := t.TempDir()
	content := `
assay:
  rules:
    line-count:
      enabled: true
      options:
        max-lines: 200
    structure:
      enabled: false
  ignore:
    - "vendor/**"
  platforms:
    - claude
`
	if err := os.WriteFile(filepath.Join(dir, ".ailloyrc.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(dir)
	if err != nil {
		t.Fatal(err)
	}

	if !cfg.IsRuleEnabled("line-count") {
		t.Error("expected line-count enabled")
	}
	if cfg.IsRuleEnabled("structure") {
		t.Error("expected structure disabled")
	}
	if cfg.RuleOption("line-count", "max-lines", nil) == nil {
		t.Error("expected max-lines option")
	}
	if len(cfg.Ignore) != 1 || cfg.Ignore[0] != "vendor/**" {
		t.Errorf("expected ignore pattern, got %v", cfg.Ignore)
	}
	if len(cfg.Platforms) != 1 || cfg.Platforms[0] != "claude" {
		t.Errorf("expected platforms [claude], got %v", cfg.Platforms)
	}
}

func TestLoadConfig_YmlExtension(t *testing.T) {
	dir := t.TempDir()
	content := `
assay:
  rules:
    empty-file:
      enabled: false
`
	if err := os.WriteFile(filepath.Join(dir, ".ailloyrc.yml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.IsRuleEnabled("empty-file") {
		t.Error("expected empty-file disabled")
	}
}

func TestIsRuleEnabled_NilConfig(t *testing.T) {
	var cfg *Config
	if !cfg.IsRuleEnabled("anything") {
		t.Error("nil config should default to enabled")
	}
}

func TestRuleOption_Fallback(t *testing.T) {
	cfg := DefaultConfig()
	val := cfg.RuleOption("line-count", "max-lines", 150)
	if val != 150 {
		t.Errorf("expected fallback 150, got %v", val)
	}
}

func TestGenerateStarterConfig(t *testing.T) {
	content := GenerateStarterConfig()
	if content == "" {
		t.Error("expected non-empty starter config")
	}
	if !contains(content, "assay:") {
		t.Error("expected 'assay:' section in starter config")
	}
	if !contains(content, "plugin-manifest:") {
		t.Error("expected 'plugin-manifest' rule in starter config")
	}
	if !contains(content, "plugin-hooks:") {
		t.Error("expected 'plugin-hooks' rule in starter config")
	}
}

func TestAddAllowedFrontmatterFields(t *testing.T) {
	t.Run("creates config and adds fields", func(t *testing.T) {
		dir := t.TempDir()
		added, err := AddAllowedFrontmatterFields(dir, []string{"topic", "source", "tags"})
		if err != nil {
			t.Fatal(err)
		}
		if len(added) != 3 {
			t.Errorf("expected 3 added fields, got %d: %v", len(added), added)
		}

		// Verify the config file was written and is readable.
		cfg, err := LoadConfig(dir)
		if err != nil {
			t.Fatal(err)
		}
		extras, ok := cfg.RuleOption("command-frontmatter", "extra-allowed-fields", nil).([]any)
		if !ok || len(extras) != 3 {
			t.Errorf("expected 3 extra-allowed-fields in saved config, got %v", extras)
		}
	})

	t.Run("deduplicates fields already present", func(t *testing.T) {
		dir := t.TempDir()
		// First call
		if _, err := AddAllowedFrontmatterFields(dir, []string{"topic", "source"}); err != nil {
			t.Fatal(err)
		}
		// Second call adds one new + one duplicate
		added, err := AddAllowedFrontmatterFields(dir, []string{"source", "tags"})
		if err != nil {
			t.Fatal(err)
		}
		if len(added) != 1 || added[0] != "tags" {
			t.Errorf("expected only 'tags' to be newly added, got %v", added)
		}

		cfg, err := LoadConfig(dir)
		if err != nil {
			t.Fatal(err)
		}
		extras, _ := cfg.RuleOption("command-frontmatter", "extra-allowed-fields", nil).([]any)
		if len(extras) != 3 {
			t.Errorf("expected 3 total fields after merge, got %d: %v", len(extras), extras)
		}
	})

	t.Run("returns nil added when all fields already present", func(t *testing.T) {
		dir := t.TempDir()
		if _, err := AddAllowedFrontmatterFields(dir, []string{"topic"}); err != nil {
			t.Fatal(err)
		}
		added, err := AddAllowedFrontmatterFields(dir, []string{"topic"})
		if err != nil {
			t.Fatal(err)
		}
		if len(added) != 0 {
			t.Errorf("expected no new fields, got %v", added)
		}
	})

	t.Run("fields are sorted in written config", func(t *testing.T) {
		dir := t.TempDir()
		if _, err := AddAllowedFrontmatterFields(dir, []string{"zzz", "aaa", "mmm"}); err != nil {
			t.Fatal(err)
		}
		cfg, err := LoadConfig(dir)
		if err != nil {
			t.Fatal(err)
		}
		extras, _ := cfg.RuleOption("command-frontmatter", "extra-allowed-fields", nil).([]any)
		if len(extras) < 3 {
			t.Fatal("expected 3 fields")
		}
		if extras[0].(string) != "aaa" || extras[1].(string) != "mmm" || extras[2].(string) != "zzz" {
			t.Errorf("expected sorted fields [aaa mmm zzz], got %v", extras)
		}
	})
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstr(s, substr))
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
