package commands

import (
	"fmt"
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

	// Show models section if any are enabled
	if hasEnabledModels(cfg) {
		fmt.Println()
		fmt.Println(styles.AccentStyle.Render("Semantic Models:"))
		fmt.Println()

		printModelSummary("Status", &cfg.Models.Status)
		printModelSummary("Priority", &cfg.Models.Priority)
		printModelSummary("Iteration", &cfg.Models.Iteration)
	}

	return nil
}

func hasEnabledModels(cfg *config.Config) bool {
	return cfg.Models.Status.Enabled || cfg.Models.Priority.Enabled || cfg.Models.Iteration.Enabled
}

func printModelSummary(name string, model *config.ModelConfig) {
	if !model.Enabled {
		fmt.Println(styles.SubtleStyle.Render("  " + name + ": disabled"))
		return
	}

	status := styles.SuccessStyle.Render("enabled")
	mapping := ""
	if model.FieldMapping != "" {
		mapping = " -> " + styles.CodeStyle.Render(model.FieldMapping)
	}
	fmt.Println("  " + styles.AccentStyle.Render(name) + ": " + status + mapping)

	if len(model.Options) > 0 {
		for concept, opt := range model.Options {
			line := "    " + styles.SubtleStyle.Render(concept) + ": " + styles.CodeStyle.Render(opt.Label)
			if opt.ID != "" {
				line += styles.SubtleStyle.Render(" (" + opt.ID + ")")
			}
			fmt.Println(line)
		}
	}
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
	return runWizardCustomize(cfg)
}
