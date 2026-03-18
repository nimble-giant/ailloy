package commands

import (
	"fmt"
	"strings"

	"github.com/nimble-giant/ailloy/pkg/assay"
	"github.com/nimble-giant/ailloy/pkg/styles"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage Ailloy project configuration",
}

var allowFieldsCmd = &cobra.Command{
	Use:   "allow-fields <field> [field...]",
	Short: "Allow custom frontmatter fields without lint warnings",
	Long: `Add one or more custom frontmatter fields to the command-frontmatter
allow-list in .ailloyrc.yaml, suppressing "unknown command frontmatter fields"
warnings for those fields in all commands, skills, and rules.

Creates .ailloyrc.yaml with starter defaults if it does not exist yet.

Examples:
  ailloy config allow-fields topic source created updated tags
  ailloy config allow-fields severity resolution_time symptoms`,
	Args: cobra.MinimumNArgs(1),
	RunE: runAllowFields,
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(allowFieldsCmd)
}

func runAllowFields(_ *cobra.Command, args []string) error {
	rootDir, err := assay.FindProjectRoot(".")
	if err != nil {
		return fmt.Errorf("finding project root: %w", err)
	}

	added, err := assay.AddAllowedFrontmatterFields(rootDir, args)
	if err != nil {
		return fmt.Errorf("updating config: %w", err)
	}

	if len(added) == 0 {
		fmt.Println(styles.InfoStyle.Render("No new fields to add — all specified fields are already allowed."))
		return nil
	}

	fmt.Println(styles.SuccessStyle.Render("Added to extra-allowed-fields: ") +
		styles.CodeStyle.Render(strings.Join(added, ", ")))
	fmt.Println(styles.SubtleStyle.Render("Saved to .ailloyrc.yaml"))
	return nil
}
