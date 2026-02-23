package index

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestMatchesMold(t *testing.T) {
	entry := MoldEntry{
		Name:        "nimble-mold",
		Source:      "github.com/nimble-giant/nimble-mold",
		Description: "Production-ready AI workflow blanks",
		Tags:        []string{"workflows", "claude", "github-actions"},
	}

	tests := []struct {
		query string
		want  bool
	}{
		{"nimble", true},
		{"workflow", true},
		{"claude", true},
		{"github-actions", true},
		{"production", true},
		{"nimble-giant", true},
		{"nonexistent", false},
		{"NIMBLE", true}, // case-insensitive
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			got := matchesMold(entry, tt.query)
			if got != tt.want {
				t.Errorf("matchesMold(%q) = %v, want %v", tt.query, got, tt.want)
			}
		})
	}
}

func TestMergeResults(t *testing.T) {
	indexResults := []SearchResult{
		{Name: "mold1", Source: "github.com/test/mold1", Origin: "index:test"},
		{Name: "mold2", Source: "github.com/test/mold2", Origin: "index:test"},
	}
	ghResults := []SearchResult{
		{Name: "test/mold1", Source: "github.com/test/mold1", Origin: "github-topics", Stars: 10},
		{Name: "test/mold3", Source: "github.com/test/mold3", Origin: "github-topics", Stars: 5},
	}

	merged := mergeResults(indexResults, ghResults)

	if len(merged) != 3 {
		t.Fatalf("len = %d, want 3", len(merged))
	}

	// First two should be from index.
	if merged[0].Origin != "index:test" || merged[0].Name != "mold1" {
		t.Errorf("merged[0] = %+v, want index mold1", merged[0])
	}
	if merged[1].Origin != "index:test" || merged[1].Name != "mold2" {
		t.Errorf("merged[1] = %+v, want index mold2", merged[1])
	}
	// Third should be from GitHub (mold3, since mold1 was deduplicated).
	if merged[2].Origin != "github-topics" || merged[2].Name != "test/mold3" {
		t.Errorf("merged[2] = %+v, want github mold3", merged[2])
	}
}

func TestMergeResults_Empty(t *testing.T) {
	merged := mergeResults(nil, nil)
	if len(merged) != 0 {
		t.Errorf("expected empty results, got %d", len(merged))
	}
}

func TestSourceToURL(t *testing.T) {
	tests := []struct {
		source string
		want   string
	}{
		{"github.com/test/mold", "https://github.com/test/mold"},
		{"https://github.com/test/mold", "https://github.com/test/mold"},
		{"http://gitlab.com/test/mold", "http://gitlab.com/test/mold"},
	}
	for _, tt := range tests {
		t.Run(tt.source, func(t *testing.T) {
			got := sourceToURL(tt.source)
			if got != tt.want {
				t.Errorf("sourceToURL(%q) = %q, want %q", tt.source, got, tt.want)
			}
		})
	}
}

func TestIsOfficialFoundry(t *testing.T) {
	tests := []struct {
		url  string
		want bool
	}{
		{"https://github.com/nimble-giant/foundry", true},
		{"https://github.com/nimble-giant/foundry/", true},
		{"HTTPS://GITHUB.COM/NIMBLE-GIANT/FOUNDRY", true},
		{"https://github.com/nimble-giant/other-repo", false},
		{"https://gitlab.com/nimble-giant/foundry", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			got := IsOfficialFoundry(tt.url)
			if got != tt.want {
				t.Errorf("IsOfficialFoundry(%q) = %v, want %v", tt.url, got, tt.want)
			}
		})
	}
}

func TestSearchIndexes_Verified(t *testing.T) {
	cacheDir := t.TempDir()

	officialEntry := FoundryEntry{
		Name: "official",
		URL:  "https://github.com/nimble-giant/foundry",
		Type: "git",
	}
	otherEntry := FoundryEntry{
		Name: "community",
		URL:  "https://github.com/someone/their-foundry",
		Type: "git",
	}

	// Create cached indexes for both foundries.
	for _, entry := range []FoundryEntry{officialEntry, otherEntry} {
		cachePath := CachedIndexPath(cacheDir, &entry)
		if err := os.MkdirAll(filepath.Dir(cachePath), 0750); err != nil {
			t.Fatal(err)
		}
		content := fmt.Sprintf(`apiVersion: v1
kind: foundry-index
name: %s
molds:
  - name: test-mold
    source: github.com/test/mold
    description: "A test mold"
`, entry.Name)
		if err := os.WriteFile(cachePath, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	cfg := &Config{Foundries: []FoundryEntry{officialEntry, otherEntry}}

	// Simulate what searchIndexes does with a known cache dir.
	var results []SearchResult
	for _, entry := range cfg.Foundries {
		idx, err := LoadCachedIndex(cacheDir, &entry)
		if err != nil {
			t.Fatalf("LoadCachedIndex(%s): %v", entry.Name, err)
		}
		verified := IsOfficialFoundry(entry.URL)
		for _, m := range idx.Molds {
			if matchesMold(m, "test") {
				results = append(results, SearchResult{
					Name:     m.Name,
					Source:   m.Source,
					Origin:   "index:" + entry.Name,
					Verified: verified,
				})
			}
		}
	}

	if len(results) != 2 {
		t.Fatalf("len(results) = %d, want 2", len(results))
	}
	if !results[0].Verified {
		t.Errorf("results[0] (official) Verified = false, want true")
	}
	if results[1].Verified {
		t.Errorf("results[1] (community) Verified = true, want false")
	}
}

func TestSearchIndexes(t *testing.T) {
	// Set up a temporary cache with a pre-cached index.
	cacheDir := t.TempDir()

	entry := FoundryEntry{
		Name: "test-foundry",
		URL:  "https://github.com/test/foundry-index",
		Type: "git",
	}

	// Create cached index.
	cachePath := CachedIndexPath(cacheDir, &entry)
	if err := os.MkdirAll(filepath.Dir(cachePath), 0750); err != nil {
		t.Fatal(err)
	}
	content := `apiVersion: v1
kind: foundry-index
name: test-foundry
molds:
  - name: workflow-mold
    source: github.com/test/workflow-mold
    description: "Workflow automation"
    tags: ["workflows", "ci"]
  - name: data-mold
    source: github.com/test/data-mold
    description: "Data science tools"
    tags: ["data", "python"]
`
	if err := os.WriteFile(cachePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &Config{Foundries: []FoundryEntry{entry}}

	// This test uses the real IndexCacheDir which won't find our temp cache.
	// Instead, test the matchesMold logic directly (already tested above)
	// and the merge logic. The searchIndexes function is integration-tested
	// via the command layer.

	// Verify the cached index can be loaded.
	idx, err := LoadCachedIndex(cacheDir, &entry)
	if err != nil {
		t.Fatalf("LoadCachedIndex: %v", err)
	}
	if len(idx.Molds) != 2 {
		t.Fatalf("len(Molds) = %d, want 2", len(idx.Molds))
	}

	// Simulate what searchIndexes does.
	var results []SearchResult
	for _, foundryEntry := range cfg.Foundries {
		for _, m := range idx.Molds {
			if matchesMold(m, "workflow") {
				results = append(results, SearchResult{
					Name:   m.Name,
					Source: m.Source,
					Origin: "index:" + foundryEntry.Name,
				})
			}
		}
	}

	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}
	if results[0].Name != "workflow-mold" {
		t.Errorf("Name = %q, want workflow-mold", results[0].Name)
	}
}
