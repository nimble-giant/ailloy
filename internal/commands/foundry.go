package commands

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/nimble-giant/ailloy/pkg/foundry/index"
	"github.com/nimble-giant/ailloy/pkg/styles"
	"github.com/spf13/cobra"
)

var (
	searchIndexOnly  bool
	searchGitHubOnly bool
)

var foundryCmd = &cobra.Command{
	Use:   "foundry",
	Short: "Work with Ailloy foundries (mold registries)",
	Long: `Commands for discovering and managing Ailloy foundries.

Foundries are registries of molds and ingots, hosted as git repos or static YAML files.`,
}

var foundrySearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search for molds and ingots",
	Long: `Search registered foundry indexes and GitHub repositories tagged with the ailloy-mold topic.

Results include the mold name, description, source, and origin.
Use --index-only to skip GitHub Topics or --github-only to skip registered indexes.`,
	Args: cobra.ExactArgs(1),
	RunE: runFoundrySearch,
}

var foundryAddCmd = &cobra.Command{
	Use:   "add <url>",
	Short: "Register a foundry index",
	Long: `Register a foundry index URL and fetch its catalog.

The URL can be a git repository containing a foundry.yaml at its root,
or a raw URL to a YAML file (detected by .yaml/.yml extension).

The index is fetched, validated, and cached locally. The registration
is saved to ~/.ailloy/config.yaml.`,
	Args: cobra.ExactArgs(1),
	RunE: runFoundryAdd,
}

var foundryListCmd = &cobra.Command{
	Use:   "list",
	Short: "List registered foundry indexes",
	Long:  `List all registered foundry indexes and their status.`,
	RunE:  runFoundryList,
}

var foundryRemoveCmd = &cobra.Command{
	Use:   "remove <name|url>",
	Short: "Remove a registered foundry index",
	Long: `Remove a registered foundry index by name or URL.

This removes the registration from ~/.ailloy/config.yaml and cleans up cached data.`,
	Args: cobra.ExactArgs(1),
	RunE: runFoundryRemove,
}

var foundryUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Refresh cached foundry indexes",
	Long:  `Fetch the latest version of all registered foundry indexes and update the local cache.`,
	RunE:  runFoundryUpdate,
}

func init() {
	rootCmd.AddCommand(foundryCmd)
	foundryCmd.AddCommand(foundrySearchCmd)
	foundryCmd.AddCommand(foundryAddCmd)
	foundryCmd.AddCommand(foundryListCmd)
	foundryCmd.AddCommand(foundryRemoveCmd)
	foundryCmd.AddCommand(foundryUpdateCmd)

	foundrySearchCmd.Flags().BoolVar(&searchIndexOnly, "index-only", false, "only search registered foundry indexes")
	foundrySearchCmd.Flags().BoolVar(&searchGitHubOnly, "github-only", false, "only search GitHub Topics")
}

func runFoundrySearch(_ *cobra.Command, args []string) error {
	query := args[0]

	fmt.Println(styles.WorkingBanner("Searching foundries..."))
	fmt.Println()

	cfg, err := index.LoadConfig()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	opts := index.SearchOptions{
		IndexOnly:  searchIndexOnly,
		GitHubOnly: searchGitHubOnly,
	}

	results, err := index.Search(cfg, query, opts)
	if err != nil {
		return err
	}

	if len(results) == 0 {
		fmt.Println(styles.InfoBoxStyle.Render(
			styles.InfoStyle.Render("No molds found matching ") +
				styles.CodeStyle.Render(query) + "\n\n" +
				"Try a broader search term or browse " +
				styles.CodeStyle.Render("github.com/topics/ailloy-mold"),
		))
		return nil
	}

	fmt.Println(styles.HeaderStyle.Render(fmt.Sprintf("Found %d result(s):", len(results))))
	fmt.Println()

	for _, r := range results {
		desc := r.Description
		if desc == "" {
			desc = "No description"
		}

		name := styles.AccentStyle.Render(r.Name)
		if r.Verified {
			name += " " + styles.SuccessStyle.Render("✓ verified")
		}
		description := styles.SubtleStyle.Render(" - " + desc)
		origin := styles.SubtleStyle.Render(fmt.Sprintf("  [%s]", r.Origin))
		url := styles.SubtleStyle.Render("  " + r.URL)

		fmt.Println("  " + styles.SuccessStyle.Render("* ") + name + description)
		fmt.Println(origin + " " + url)
	}

	return nil
}

func runFoundryAdd(_ *cobra.Command, args []string) error {
	url := args[0]

	cfg, err := index.LoadConfig()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	entry := index.FoundryEntry{
		Name:   index.DetectType(url), // placeholder, will be overridden by index name
		URL:    url,
		Type:   index.DetectType(url),
		Status: "pending",
	}
	// Derive initial name from URL.
	entry.Name = nameFromFoundryURL(url)

	// Check for duplicates before fetching.
	if existing := cfg.FindFoundry(url); existing != nil {
		fmt.Println(styles.InfoStyle.Render("Foundry already registered: ") + styles.CodeStyle.Render(url))
		return nil
	}

	fmt.Println(styles.WorkingBanner("Fetching foundry index..."))

	// Fetch and validate the index.
	git := defaultGitRunner()
	fetcher, err := index.NewFetcher(git)
	if err != nil {
		return err
	}

	idx, err := fetcher.FetchIndex(&entry)
	if err != nil {
		return fmt.Errorf("fetching foundry index: %w", err)
	}

	// Add to config and save.
	cfg.AddFoundry(entry)
	if err := index.SaveConfig(cfg); err != nil {
		return err
	}

	configPath, _ := index.ConfigPath()

	fmt.Println()
	fmt.Println(styles.SuccessStyle.Render("Foundry registered: ") + styles.AccentStyle.Render(entry.Name))
	fmt.Println(styles.SubtleStyle.Render(fmt.Sprintf("  URL:   %s", url)))
	fmt.Println(styles.SubtleStyle.Render(fmt.Sprintf("  Type:  %s", entry.Type)))
	fmt.Println(styles.SubtleStyle.Render(fmt.Sprintf("  Molds: %d", len(idx.Molds))))
	fmt.Println(styles.SubtleStyle.Render("  Config: " + configPath))

	return nil
}

