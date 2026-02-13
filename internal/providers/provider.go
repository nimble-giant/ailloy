package providers

import (
	"context"
	"fmt"
)

// Provider defines the interface for AI providers
type Provider interface {
	// Name returns the provider name (e.g., "claude", "gpt")
	Name() string
	
	// ExecuteTemplate runs a template against the provider
	ExecuteTemplate(ctx context.Context, template Template, context map[string]interface{}) (*Response, error)
	
	// ValidateConfig checks if the provider configuration is valid
	ValidateConfig() error
	
	// IsEnabled returns whether the provider is enabled
	IsEnabled() bool
}

// Template represents an AI command template
type Template struct {
	Name        string            `yaml:"name"`
	Provider    string            `yaml:"provider"`
	Stage       string            `yaml:"stage"`
	Purpose     string            `yaml:"purpose"`
	Version     string            `yaml:"version"`
	Content     string            `yaml:"content"`
	Metadata    map[string]string `yaml:"metadata"`
	Validation  []string          `yaml:"validation"`
}

// Response represents the result from an AI provider
type Response struct {
	Content   string            `json:"content"`
	Metadata  map[string]string `json:"metadata"`
	Provider  string            `json:"provider"`
	Template  string            `json:"template"`
	Success   bool              `json:"success"`
	Error     string            `json:"error,omitempty"`
}

// Registry manages available providers
type Registry struct {
	providers map[string]Provider
}

// NewRegistry creates a new provider registry
func NewRegistry() *Registry {
	return &Registry{
		providers: make(map[string]Provider),
	}
}

// Register adds a provider to the registry
func (r *Registry) Register(provider Provider) {
	r.providers[provider.Name()] = provider
}

// Get retrieves a provider by name
func (r *Registry) Get(name string) (Provider, error) {
	provider, exists := r.providers[name]
	if !exists {
		return nil, fmt.Errorf("provider %s not found", name)
	}
	return provider, nil
}

// List returns all registered provider names
func (r *Registry) List() []string {
	names := make([]string, 0, len(r.providers))
	for name := range r.providers {
		names = append(names, name)
	}
	return names
}

// GetEnabled returns all enabled providers
func (r *Registry) GetEnabled() []Provider {
	enabled := make([]Provider, 0)
	for _, provider := range r.providers {
		if provider.IsEnabled() {
			enabled = append(enabled, provider)
		}
	}
	return enabled
}