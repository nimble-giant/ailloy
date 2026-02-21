package providers

import (
	"context"
	"sort"
	"testing"
)

// mockProvider implements Provider for testing
type mockProvider struct {
	name    string
	enabled bool
}

func (m *mockProvider) Name() string { return m.name }
func (m *mockProvider) ExecuteBlank(_ context.Context, _ Blank, _ map[string]interface{}) (*Response, error) {
	return &Response{
		Content:  "mock response",
		Provider: m.name,
		Success:  true,
	}, nil
}
func (m *mockProvider) ValidateConfig() error { return nil }
func (m *mockProvider) IsEnabled() bool       { return m.enabled }

func TestNewRegistry(t *testing.T) {
	r := NewRegistry()
	if r == nil {
		t.Fatal("expected non-nil registry")
	}
	if len(r.providers) != 0 {
		t.Errorf("expected empty registry, got %d providers", len(r.providers))
	}
}

func TestRegistry_Register(t *testing.T) {
	r := NewRegistry()
	p := &mockProvider{name: "test", enabled: true}
	r.Register(p)

	if len(r.providers) != 1 {
		t.Errorf("expected 1 provider, got %d", len(r.providers))
	}
}

func TestRegistry_Get_Exists(t *testing.T) {
	r := NewRegistry()
	p := &mockProvider{name: "test", enabled: true}
	r.Register(p)

	got, err := r.Get("test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Name() != "test" {
		t.Errorf("expected provider name 'test', got '%s'", got.Name())
	}
}

func TestRegistry_Get_NotFound(t *testing.T) {
	r := NewRegistry()
	_, err := r.Get("nonexistent")
	if err == nil {
		t.Error("expected error for non-existent provider")
	}
}

func TestRegistry_List(t *testing.T) {
	r := NewRegistry()
	r.Register(&mockProvider{name: "alpha", enabled: true})
	r.Register(&mockProvider{name: "beta", enabled: false})

	names := r.List()
	if len(names) != 2 {
		t.Fatalf("expected 2 names, got %d", len(names))
	}

	sort.Strings(names)
	if names[0] != "alpha" || names[1] != "beta" {
		t.Errorf("expected [alpha, beta], got %v", names)
	}
}

func TestRegistry_List_Empty(t *testing.T) {
	r := NewRegistry()
	names := r.List()
	if len(names) != 0 {
		t.Errorf("expected 0 names, got %d", len(names))
	}
}

func TestRegistry_GetEnabled(t *testing.T) {
	r := NewRegistry()
	r.Register(&mockProvider{name: "enabled1", enabled: true})
	r.Register(&mockProvider{name: "disabled", enabled: false})
	r.Register(&mockProvider{name: "enabled2", enabled: true})

	enabled := r.GetEnabled()
	if len(enabled) != 2 {
		t.Fatalf("expected 2 enabled providers, got %d", len(enabled))
	}

	names := make([]string, len(enabled))
	for i, p := range enabled {
		names[i] = p.Name()
	}
	sort.Strings(names)
	if names[0] != "enabled1" || names[1] != "enabled2" {
		t.Errorf("expected [enabled1, enabled2], got %v", names)
	}
}

func TestRegistry_GetEnabled_NoneEnabled(t *testing.T) {
	r := NewRegistry()
	r.Register(&mockProvider{name: "disabled1", enabled: false})
	r.Register(&mockProvider{name: "disabled2", enabled: false})

	enabled := r.GetEnabled()
	if len(enabled) != 0 {
		t.Errorf("expected 0 enabled providers, got %d", len(enabled))
	}
}

func TestRegistry_Register_Overwrite(t *testing.T) {
	r := NewRegistry()
	p1 := &mockProvider{name: "test", enabled: true}
	p2 := &mockProvider{name: "test", enabled: false}

	r.Register(p1)
	r.Register(p2)

	got, _ := r.Get("test")
	if got.IsEnabled() {
		t.Error("expected overwritten provider to be disabled")
	}
}

func TestBlank_Struct(t *testing.T) {
	blank := Blank{
		Name:       "test-template",
		Provider:   "claude",
		Stage:      "plan",
		Purpose:    "testing",
		Version:    "1.0.0",
		Content:    "template content",
		Metadata:   map[string]string{"key": "value"},
		Validation: []string{"check1", "check2"},
	}

	if blank.Name != "test-template" {
		t.Errorf("expected name 'test-template', got '%s'", blank.Name)
	}
	if blank.Provider != "claude" {
		t.Errorf("expected provider 'claude', got '%s'", blank.Provider)
	}
	if len(blank.Validation) != 2 {
		t.Errorf("expected 2 validation rules, got %d", len(blank.Validation))
	}
}

func TestResponse_Struct(t *testing.T) {
	resp := Response{
		Content:  "response content",
		Metadata: map[string]string{"model": "claude-3"},
		Provider: "claude",
		Blank:    "create-issue",
		Success:  true,
	}

	if resp.Content != "response content" {
		t.Errorf("unexpected content: %s", resp.Content)
	}
	if !resp.Success {
		t.Error("expected success to be true")
	}
	if resp.Provider != "claude" {
		t.Errorf("unexpected provider: %s", resp.Provider)
	}
}
