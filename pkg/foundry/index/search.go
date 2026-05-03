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
//
// Warnings carries non-fatal sub-foundry resolution failures (e.g. a private
// nested foundry the caller cannot access). The CLI is expected to render
// these so users understand why some molds may be missing.
func Search(cfg *Config, query string, opts SearchOptions) ([]SearchResult, []ResolutionWarning, error) {
	var indexResults []SearchResult
	var ghResults []SearchResult
	var warnings []ResolutionWarning
	var indexErr, ghErr error

	if !opts.GitHubOnly {
		indexResults, warnings, indexErr = searchIndexes(cfg, query)
	}

	if !opts.IndexOnly {
		ghResults, ghErr = SearchGitHubTopics(query)
	}

	// If both failed, return the first error.
	if indexErr != nil && ghErr != nil {
		return nil, warnings, fmt.Errorf("searching indexes: %w", indexErr)
	}

	// Merge and deduplicate.
	results := mergeResults(indexResults, ghResults)
	return results, warnings, nil
}

// searchIndexes searches all cached foundry indexes for matching molds,
// transitively resolving any nested foundries declared in their `foundries:`
// fields. Lookups are cache-first; child foundries not yet cached fall back
// to the network so a freshly-added parent works without an explicit update.
func searchIndexes(cfg *Config, query string) ([]SearchResult, []ResolutionWarning, error) {
	cacheDir, err := IndexCacheDir()
	if err != nil {
		return nil, nil, err
	}
	git := defaultGitRunnerForSearch()
	fetcher := NewFetcherWithCacheDir(git, cacheDir)
	lookup := CacheFirstLookup(cacheDir, fetcher)
	return searchWithLookup(cfg, query, lookup)
}

// searchWithLookup is the testable core: given an explicit IndexLookup, walk
// every registered root via a fresh Resolver and return matching molds with
// their resolution chain. Sub-foundry resolution failures are returned as
// warnings rather than swallowed.
func searchWithLookup(cfg *Config, query string, lookup IndexLookup) ([]SearchResult, []ResolutionWarning, error) {
	var results []SearchResult
	var warnings []ResolutionWarning
	q := strings.ToLower(query)

	for _, entry := range cfg.EffectiveFoundries() {
		verified := IsOfficialFoundry(entry.URL)
		r := NewResolver(lookup)
		root, molds, err := r.Resolve(entry.URL)
		if err != nil {
			// Root failed (e.g., not cached and offline). Skip — preserves
			// the existing "skip indexes that aren't cached yet" behavior.
			// Sub-foundry failures of an otherwise-resolvable root are still
			// surfaced as warnings below.
			continue
		}
		warnings = append(warnings, r.Warnings()...)
		for _, m := range molds {
			if !matchesMold(m.Entry, q) {
				continue
			}
			results = append(results, SearchResult{
				Name:        m.Foundry.Index.Name + "/" + m.Entry.Name,
				Source:      m.Entry.Source,
				Description: m.Entry.Description,
				Tags:        m.Entry.Tags,
				Origin:      formatOrigin(root, m.Foundry),
				URL:         sourceToURL(m.Entry.Source),
				Verified:    verified && m.Foundry == root,
			})
		}
	}
	return results, warnings, nil
}

// formatOrigin renders the resolution chain. Root molds get "index:<root>";
// nested molds get "index:<root> via <parent> → <child>".
func formatOrigin(root, owner *ResolvedFoundry) string {
	base := "index:" + root.Index.Name
	if owner == root {
		return base
	}
	chain := append(append([]string(nil), owner.Parents...), owner.Index.Name)
	return base + " via " + strings.Join(chain, " → ")
}

// CacheFirstLookup returns an IndexLookup that reads the cache when present
// and falls back to the network (via Fetcher) otherwise. The synthesized
// FoundryEntry mirrors what `foundry add` would build for the same URL.
// Used by both search and the foundry list command.
func CacheFirstLookup(cacheDir string, fetcher *Fetcher) IndexLookup {
	return func(source string) (*Index, error) {
		url := NormalizeFoundryURL(source)
		entry := &FoundryEntry{
			URL:  url,
			Type: DetectType(url),
		}
		if idx, err := LoadCachedIndex(cacheDir, entry); err == nil {
			return idx, nil
		}
		return fetcher.FetchIndex(entry)
	}
}

// defaultGitRunnerForSearch returns a GitRunner used by the cache-fallback
// fetcher inside searchIndexes. Defined locally to avoid importing
// internal/commands from the index package.
func defaultGitRunnerForSearch() GitRunner {
	return func(args ...string) ([]byte, error) {
		cmd := exec.Command("git", args...) // #nosec G204 -- args are constructed internally
		return cmd.CombinedOutput()
	}
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
