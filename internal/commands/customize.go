package commands

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/nimble-giant/ailloy/pkg/config"
	"github.com/nimble-giant/ailloy/pkg/styles"
	"github.com/spf13/cobra"
)

var customizeCmd = &cobra.Command{
	Use:   "customize",
	Short: "Customize template variables",
	Long: `Configure team-specific defaults for templates.
	
This command allows you to set variables that will be used to customize
templates during initialization. For example, you can set default board
names, priorities, and other team-specific values.

Interactive mode provides guided setup without forcing any defaults,
ensuring you only configure what's relevant to your team.`,
	RunE: runCustomize,
}

var (
	setVar          []string
	listVars        bool
	deleteVar       string
	globalCustomize bool
)

func init() {
	rootCmd.AddCommand(customizeCmd)

	customizeCmd.Flags().StringArrayVarP(&setVar, "set", "s", nil, "set variable (format: key=value)")
	customizeCmd.Flags().BoolVarP(&listVars, "list", "l", false, "list current variables")
	customizeCmd.Flags().StringVarP(&deleteVar, "delete", "d", "", "delete variable")
	customizeCmd.Flags().BoolVarP(&globalCustomize, "global", "g", false, "customize global configuration")
}

func runCustomize(cmd *cobra.Command, args []string) error {
	// Load existing config
	cfg, err := config.LoadConfig(globalCustomize)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Handle list operation
	if listVars {
		return listTemplateVariables(cfg)
	}

	// Handle delete operation
	if deleteVar != "" {
		return deleteTemplateVariable(cfg, deleteVar)
	}

	// Handle set operations
	if len(setVar) > 0 {
		return setTemplateVariables(cfg, setVar)
	}

	// Interactive mode
	return runInteractiveCustomize(cfg)
}

func listTemplateVariables(cfg *config.Config) error {
	scope := "project"
	if globalCustomize {
		scope = "global"
	}

	// Header with sleepy fox for passive display
	header := lipgloss.JoinVertical(
		lipgloss.Center,
		styles.FoxArt("sleepy"),
		styles.HeaderStyle.Render("Template Variables ("+scope+")"),
	)
	fmt.Println(header)
	fmt.Println()

	if len(cfg.Templates.Variables) == 0 {
		noVarsMsg := styles.InfoBoxStyle.Render(
			styles.InfoStyle.Render("ℹ️  No variables configured.\n\n") +
			"Use " + styles.CodeStyle.Render("ailloy customize") + " to set up variables interactively.",
		)
		fmt.Println(noVarsMsg)
		return nil
	}

	// Create a styled table
	table := styles.NewTable()
	table.Headers("Variable", "Value")
	
	for key, value := range cfg.Templates.Variables {
		table.Row(
			styles.AccentStyle.Render(key),
			styles.CodeStyle.Render(value),
		)
	}
	
	fmt.Println(table.Render())
	return nil
}

func deleteTemplateVariable(cfg *config.Config, key string) error {
	if _, exists := cfg.Templates.Variables[key]; !exists {
		errorMsg := styles.ErrorBoxStyle.Render(
			styles.ErrorStyle.Render("❌ Variable not found: ") + 
			styles.CodeStyle.Render(key),
		)
		fmt.Println(errorMsg)
		return nil
	}

	delete(cfg.Templates.Variables, key)

	if err := config.SaveConfig(cfg, globalCustomize); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	scope := "project"
	if globalCustomize {
		scope = "global"
	}

	successMsg := styles.SuccessStyle.Render("✅ Deleted variable ") + 
		styles.CodeStyle.Render(key) + 
		styles.SuccessStyle.Render(" from "+scope+" configuration")
	fmt.Println(successMsg)
	return nil
}

