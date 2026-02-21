package providers

import (
	"context"
	"os"
	"testing"
)

func TestNewClaudeProvider_WithKey(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "test-key-123")

	p := NewClaudeProvider()
	if p == nil {
		t.Fatal("expected non-nil provider")
	}
	if !p.IsEnabled() {
		t.Error("expected provider to be enabled when API key is set")
	}
}

func TestNewClaudeProvider_WithoutKey(t *testing.T) {
	// Unset the key for this test
	origKey := os.Getenv("ANTHROPIC_API_KEY")
	t.Setenv("ANTHROPIC_API_KEY", "")
	defer func() {
		if origKey != "" {
			_ = os.Setenv("ANTHROPIC_API_KEY", origKey)
		}
	}()

	p := NewClaudeProvider()
	if p == nil {
		t.Fatal("expected non-nil provider")
	}
	if p.IsEnabled() {
		t.Error("expected provider to be disabled when API key is not set")
	}
}

func TestClaudeProvider_Name(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "test-key")
	p := NewClaudeProvider()
	if p.Name() != "claude" {
		t.Errorf("expected name 'claude', got '%s'", p.Name())
	}
}

func TestClaudeProvider_ValidateConfig_WithKey(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "test-key")
	p := NewClaudeProvider()

	err := p.ValidateConfig()
	if err != nil {
		t.Errorf("unexpected error with valid key: %v", err)
	}
}

func TestClaudeProvider_ValidateConfig_WithoutKey(t *testing.T) {
	origKey := os.Getenv("ANTHROPIC_API_KEY")
	t.Setenv("ANTHROPIC_API_KEY", "")
	defer func() {
		if origKey != "" {
			_ = os.Setenv("ANTHROPIC_API_KEY", origKey)
		}
	}()

	p := NewClaudeProvider()

	err := p.ValidateConfig()
	if err == nil {
		t.Error("expected error for missing API key")
	}
}

func TestClaudeProvider_ExecuteBlank_Enabled(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "test-key")
	p := NewClaudeProvider()

	blank := Blank{
		Name:    "test-template",
		Content: "test content",
	}

	resp, err := p.ExecuteBlank(context.Background(), blank, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if !resp.Success {
		t.Error("expected success to be true")
	}
	if resp.Provider != "claude" {
		t.Errorf("expected provider 'claude', got '%s'", resp.Provider)
	}
	if resp.Blank != "test-template" {
		t.Errorf("expected template 'test-template', got '%s'", resp.Blank)
	}
}

func TestClaudeProvider_ExecuteBlank_Disabled(t *testing.T) {
	origKey := os.Getenv("ANTHROPIC_API_KEY")
	t.Setenv("ANTHROPIC_API_KEY", "")
	defer func() {
		if origKey != "" {
			_ = os.Setenv("ANTHROPIC_API_KEY", origKey)
		}
	}()

	p := NewClaudeProvider()

	blank := Blank{
		Name:    "test-template",
		Content: "test content",
	}

	_, err := p.ExecuteBlank(context.Background(), blank, nil)
	if err == nil {
		t.Error("expected error when provider is disabled")
	}
}

func TestClaudeProvider_ImplementsInterface(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "test-key")
	var _ Provider = NewClaudeProvider()
}
