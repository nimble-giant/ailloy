package commands

import (
	"context"
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

Bare references like "github.com/owner/repo" default to https://, the
same way ailloy cast resolves shorthand sources. Explicit https://,
http://, and git@ schemes are kept as-is.

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

var (
	foundryInstallGlobal        bool
	foundryInstallWithWorkflows bool
	foundryInstallDryRun        bool
	foundryInstallForce         bool
	foundryInstallClaudePlugin  bool
	foundryInstallShallow       bool
)

var foundryInstallCmd = &cobra.Command{
	Use:     "install <name|url>",
	Aliases: []string{"cast-all"},
	Short:   "Cast every mold listed by a foundry",
	Long: `Cast every mold listed by the named foundry.

The foundry is looked up by name or URL across the effective foundries
(verified default + registered). Molds already present in the target
lockfile are skipped unless --force is given.`,
	Args: cobra.ExactArgs(1),
	RunE: runFoundryInstall,
}

func init() {
	rootCmd.AddCommand(foundryCmd)
	foundryCmd.AddCommand(foundrySearchCmd)
	foundryCmd.AddCommand(foundryAddCmd)
	foundryCmd.AddCommand(foundryNewCmd)
	foundryCmd.AddCommand(foundryListCmd)
	foundryCmd.AddCommand(foundryRemoveCmd)
	foundryCmd.AddCommand(foundryUpdateCmd)
	foundryCmd.AddCommand(foundryInstallCmd)

	foundrySearchCmd.Flags().BoolVar(&searchIndexOnly, "index-only", false, "only search registered foundry indexes")
	foundrySearchCmd.Flags().BoolVar(&searchGitHubOnly, "github-only", false, "only search GitHub Topics")

	foundryInstallCmd.Flags().BoolVarP(&foundryInstallGlobal, "global", "g", false, "install each mold under ~/ instead of the current project")
	foundryInstallCmd.Flags().BoolVar(&foundryInstallWithWorkflows, "with-workflows", false, "include GitHub Actions workflow blanks")
	foundryInstallCmd.Flags().BoolVar(&foundryInstallDryRun, "dry-run", false, "list what would be installed without casting")
	foundryInstallCmd.Flags().BoolVar(&foundryInstallForce, "force", false, "re-cast molds that are already installed")
	foundryInstallCmd.Flags().BoolVar(&foundryInstallClaudePlugin, "claude-plugin", false, "package each mold as a Claude Code plugin under .claude/plugins/<slug>/")
	foundryInstallCmd.Flags().BoolVar(&foundryInstallShallow, "shallow", false, "install only the named foundry's direct molds (skip nested foundries)")
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
	url := index.NormalizeFoundryURL(args[0])

	cfg, err := index.LoadConfig()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if existing := cfg.FindFoundry(url); existing != nil {
		fmt.Println(styles.InfoStyle.Render("Foundry already registered: ") + styles.CodeStyle.Render(url))
		return nil
	}

	fmt.Println(styles.WorkingBanner("Fetching foundry index..."))

	res, err := AddFoundryCore(cfg, url)
	if err != nil {
		return err
	}
	if res.AlreadyExists {
		fmt.Println(styles.InfoStyle.Render("Foundry already registered: ") + styles.CodeStyle.Render(url))
		return nil
	}

	if err := index.SaveConfig(cfg); err != nil {
		return err
	}

	configPath, _ := index.ConfigPath()

	fmt.Println()
	fmt.Println(styles.SuccessStyle.Render("Foundry registered: ") + styles.AccentStyle.Render(res.Entry.Name))
	fmt.Println(styles.SubtleStyle.Render(fmt.Sprintf("  URL:   %s", url)))
	fmt.Println(styles.SubtleStyle.Render(fmt.Sprintf("  Type:  %s", res.Entry.Type)))
	fmt.Println(styles.SubtleStyle.Render(fmt.Sprintf("  Molds: %d", res.MoldCount)))
	fmt.Println(styles.SubtleStyle.Render("  Config: " + configPath))

	return nil
}

func runFoundryList(_ *cobra.Command, _ []string) error {
	cfg, err := index.LoadConfig()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	foundries := cfg.EffectiveFoundries()

	fmt.Println(styles.HeaderStyle.Render(fmt.Sprintf("Registered foundries (%d):", len(foundries))))
	fmt.Println()

	cacheDir, _ := index.IndexCacheDir()
	fetcher, _ := index.NewFetcher(defaultGitRunner())
	var lookup index.IndexLookup
	if cacheDir != "" && fetcher != nil {
		lookup = index.CacheFirstLookup(cacheDir, fetcher)
	}

	for _, entry := range foundries {
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

		// Render nested-foundry tree if any. Skip silently when the index
		// can't be resolved (offline / not cached / no children).
		if lookup != nil {
			r := index.NewResolver(lookup)
			root, _, err := r.Resolve(entry.URL)
			if err == nil && len(root.Children) > 0 {
				fmt.Println(styles.SubtleStyle.Render("    Nested foundries:"))
				for _, child := range root.Children {
					printFoundryTree(child, "      ")
				}
			}
		}

		fmt.Println()
	}

	return nil
}

// printFoundryTree recursively prints a child foundry node and its descendants.
func printFoundryTree(node *index.ResolvedFoundry, indent string) {
	line := fmt.Sprintf("%s└─ %s (%s)", indent, node.Index.Name, node.Source)
	fmt.Println(styles.SubtleStyle.Render(line))
	for _, c := range node.Children {
		printFoundryTree(c, indent+"   ")
	}
}

func runFoundryRemove(_ *cobra.Command, args []string) error {
	nameOrURL := args[0]

	cfg, err := index.LoadConfig()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	removed, err := RemoveFoundryCore(cfg, nameOrURL)
	if err != nil {
		return err
	}

	if cacheDir, cerr := index.IndexCacheDir(); cerr == nil {
		_ = index.CleanIndexCache(cacheDir, &removed)
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

	fmt.Println(styles.WorkingBanner("Updating foundry indexes..."))
	fmt.Println()

	reports, err := UpdateFoundriesCore(cfg)
	if err != nil {
		return err
	}

	var failures int
	for _, r := range reports {
		fmt.Printf("  Updating %s...", styles.AccentStyle.Render(r.Name))
		if r.Err != nil {
			failures++
			fmt.Println(" " + styles.ErrorStyle.Render("error"))
			fmt.Println(styles.SubtleStyle.Render("    " + r.Err.Error()))
			continue
		}
		fmt.Println(" " + styles.SuccessStyle.Render("ok") +
			styles.SubtleStyle.Render(fmt.Sprintf(" (%d molds)", r.MoldCount)))
	}

	if err := index.SaveConfig(cfg); err != nil {
		return err
	}

	fmt.Println()
	if failures > 0 {
		return fmt.Errorf("%d foundry(ies) failed to update", failures)
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

func runFoundryInstall(_ *cobra.Command, args []string) error {
	nameOrURL := args[0]

	cfg, err := index.LoadConfig()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	fmt.Println(styles.WorkingBanner("Casting every mold from " + nameOrURL + "..."))
	fmt.Println()

	reports, err := InstallFoundryCore(context.Background(), cfg, nameOrURL, InstallFoundryOptions{
		Global:        foundryInstallGlobal,
		WithWorkflows: foundryInstallWithWorkflows,
		DryRun:        foundryInstallDryRun,
		Force:         foundryInstallForce,
		ClaudePlugin:  foundryInstallClaudePlugin,
		Shallow:       foundryInstallShallow,
	})
	if err != nil {
		return err
	}

	var (
		installed int
		skipped   int
		failed    int
	)
	lastFnd := "\x00" // sentinel so the first iteration always prints a header
	for _, r := range reports {
		if r.Foundry != lastFnd {
			if lastFnd != "\x00" {
				fmt.Println()
			}
			header := "Installing from " + styles.AccentStyle.Render(r.Foundry)
			if len(r.Chain) > 0 {
				header += styles.SubtleStyle.Render(" (via " + strings.Join(r.Chain, " → ") + ")")
			}
			fmt.Println(header)
			lastFnd = r.Foundry
		}
		display := r.Foundry + "/" + r.Name
		fmt.Printf("  %s", styles.AccentStyle.Render(display))
		switch {
		case r.Err != nil:
			failed++
			fmt.Println(" " + styles.ErrorStyle.Render("error"))
			fmt.Println(styles.SubtleStyle.Render("    " + r.Err.Error()))
		case r.Skipped:
			skipped++
			extra := ""
			if r.Version != "" {
				extra = " " + styles.SubtleStyle.Render(r.Version)
			}
			fmt.Println(" " + styles.InfoStyle.Render("skipped (already installed)") + extra)
		case foundryInstallDryRun:
			fmt.Println(" " + styles.InfoStyle.Render("would install"))
		default:
			installed++
			fmt.Println(" " + styles.SuccessStyle.Render("ok"))
		}
	}

	if len(reports) == 0 && foundryInstallShallow {
		fmt.Println(styles.InfoStyle.Render("Nothing to install — this foundry only aggregates nested foundries."))
		fmt.Println(styles.SubtleStyle.Render("Drop --shallow to install transitively."))
		return nil
	}

	fmt.Println()
	summary := fmt.Sprintf("installed %d · skipped %d · failed %d", installed, skipped, failed)
	if foundryInstallDryRun {
		summary = fmt.Sprintf("would install %d · skipped %d · failed %d", len(reports)-skipped-failed, skipped, failed)
	}
	fmt.Println(styles.SuccessStyle.Render(summary))

	if failed > 0 {
		return fmt.Errorf("%d mold(s) failed to install", failed)
	}
	return nil
}
