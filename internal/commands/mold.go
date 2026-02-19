package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/nimble-giant/ailloy/pkg/config"
	"github.com/nimble-giant/ailloy/pkg/styles"
	"github.com/spf13/cobra"
)

var moldCmd = &cobra.Command{
	Use:   "mold",
	Short: "Work with Ailloy molds (templates)",
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

func init() {
	rootCmd.AddCommand(moldCmd)
	moldCmd.AddCommand(listMoldsCmd)
	moldCmd.AddCommand(showMoldCmd)

	// Bidirectional: "show mold <name>" also works
	rootCmd.AddCommand(showCmd)
	showCmd.AddCommand(showMoldSubCmd)
}

func runListMolds(cmd *cobra.Command, args []string) error {
	// Load config to get ignore patterns
	cfg, _ := config.LoadConfig(false) // Ignore errors, use empty config if not found

	moldDirs := []string{
		".claude/commands",         // Project commands directory (created by cast)
		".claude/skills",           // Project skills directory (created by cast)
		"commands",                 // Legacy project commands directory
		"templates/claude",         // Source templates directory
		".ailloy/templates/claude", // Legacy project templates
		filepath.Join(os.Getenv("HOME"), ".ailloy/templates/claude"), // Global templates
	}

	workflowDirs := []string{
		".github/workflows", // Project workflows directory
	}

	// Header with inquisitive fox for exploring molds
	header := lipgloss.JoinVertical(
		lipgloss.Center,
		styles.FoxArt("inquisitive"),
		styles.HeaderStyle.Render("Available Claude Code Molds"),
	)
	fmt.Println(header)
	fmt.Println()

	foundMolds := false

	for _, dir := range moldDirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) { // #nosec G703 -- CLI tool intentionally accesses user-specified template directories
			continue
		}

		// Walk through subdirectories to find templates
		err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error { // #nosec G703 -- Intentional directory traversal for template discovery
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
				templateName := strings.TrimSuffix(fileName, ".md")

				// Check if this template should be ignored
				if isIgnored(fileName, templateName, category+"/"+templateName, cfg.Templates.Ignore) {
					return nil
				}

				// Try to extract the first line as description
				content, err := os.ReadFile(path) // #nosec G304 -- CLI tool reads user template files
				if err != nil {
					errorMsg := styles.ErrorStyle.Render("❌ ") +
						styles.AccentStyle.Render(category+"/"+templateName) +
						styles.SubtleStyle.Render(" (unreadable)")
					fmt.Println("  " + errorMsg)
					return nil
				}

				lines := strings.Split(string(content), "\n")
				var description string
				if len(lines) > 0 && strings.HasPrefix(lines[0], "# ") {
					description = strings.TrimPrefix(lines[0], "# ")
				} else {
					description = "Claude Code template"
				}

				// Style the template listing
				icon := getMoldIcon(templateName)
				templateDisplay := styles.SuccessStyle.Render(icon+" ") +
					styles.AccentStyle.Render(category+"/"+templateName) +
					styles.SubtleStyle.Render(" - "+description)
				fmt.Println("  " + templateDisplay)
				foundMolds = true
			}

			return nil
		})

		if err != nil {
			continue
		}
	}

	// List workflow templates
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
				templateName := strings.TrimSuffix(fileName, ".yml")

				// Check if this workflow should be ignored
				if isIgnored(fileName, templateName, "workflows/"+templateName, cfg.Templates.Ignore) {
					return nil
				}

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

				icon := getMoldIcon(templateName)
				templateDisplay := styles.SuccessStyle.Render(icon+" ") +
					styles.AccentStyle.Render("workflows/"+templateName) +
					styles.SubtleStyle.Render(" - "+description)
				fmt.Println("  " + templateDisplay)
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
	content, err := os.ReadFile(moldPath) // #nosec G304 -- CLI tool reads user template files
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
	moldDirs := []string{
		".claude/commands",         // Project commands directory (created by cast)
		".claude/skills",           // Project skills directory (created by cast)
		"commands",                 // Legacy project commands directory
		"templates/claude",         // Source templates directory
		".ailloy/templates/claude", // Legacy project templates
		filepath.Join(os.Getenv("HOME"), ".ailloy/templates/claude"), // Global templates
	}

	for _, dir := range moldDirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) { // #nosec G703 -- CLI tool checks template directory existence
			continue
		}

		// First try direct path (for backward compatibility)
		templatePath := filepath.Join(dir, name+".md")
		if _, err := os.Stat(templatePath); err == nil { // #nosec G703 -- CLI tool checks template file existence
			return templatePath, nil
		}

		// Then try searching in subdirectories
		var foundPath string
		_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error { // #nosec G703 -- Intentional directory traversal for template discovery
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

// isIgnored checks if a template should be ignored based on the ignore patterns
func isIgnored(fileName, templateName, fullPath string, patterns []string) bool {
	for _, pattern := range patterns {
		// Match against filename (e.g., "ci.yml")
		if pattern == fileName {
			return true
		}
		// Match against template name without extension (e.g., "ci")
		if pattern == templateName {
			return true
		}
		// Match against full path (e.g., "workflows/ci")
		if pattern == fullPath {
			return true
		}
	}
	return false
}

// getMoldIcon returns an appropriate icon based on mold name
func getMoldIcon(moldName string) string {
	switch {
	case strings.Contains(moldName, "brainstorm"):
		return "💡"
	case strings.Contains(moldName, "claude-code"):
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
