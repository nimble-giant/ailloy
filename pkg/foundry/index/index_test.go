package index

import (
	"strings"
	"testing"
)

func TestParseIndex(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Index
		wantErr bool
	}{
		{
			name: "valid index",
			input: `
apiVersion: v1
kind: foundry-index
name: test-foundry
description: "A test foundry"
author:
  name: Test Author
  url: https://example.com
molds:
  - name: my-mold
    source: github.com/test/my-mold
    description: "A test mold"
    tags: ["test", "example"]
    version: v1.0.0
`,
			want: Index{
				APIVersion:  "v1",
				Kind:        "foundry-index",
				Name:        "test-foundry",
				Description: "A test foundry",
				Author:      Author{Name: "Test Author", URL: "https://example.com"},
				Molds: []MoldEntry{
					{
						Name:        "my-mold",
						Source:      "github.com/test/my-mold",
						Description: "A test mold",
						Tags:        []string{"test", "example"},
						Version:     "v1.0.0",
					},
				},
			},
		},
		{
			name: "minimal index",
			input: `
apiVersion: v1
kind: foundry-index
name: minimal
molds: []
`,
			want: Index{
				APIVersion: "v1",
				Kind:       "foundry-index",
				Name:       "minimal",
				Molds:      []MoldEntry{},
			},
		},
		{
			name:    "invalid yaml",
			input:   `{{{not yaml`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			idx, err := ParseIndex([]byte(tt.input))
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if idx.APIVersion != tt.want.APIVersion {
				t.Errorf("APIVersion = %q, want %q", idx.APIVersion, tt.want.APIVersion)
			}
			if idx.Kind != tt.want.Kind {
				t.Errorf("Kind = %q, want %q", idx.Kind, tt.want.Kind)
			}
			if idx.Name != tt.want.Name {
				t.Errorf("Name = %q, want %q", idx.Name, tt.want.Name)
			}
			if idx.Description != tt.want.Description {
				t.Errorf("Description = %q, want %q", idx.Description, tt.want.Description)
			}
			if idx.Author.Name != tt.want.Author.Name {
				t.Errorf("Author.Name = %q, want %q", idx.Author.Name, tt.want.Author.Name)
			}
			if len(idx.Molds) != len(tt.want.Molds) {
				t.Fatalf("len(Molds) = %d, want %d", len(idx.Molds), len(tt.want.Molds))
			}
			for i, m := range idx.Molds {
				if m.Name != tt.want.Molds[i].Name {
					t.Errorf("Molds[%d].Name = %q, want %q", i, m.Name, tt.want.Molds[i].Name)
				}
				if m.Source != tt.want.Molds[i].Source {
					t.Errorf("Molds[%d].Source = %q, want %q", i, m.Source, tt.want.Molds[i].Source)
				}
			}
		})
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		idx     Index
		wantErr string
	}{
		{
			name: "valid",
			idx: Index{
				APIVersion: "v1",
				Kind:       "foundry-index",
				Name:       "test",
				Molds: []MoldEntry{
					{Name: "mold1", Source: "github.com/test/mold1"},
				},
			},
		},
		{
			name: "missing apiVersion",
			idx: Index{
				Kind: "foundry-index",
				Name: "test",
			},
			wantErr: "apiVersion is required",
		},
		{
			name: "wrong kind",
			idx: Index{
				APIVersion: "v1",
				Kind:       "mold",
				Name:       "test",
			},
			wantErr: "kind must be",
		},
		{
			name: "missing name",
			idx: Index{
				APIVersion: "v1",
				Kind:       "foundry-index",
			},
			wantErr: "name is required",
		},
		{
			name: "mold missing name",
			idx: Index{
				APIVersion: "v1",
				Kind:       "foundry-index",
				Name:       "test",
				Molds: []MoldEntry{
					{Source: "github.com/test/mold1"},
				},
			},
			wantErr: "molds[0].name is required",
		},
		{
			name: "mold missing source",
			idx: Index{
				APIVersion: "v1",
				Kind:       "foundry-index",
				Name:       "test",
				Molds: []MoldEntry{
					{Name: "mold1"},
				},
			},
			wantErr: "molds[0].source is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.idx.Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error %q does not contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}
