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
