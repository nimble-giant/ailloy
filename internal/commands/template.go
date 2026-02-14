package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/nimble-giant/ailloy/pkg/styles"
	"github.com/spf13/cobra"
)

var templateCmd = &cobra.Command{
	Use:   "template",
	Short: "Work with Ailloy templates",
	Long: `Commands for managing and executing Ailloy templates.
	
Templates are Markdown files that define AI commands and workflows.`,
}

var listTemplatesCmd = &cobra.Command{
	Use:   "list",
	Short: "List available templates",
	RunE:  runListTemplates,
}

var showTemplateCmd = &cobra.Command{
	Use:   "show <template-name>",
	Short: "Display a template's content",
	Args:  cobra.ExactArgs(1),
	RunE:  runShowTemplate,
}

func init() {
	rootCmd.AddCommand(templateCmd)
	templateCmd.AddCommand(listTemplatesCmd)
	templateCmd.AddCommand(showTemplateCmd)
}

func runListTemplates(cmd *cobra.Command, args []string) error {
	templateDirs := []string{
		".claude/commands",            // Project commands directory (created by init)
		"commands",                    // Legacy project commands directory
		"templates/claude",            // Source templates directory
		".ailloy/templates/claude",    // Legacy project templates
		filepath.Join(os.Getenv("HOME"), ".ailloy/templates/claude"), // Global templates
	}
	
	// Header with inquisitive fox for exploring templates
	header := lipgloss.JoinVertical(
		lipgloss.Center,
		styles.FoxArt("inquisitive"),
		styles.HeaderStyle.Render("Available Claude Code Templates"),
	)
	fmt.Println(header)
	fmt.Println()
	
	foundTemplates := false
	
	for _, dir := range templateDirs {
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
				icon := getTemplateIcon(templateName)
				templateDisplay := styles.SuccessStyle.Render(icon+" ") + 
					styles.AccentStyle.Render(category+"/"+templateName) + 
					styles.SubtleStyle.Render(" - "+description)
				fmt.Println("  " + templateDisplay)
				foundTemplates = true
			}
			
			return nil
		})
		
		if err != nil {
			continue
		}
	}
	
	if !foundTemplates {
		noTemplatesMsg := styles.InfoBoxStyle.Render(
			styles.InfoStyle.Render("ℹ️  No templates found.\n\n") +
			"Run " + styles.CodeStyle.Render("ailloy init --project") + " to set up templates.",
		)
		fmt.Println(noTemplatesMsg)
	}
	
	return nil
}

func runShowTemplate(cmd *cobra.Command, args []string) error {
	templateName := args[0]
	
	// Find template file
	templatePath, err := findTemplate(templateName)
	if err != nil {
		errorMsg := styles.ErrorBoxStyle.Render(
			styles.ErrorStyle.Render("❌ Template not found: ") + 
			styles.CodeStyle.Render(templateName) + "\n\n" +
			"Run " + styles.CodeStyle.Render("ailloy template list") + " to see available templates.",
		)
		fmt.Println(errorMsg)
		return nil
	}
	
	// Read and display the template content
	content, err := os.ReadFile(templatePath) // #nosec G304 -- CLI tool reads user template files
	if err != nil {
		return fmt.Errorf("failed to read template: %w", err)
	}
	
	// Header with small fox emoji
	icon := getTemplateIcon(templateName)
	header := lipgloss.JoinVertical(
		lipgloss.Center,
		styles.FoxArt("small"),
		styles.HeaderStyle.Render(icon+" Template: "+templateName),
	)
	fmt.Println(header)
	
	// Path info
	pathInfo := styles.SubtleStyle.Render("📁 Path: " + templatePath)
	fmt.Println(pathInfo)
	fmt.Println()
	
	// Content in a styled box
	contentBox := styles.BoxStyle.Render(string(content))
	fmt.Println(contentBox)
	
	return nil
}

func findTemplate(name string) (string, error) {
	templateDirs := []string{
		".claude/commands",            // Project commands directory (created by init)
		"commands",                    // Legacy project commands directory
		"templates/claude",            // Source templates directory
		".ailloy/templates/claude",    // Legacy project templates
		filepath.Join(os.Getenv("HOME"), ".ailloy/templates/claude"), // Global templates
	}
	
	for _, dir := range templateDirs {
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
	
	return "", fmt.Errorf("template %s not found", name)
}

// getTemplateIcon returns an appropriate icon based on template name
func getTemplateIcon(templateName string) string {
	switch {
	case strings.Contains(templateName, "issue"):
		return "🎯"
	case strings.Contains(templateName, "pr"):
		return "🔄"
	case strings.Contains(templateName, "review"):
		return "👀"
	case strings.Contains(templateName, "comment"):
		return "💬"
	case strings.Contains(templateName, "preflight"):
		return "✈️"
	case strings.Contains(templateName, "update"):
		return "🔧"
	default:
		return "📋"
	}
}