func setTemplateVariables(cfg *config.Config, variables []string) error {
	for _, variable := range variables {
		parts := strings.SplitN(variable, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid variable format: %s (expected key=value)", variable)
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		if key == "" {
			return fmt.Errorf("variable key cannot be empty")
		}

		cfg.Templates.Variables[key] = value
	}

	if err := config.SaveConfig(cfg, globalCustomize); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	scope := "project"
	if globalCustomize {
		scope = "global"
	}

	successMsg := styles.SuccessStyle.Render("✅ Updated ") + 
		fmt.Sprintf("%d", len(variables)) + 
		styles.SuccessStyle.Render(" variable(s) in "+scope+" configuration")
	fmt.Println(successMsg)
	return nil
}

func runInteractiveCustomize(cfg *config.Config) error {
	scope := "project"
	if globalCustomize {
		scope = "global"
	}

	// Welcome message
	fmt.Println(styles.WorkingBanner("Interactive template customization ("+scope+")"))
	fmt.Println()
	
	description := styles.InfoBoxStyle.Render(
		"This will help you set up team-specific defaults for templates.\n" +
		styles.SubtleStyle.Render("• Press Enter to keep existing values\n") +
		styles.SubtleStyle.Render("• Type new values to update settings\n") +
		styles.SubtleStyle.Render("• Leave blank to skip optional settings"),
	)
	fmt.Println(description)
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)

	// Basic template variables - prompt for essential ones only
	// Don't provide defaults for team-specific values like project IDs
	basicVars := []struct {
		key         string
		description string
		example     string
	}{
		{"default_board", "Default GitHub project board name", "Engineering"},
		{"default_priority", "Default issue priority", "P1"},
		{"default_status", "Default issue status", "Ready"},
		{"organization", "GitHub organization name", "mycompany"},
	}

	fmt.Println(styles.AccentStyle.Render("🔧 Basic template variables:"))
	fmt.Println()
	
	for _, variable := range basicVars {
		current := cfg.Templates.Variables[variable.key]
		
		var prompt string
		if current != "" {
			prompt = styles.InfoStyle.Render(variable.description) + " " + 
				styles.SubtleStyle.Render("["+current+"]: ")
		} else {
			prompt = styles.InfoStyle.Render(variable.description) + " " + 
				styles.SubtleStyle.Render("(e.g., "+variable.example+"): ")
		}
		
		fmt.Print(prompt)
		input, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read input: %w", err)
		}

		input = strings.TrimSpace(input)
		if input != "" {
			cfg.Templates.Variables[variable.key] = input
		}
		// Don't set any defaults - let users decide what they need
	}

	// Advanced GitHub Project API configuration (optional)
	fmt.Print("\nWould you like to configure GitHub Project API integration? (y/N): ")
	advancedInput, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}
	
	if strings.ToLower(strings.TrimSpace(advancedInput)) == "y" {
		fmt.Println("\nAdvanced GitHub Project configuration:")
		fmt.Println("(Run 'gh api graphql' commands to find these values for your project)")
		
		advancedVars := []struct {
			key         string
			description string
		}{
			{"project_id", "GitHub project ID (PVT_...)"},
			{"status_field_id", "Status field ID (PVTSSF_...)"},
			{"priority_field_id", "Priority field ID (PVTSSF_...)"},
			{"iteration_field_id", "Iteration field ID (PVTIF_...)"},
		}
		
		for _, variable := range advancedVars {
			current := cfg.Templates.Variables[variable.key]
			
			var prompt string
			if current != "" {
				prompt = fmt.Sprintf("%s [%s]: ", variable.description, current)
			} else {
				prompt = fmt.Sprintf("%s: ", variable.description)
			}
			
			fmt.Print(prompt)
			input, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("failed to read input: %w", err)
			}

			input = strings.TrimSpace(input)
			if input != "" {
				cfg.Templates.Variables[variable.key] = input
			}
		}
	}

	// Allow setting custom variables
	fmt.Println("\nCustom variables (press Enter when done):")
	for {
		fmt.Print("Variable name: ")
		name, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read input: %w", err)
		}

		name = strings.TrimSpace(name)
		if name == "" {
			break
		}

		fmt.Printf("Value for %s: ", name)
		value, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read input: %w", err)
		}

		value = strings.TrimSpace(value)
		cfg.Templates.Variables[name] = value
	}

	if err := config.SaveConfig(cfg, globalCustomize); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Println()
	successMessage := "Template customization saved to "+scope+" configuration"
	fmt.Println(styles.SuccessBanner(successMessage))
	return nil
}
