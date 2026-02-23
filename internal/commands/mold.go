package commands

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/goccy/go-yaml"
	"github.com/nimble-giant/ailloy/pkg/foundry"
	"github.com/nimble-giant/ailloy/pkg/mold"
	"github.com/nimble-giant/ailloy/pkg/styles"
	"github.com/spf13/cobra"
)

var moldCmd = &cobra.Command{
	Use:   "mold",
	Short: "Work with Ailloy molds (blanks)",
	Long: `Commands for managing and executing Ailloy molds.

Molds are Markdown files that define AI commands and workflows.`,
}

var listMoldsCmd = &cobra.Command{
	Use:   "list",
	Short: "List available molds",
	RunE:  runListMolds,
}

var showMoldCmd = &cobra.Command{
	Use:   "show <mold-name>",
	Short: "Display a mold's content",
	Args:  cobra.ExactArgs(1),
	RunE:  runShowMold,
}

// showCmd is a top-level command that enables bidirectional syntax: "mold show" and "show mold"
var showCmd = &cobra.Command{
	Use:   "show",
	Short: "Show resources",
}

var showMoldSubCmd = &cobra.Command{
	Use:   "mold <mold-name>",
	Short: "Display a mold's content",
	Args:  cobra.ExactArgs(1),
	RunE:  runShowMold,
}

var getMoldCmd = &cobra.Command{
	Use:   "get <reference>",
	Short: "Download a mold without installing",
	Long: `Download a mold to the local cache without installing it.

The reference follows the standard format: <host>/<owner>/<repo>[@<version>][//<subpath>]
After download, the cache path is printed and the mold manifest is validated.`,
	Args: cobra.ExactArgs(1),
	RunE: runGetMold,
}

func init() {
	rootCmd.AddCommand(moldCmd)
	moldCmd.AddCommand(listMoldsCmd)
	moldCmd.AddCommand(showMoldCmd)
	moldCmd.AddCommand(getMoldCmd)

	// Bidirectional: "show mold <name>" also works
	rootCmd.AddCommand(showCmd)
	showCmd.AddCommand(showMoldSubCmd)
}

func runListMolds(cmd *cobra.Command, args []string) error {
	moldDirs, workflowDirs := loadInstalledDirs()

	// Header with inquisitive fox for exploring molds
	header := lipgloss.JoinVertical(
		lipgloss.Center,
		styles.FoxArt("inquisitive"),
		styles.HeaderStyle.Render("Available Blanks"),
	)
	fmt.Println(header)
	fmt.Println()

	foundMolds := false

	for _, dir := range moldDirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) { // #nosec G703 -- CLI tool intentionally accesses user-specified blank directories
			continue
		}

		// Walk through subdirectories to find blanks
		err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error { // #nosec G703 -- Intentional directory traversal for blank discovery
			if err != nil {
				return nil // Skip errors, continue walking
			}

			// Only process .md files
			if !d.IsDir() && strings.HasSuffix(path, ".md") {
				// Get relative path from base dir for category
				relPath, _ := filepath.Rel(dir, path)
				pathParts := strings.Split(filepath.Dir(relPath), string(filepath.Separator))

				var category string
				if len(pathParts) > 0 && pathParts[0] != "." {
					category = pathParts[0]
				} else {
					category = "general"
				}

				fileName := filepath.Base(path)
				blankName := strings.TrimSuffix(fileName, ".md")

				// Try to extract the first line as description
				content, err := os.ReadFile(path) // #nosec G304 -- CLI tool reads user blank files
				if err != nil {
					errorMsg := styles.ErrorStyle.Render("❌ ") +
						styles.AccentStyle.Render(category+"/"+blankName) +
						styles.SubtleStyle.Render(" (unreadable)")
					fmt.Println("  " + errorMsg)
					return nil
				}

				lines := strings.Split(string(content), "\n")
				var description string
				if len(lines) > 0 && strings.HasPrefix(lines[0], "# ") {
					description = strings.TrimPrefix(lines[0], "# ")
				} else {
					description = "Blank"
				}

				// Style the blank listing
				icon := getMoldIcon(blankName)
				blankDisplay := styles.SuccessStyle.Render(icon+" ") +
					styles.AccentStyle.Render(category+"/"+blankName) +
					styles.SubtleStyle.Render(" - "+description)
				fmt.Println("  " + blankDisplay)
				foundMolds = true
			}

			return nil
		})

		if err != nil {
			continue
		}
	}

	// List workflow blanks
	for _, dir := range workflowDirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			continue
		}

		err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}

			if !d.IsDir() && strings.HasSuffix(path, ".yml") {
				fileName := filepath.Base(path)
				blankName := strings.TrimSuffix(fileName, ".yml")

				// Extract the workflow name from the YAML
				content, err := os.ReadFile(path) // #nosec G304 -- CLI tool reads user workflow files
				if err != nil {
					return nil
				}

				var description string
				for _, line := range strings.Split(string(content), "\n") {
					if strings.HasPrefix(line, "name:") {
						description = strings.TrimSpace(strings.TrimPrefix(line, "name:"))
						break
					}
				}
				if description == "" {
					description = "GitHub Actions workflow"
				}

				icon := getMoldIcon(blankName)
				blankDisplay := styles.SuccessStyle.Render(icon+" ") +
					styles.AccentStyle.Render("workflows/"+blankName) +
					styles.SubtleStyle.Render(" - "+description)
				fmt.Println("  " + blankDisplay)
				foundMolds = true
			}

			return nil
		})

		if err != nil {
			continue
		}
	}

	if !foundMolds {
		noMoldsMsg := styles.InfoBoxStyle.Render(
			styles.InfoStyle.Render("ℹ️  No molds found.\n\n") +
				"Run " + styles.CodeStyle.Render("ailloy cast") + " to set up molds.",
		)
		fmt.Println(noMoldsMsg)
	}

	return nil
}

