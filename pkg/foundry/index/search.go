package index

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// OfficialFoundryURL is the URL of the official nimble-giant foundry.
// Molds from this foundry are marked as verified.
const OfficialFoundryURL = "https://github.com/nimble-giant/foundry"

// SearchResult represents a single search result from any source.
type SearchResult struct {
	Name        string
	Source      string // e.g. "github.com/owner/repo"
	Description string
	Tags        []string
	Origin      string // "index:<foundry-name>" or "github-topics"
	Stars       int    // only for GitHub Topics results
	URL         string // browsable URL
	Verified    bool   // true if from the official nimble-giant foundry
}

// SearchOptions controls search behavior.
type SearchOptions struct {
	IndexOnly  bool // skip GitHub Topics
	GitHubOnly bool // skip indexes
}

// ghSearchResult represents a single GitHub search result.
type ghSearchResult struct {
	FullName    string `json:"full_name"`
	Description string `json:"description"`
	HTMLURL     string `json:"html_url"`
	StarCount   int    `json:"stargazers_count"`
}

// ghSearchResponse represents the GitHub search API response.
type ghSearchResponse struct {
	Items []ghSearchResult `json:"items"`
}

// Search queries all registered foundry indexes and optionally GitHub Topics.
// Results from indexes appear first, followed by GitHub Topics results.
// Duplicates (same source) are collapsed, preferring the index entry.
func Search(cfg *Config, query string, opts SearchOptions) ([]SearchResult, error) {
	var indexResults []SearchResult
	var ghResults []SearchResult
	var indexErr, ghErr error

	if !opts.GitHubOnly {
		indexResults, indexErr = searchIndexes(cfg, query)
	}

	if !opts.IndexOnly {
		ghResults, ghErr = SearchGitHubTopics(query)
	}

	// If both failed, return the first error.
	if indexErr != nil && ghErr != nil {
		return nil, fmt.Errorf("searching indexes: %w", indexErr)
	}

	// Merge and deduplicate.
	results := mergeResults(indexResults, ghResults)
	return results, nil
}

// searchIndexes searches all cached foundry indexes for matching molds.
func searchIndexes(cfg *Config, query string) ([]SearchResult, error) {
	cacheDir, err := IndexCacheDir()
	if err != nil {
		return nil, err
	}

	var results []SearchResult
	q := strings.ToLower(query)

	for _, entry := range cfg.Foundries {
		idx, err := LoadCachedIndex(cacheDir, &entry)
		if err != nil {
			continue // Skip indexes that aren't cached yet.
		}

		verified := IsOfficialFoundry(entry.URL)
		for _, m := range idx.Molds {
			if matchesMold(m, q) {
				results = append(results, SearchResult{
					Name:        m.Name,
					Source:      m.Source,
					Description: m.Description,
					Tags:        m.Tags,
					Origin:      "index:" + entry.Name,
					URL:         sourceToURL(m.Source),
					Verified:    verified,
				})
			}
		}
	}

	return results, nil
}

// matchesMold checks if a mold entry matches the search query.
// Matches against name, description, and tags (case-insensitive substring).
func matchesMold(m MoldEntry, query string) bool {
	query = strings.ToLower(query)
	if strings.Contains(strings.ToLower(m.Name), query) {
		return true
	}
	if strings.Contains(strings.ToLower(m.Description), query) {
		return true
	}
	for _, tag := range m.Tags {
		if strings.Contains(strings.ToLower(tag), query) {
			return true
		}
	}
	return strings.Contains(strings.ToLower(m.Source), query)
}

// SearchGitHubTopics searches GitHub repositories tagged with ailloy-mold.
// This is extracted from the original runFoundrySearch command handler.
func SearchGitHubTopics(query string) ([]SearchResult, error) {
	endpoint := fmt.Sprintf("search/repositories?q=topic:ailloy-mold+%s&sort=stars&order=desc&per_page=25", query)
	out, err := exec.Command("gh", "api", endpoint).Output() // #nosec G204 -- query is user-provided search term for GitHub API
	if err != nil {
		return nil, fmt.Errorf("searching GitHub (is gh CLI installed and authenticated?): %w", err)
	}

	var resp ghSearchResponse
	if err := json.Unmarshal(out, &resp); err != nil {
		return nil, fmt.Errorf("parsing search results: %w", err)
	}

	var results []SearchResult
	for _, item := range resp.Items {
		results = append(results, SearchResult{
			Name:        item.FullName,
			Source:      item.FullName,
			Description: item.Description,
			Origin:      "github-topics",
			Stars:       item.StarCount,
			URL:         item.HTMLURL,
		})
	}
	return results, nil
}

// mergeResults combines index and GitHub results, deduplicating by source.
// Index results take priority over GitHub results for the same source.
func mergeResults(indexResults, ghResults []SearchResult) []SearchResult {
	seen := make(map[string]bool)
	var merged []SearchResult

	// Index results first.
	for _, r := range indexResults {
		key := strings.ToLower(r.Source)
		seen[key] = true
		merged = append(merged, r)
	}

	// GitHub results second, skipping duplicates.
	for _, r := range ghResults {
		key := strings.ToLower(r.Source)
		if seen[key] {
			continue
		}
		seen[key] = true
		merged = append(merged, r)
	}

	return merged
}

// IsOfficialFoundry returns true if the given URL matches the official nimble-giant foundry.
func IsOfficialFoundry(url string) bool {
	normalized := strings.TrimSuffix(strings.ToLower(url), "/")
	official := strings.ToLower(OfficialFoundryURL)
	return normalized == official
}

// sourceToURL converts a source reference to a browsable URL.
func sourceToURL(source string) string {
	// If it already looks like a URL, return it.
	if strings.HasPrefix(source, "https://") || strings.HasPrefix(source, "http://") {
		return source
	}
	return "https://" + source
}
