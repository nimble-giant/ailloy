package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/nimble-giant/ailloy/pkg/styles"
	"github.com/spf13/cobra"
)

var foundryCmd = &cobra.Command{
	Use:   "foundry",
	Short: "Work with Ailloy foundries (mold registries)",
	Long: `Commands for discovering and managing Ailloy foundries.

Foundries are SCM-hosted registries of molds and ingots.`,
}

var foundrySearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search for molds and ingots",
	Long: `Search GitHub repositories tagged with the ailloy-mold topic.

Results include the repository name, description, latest version, and URL.`,
	Args: cobra.ExactArgs(1),
	RunE: runFoundrySearch,
}

var foundryAddCmd = &cobra.Command{
	Use:   "add <url>",
	Short: "Register a foundry index URL",
	Long: `Register a foundry index URL in your configuration.

The URL is saved to ~/.ailloy/config.yaml for use by search and resolution commands.`,
	Args: cobra.ExactArgs(1),
	RunE: runFoundryAdd,
}

func init() {
	rootCmd.AddCommand(foundryCmd)
	foundryCmd.AddCommand(foundrySearchCmd)
	foundryCmd.AddCommand(foundryAddCmd)
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

func runFoundrySearch(_ *cobra.Command, args []string) error {
	query := args[0]

	fmt.Println(styles.WorkingBanner("Searching foundries..."))
	fmt.Println()

	// Search GitHub repos by topic ailloy-mold
	endpoint := fmt.Sprintf("search/repositories?q=topic:ailloy-mold+%s&sort=stars&order=desc&per_page=25", query)
	out, err := exec.Command("gh", "api", endpoint).Output() // #nosec G204 -- query is user-provided search term for GitHub API
	if err != nil {
		return fmt.Errorf("searching GitHub (is gh CLI installed and authenticated?): %w", err)
	}

	var resp ghSearchResponse
	if err := json.Unmarshal(out, &resp); err != nil {
		return fmt.Errorf("parsing search results: %w", err)
	}

	if len(resp.Items) == 0 {
		fmt.Println(styles.InfoBoxStyle.Render(
			styles.InfoStyle.Render("No molds found matching ") +
				styles.CodeStyle.Render(query) + "\n\n" +
				"Try a broader search term or browse " +
				styles.CodeStyle.Render("github.com/topics/ailloy-mold"),
		))
		return nil
	}

	fmt.Println(styles.HeaderStyle.Render(fmt.Sprintf("Found %d result(s):", len(resp.Items))))
	fmt.Println()

	for _, item := range resp.Items {
		desc := item.Description
		if desc == "" {
			desc = "No description"
		}

		name := styles.AccentStyle.Render(item.FullName)
		description := styles.SubtleStyle.Render(" - " + desc)
		url := styles.SubtleStyle.Render("  " + item.HTMLURL)

		fmt.Println("  " + styles.SuccessStyle.Render("* ") + name + description)
		fmt.Println(url)
	}

	return nil
}

// foundryConfig represents the ~/.ailloy/config.yaml structure.
type foundryConfig struct {
	Foundries []string `yaml:"foundries,omitempty"`
}

func runFoundryAdd(_ *cobra.Command, args []string) error {
	url := args[0]

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("cannot determine home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".ailloy")
	configPath := filepath.Join(configDir, "config.yaml")

	// Load existing config or create new one.
	var cfg foundryConfig
	if data, err := os.ReadFile(configPath); err == nil { // #nosec G304 -- reading user config file
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return fmt.Errorf("parsing config file: %w", err)
		}
	}

	// Check for duplicates.
	for _, existing := range cfg.Foundries {
		if strings.EqualFold(existing, url) {
			fmt.Println(styles.InfoStyle.Render("Foundry already registered: ") + styles.CodeStyle.Render(url))
			return nil
		}
	}

	cfg.Foundries = append(cfg.Foundries, url)

	// Ensure directory exists.
	if err := os.MkdirAll(configDir, 0750); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	data, err := yaml.Marshal(&cfg)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil { // #nosec G306 -- user config file
		return fmt.Errorf("writing config file: %w", err)
	}

	fmt.Println(styles.SuccessStyle.Render("Foundry registered: ") + styles.CodeStyle.Render(url))
	fmt.Println(styles.SubtleStyle.Render("Config: " + configPath))

	return nil
}