func runShowMold(cmd *cobra.Command, args []string) error {
	moldName := args[0]

	// Find mold file
	moldPath, err := findMold(moldName)
	if err != nil {
		errorMsg := styles.ErrorBoxStyle.Render(
			styles.ErrorStyle.Render("❌ Mold not found: ") +
				styles.CodeStyle.Render(moldName) + "\n\n" +
				"Run " + styles.CodeStyle.Render("ailloy mold list") + " to see available molds.",
		)
		fmt.Println(errorMsg)
		return nil
	}

	// Read and display the mold content
	content, err := os.ReadFile(moldPath) // #nosec G304 -- CLI tool reads user blank files
	if err != nil {
		return fmt.Errorf("failed to read mold: %w", err)
	}

	// Header with small fox emoji
	icon := getMoldIcon(moldName)
	header := lipgloss.JoinVertical(
		lipgloss.Center,
		styles.FoxArt("small"),
		styles.HeaderStyle.Render(icon+" Mold: "+moldName),
	)
	fmt.Println(header)

	// Path info
	pathInfo := styles.SubtleStyle.Render("📁 Path: " + moldPath)
	fmt.Println(pathInfo)
	fmt.Println()

	// Content in a styled box
	contentBox := styles.BoxStyle.Render(string(content))
	fmt.Println(contentBox)

	return nil
}

func findMold(name string) (string, error) {
	blankDirs, _ := loadInstalledDirs()

	for _, dir := range blankDirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) { // #nosec G703 -- CLI tool checks blank directory existence
			continue
		}

		// First try direct path (for backward compatibility)
		blankPath := filepath.Join(dir, name+".md")
		if _, err := os.Stat(blankPath); err == nil { // #nosec G703 -- CLI tool checks blank file existence
			return blankPath, nil
		}

		// Then try searching in subdirectories
		var foundPath string
		_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error { // #nosec G703 -- Intentional directory traversal for blank discovery
			if err != nil {
				return nil
			}

			if !d.IsDir() && strings.HasSuffix(path, ".md") {
				fileName := strings.TrimSuffix(filepath.Base(path), ".md")

				// Check if filename matches (with or without category prefix)
				if fileName == name {
					foundPath = path
					return filepath.SkipAll // Stop walking
				}

				// Check if category/filename matches
				relPath, _ := filepath.Rel(dir, path)
				pathParts := strings.Split(filepath.Dir(relPath), string(filepath.Separator))
				if len(pathParts) > 0 && pathParts[0] != "." {
					category := pathParts[0]
					if category+"/"+fileName == name {
						foundPath = path
						return filepath.SkipAll // Stop walking
					}
				}
			}

			return nil
		})

		if foundPath != "" {
			return foundPath, nil
		}
	}

	return "", fmt.Errorf("mold %s not found", name)
}