func runFoundryList(_ *cobra.Command, _ []string) error {
	cfg, err := index.LoadConfig()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if len(cfg.Foundries) == 0 {
		fmt.Println(styles.InfoBoxStyle.Render(
			styles.InfoStyle.Render("No foundries registered.") + "\n\n" +
				"Add one with: " + styles.CodeStyle.Render("ailloy foundry add <url>"),
		))
		return nil
	}

	fmt.Println(styles.HeaderStyle.Render(fmt.Sprintf("Registered foundries (%d):", len(cfg.Foundries))))
	fmt.Println()

	for _, entry := range cfg.Foundries {
		name := styles.AccentStyle.Render(entry.Name)
		if index.IsOfficialFoundry(entry.URL) {
			name += " " + styles.SuccessStyle.Render("✓ verified")
		}
		status := formatStatus(entry.Status)
		lastUpdated := "never"
		if !entry.LastUpdated.IsZero() {
			lastUpdated = entry.LastUpdated.Format(time.RFC3339)
		}

		fmt.Println("  " + styles.SuccessStyle.Render("* ") + name + " " + status)
		fmt.Println(styles.SubtleStyle.Render(fmt.Sprintf("    URL:     %s", entry.URL)))
		fmt.Println(styles.SubtleStyle.Render(fmt.Sprintf("    Type:    %s", entry.Type)))
		fmt.Println(styles.SubtleStyle.Render(fmt.Sprintf("    Updated: %s", lastUpdated)))
		fmt.Println()
	}

	return nil
}

func runFoundryRemove(_ *cobra.Command, args []string) error {
	nameOrURL := args[0]

	cfg, err := index.LoadConfig()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	// Find the entry before removing (for cache cleanup).
	entry := cfg.FindFoundry(nameOrURL)
	if entry == nil {
		return fmt.Errorf("foundry %q not found", nameOrURL)
	}

	// Clean up cached index files.
	cacheDir, err := index.IndexCacheDir()
	if err == nil {
		_ = index.CleanIndexCache(cacheDir, entry)
	}

	if !cfg.RemoveFoundry(nameOrURL) {
		return fmt.Errorf("foundry %q not found", nameOrURL)
	}

	if err := index.SaveConfig(cfg); err != nil {
		return err
	}

	fmt.Println(styles.SuccessStyle.Render("Foundry removed: ") + styles.CodeStyle.Render(nameOrURL))

	return nil
}

func runFoundryUpdate(_ *cobra.Command, _ []string) error {
	cfg, err := index.LoadConfig()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if len(cfg.Foundries) == 0 {
		fmt.Println(styles.InfoStyle.Render("No foundries registered."))
		return nil
	}

	fmt.Println(styles.WorkingBanner("Updating foundry indexes..."))
	fmt.Println()

	git := defaultGitRunner()
	fetcher, err := index.NewFetcher(git)
	if err != nil {
		return err
	}

	var errors []string
	for i := range cfg.Foundries {
		entry := &cfg.Foundries[i]
		fmt.Printf("  Updating %s...", styles.AccentStyle.Render(entry.Name))

		idx, err := fetcher.FetchIndex(entry)
		if err != nil {
			fmt.Println(" " + styles.ErrorStyle.Render("error"))
			fmt.Println(styles.SubtleStyle.Render("    " + err.Error()))
			errors = append(errors, fmt.Sprintf("%s: %v", entry.Name, err))
			continue
		}

		fmt.Println(" " + styles.SuccessStyle.Render("ok") +
			styles.SubtleStyle.Render(fmt.Sprintf(" (%d molds)", len(idx.Molds))))
	}

	// Save updated timestamps and statuses.
	if err := index.SaveConfig(cfg); err != nil {
		return err
	}

	fmt.Println()
	if len(errors) > 0 {
		return fmt.Errorf("%d foundry(ies) failed to update", len(errors))
	}

	fmt.Println(styles.SuccessStyle.Render("All foundries updated successfully."))
	return nil
}

func formatStatus(status string) string {
	switch status {
	case "ok":
		return styles.SuccessStyle.Render("[ok]")
	case "error":
		return styles.ErrorStyle.Render("[error]")
	default:
		return styles.SubtleStyle.Render("[pending]")
	}
}

// defaultGitRunner returns a GitRunner that shells out to git.
func defaultGitRunner() index.GitRunner {
	return func(args ...string) ([]byte, error) {
		cmd := exec.Command("git", args...) // #nosec G204 -- args are constructed internally
		return cmd.CombinedOutput()
	}
}

// nameFromFoundryURL derives a short name from a foundry URL.
func nameFromFoundryURL(url string) string {
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "http://")

	parts := strings.Split(strings.TrimSuffix(url, "/"), "/")
	if len(parts) > 0 {
		name := parts[len(parts)-1]
		name = strings.TrimSuffix(name, ".yaml")
		name = strings.TrimSuffix(name, ".yml")
		if name != "" {
			return name
		}
	}
	return "foundry"
}
