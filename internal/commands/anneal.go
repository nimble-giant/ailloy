package commands

import (
	"fmt"
	"os"

	"github.com/goccy/go-yaml"
	"github.com/nimble-giant/ailloy/pkg/mold"
	"github.com/nimble-giant/ailloy/pkg/styles"
	"github.com/spf13/cobra"
)

var annealCmd = &cobra.Command{
	Use:     "anneal",
	Aliases: []string{"configure"},
	Short:   "Anneal template flux variables",
	Long: `Anneal team-specific flux values for templates (alias: configure).

This command runs an interactive wizard to configure flux variables
(including ore models for GitHub Projects) and writes the result as a
YAML file that can be passed to cast or forge via the -f flag.

Example:
  ailloy anneal -o ore.yaml
  ailloy cast ./nimble-mold -f ore.yaml`,
	RunE: runAnneal,
}

var (
	annealSetVars []string
	annealOutput  string
)

func init() {
	rootCmd.AddCommand(annealCmd)

	annealCmd.Flags().StringArrayVarP(&annealSetVars, "set", "s", nil, "set flux variable (format: key=value)")
	annealCmd.Flags().StringVarP(&annealOutput, "output", "o", "", "write flux YAML to file (default: stdout)")
}

func runAnneal(cmd *cobra.Command, args []string) error {
	// Scripted mode: --set flags
	if len(annealSetVars) > 0 {
		flux := make(map[string]any)
		if err := mold.ApplySetOverrides(flux, annealSetVars); err != nil {
			return err
		}
		return writeFluxOutput(flux)
	}

	// Interactive mode
	flux := make(map[string]any)
	if err := runWizardAnneal(flux); err != nil {
		return err
	}
	return nil
}

// writeFluxOutput marshals flux to YAML and writes to --output path or stdout.
func writeFluxOutput(flux map[string]any) error {
	data, err := yaml.Marshal(flux)
	if err != nil {
		return fmt.Errorf("failed to marshal flux: %w", err)
	}

	if annealOutput == "" {
		fmt.Print(string(data))
		return nil
	}

	if err := os.WriteFile(annealOutput, data, 0600); err != nil {
		return fmt.Errorf("failed to write flux file: %w", err)
	}

	fmt.Println(styles.SuccessStyle.Render("✅ Wrote flux values to ") + styles.CodeStyle.Render(annealOutput))
	return nil
}