func runGetMold(_ *cobra.Command, args []string) error {
	ref := args[0]

	if !foundry.IsRemoteReference(ref) {
		return fmt.Errorf("expected a remote reference (e.g. github.com/owner/repo), got %q", ref)
	}

	fmt.Println(styles.WorkingBanner("Downloading mold..."))
	fmt.Println()

	parsed, err := foundry.ParseReference(ref)
	if err != nil {
		return fmt.Errorf("parsing reference: %w", err)
	}

	fsys, err := foundry.Resolve(ref)
	if err != nil {
		return fmt.Errorf("resolving mold: %w", err)
	}

	// Validate mold.yaml exists and is parseable.
	if _, err := fs.ReadFile(fsys, "mold.yaml"); err != nil {
		return fmt.Errorf("mold.yaml not found in %s: %w", ref, err)
	}

	manifest, err := mold.LoadMoldFromFS(fsys, "mold.yaml")
	if err != nil {
		return fmt.Errorf("invalid mold manifest: %w", err)
	}

	// Print cache path.
	cacheDir, err := foundry.CacheDir()
	if err != nil {
		return fmt.Errorf("determining cache directory: %w", err)
	}

	// Read lock to get resolved version for path display.
	lock, _ := foundry.ReadLockFile(foundry.LockFileName)
	version := "latest"
	if entry := lock.FindEntry(parsed.CacheKey()); entry != nil {
		version = entry.Version
	}

	cachePath := foundry.VersionDir(cacheDir, parsed, version)
	if parsed.Subpath != "" {
		cachePath = filepath.Join(cachePath, parsed.Subpath)
	}

	fmt.Println(styles.SuccessStyle.Render("Downloaded: ") + styles.AccentStyle.Render(manifest.Name+" "+manifest.Version))
	if manifest.Description != "" {
		fmt.Println(styles.SubtleStyle.Render("  " + manifest.Description))
	}
	fmt.Println(styles.InfoStyle.Render("Cache path: ") + styles.CodeStyle.Render(cachePath))

	return nil
}

// loadInstalledDirs reads .ailloy/state.yaml to find where blanks are installed.
// Falls back to empty lists when no state file exists.
func loadInstalledDirs() (blankDirs, workflowDirs []string) {
	data, err := os.ReadFile(".ailloy/state.yaml") // #nosec G304 -- fixed path in project directory
	if err == nil {
		var state installState
		if yaml.Unmarshal(data, &state) == nil && len(state.BlankDirs) > 0 {
			return state.BlankDirs, state.WorkflowDirs
		}
	}
	// Fallback: default workflow dir only
	return nil, []string{".github/workflows"}
}

// getMoldIcon returns an appropriate icon based on mold name
func getMoldIcon(moldName string) string {
	switch {
	case strings.Contains(moldName, "brainstorm"):
		return "💡"
	case strings.Contains(moldName, "agent"):
		return "🤖"
	case strings.Contains(moldName, "issue"):
		return "🎯"
	case strings.Contains(moldName, "pr"):
		return "🔄"
	case strings.Contains(moldName, "review"):
		return "👀"
	case strings.Contains(moldName, "comment"):
		return "💬"
	case strings.Contains(moldName, "preflight"):
		return "✈️"
	case strings.Contains(moldName, "update"):
		return "🔧"
	default:
		return "📋"
	}
}
