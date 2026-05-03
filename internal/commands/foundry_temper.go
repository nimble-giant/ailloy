package commands

import (
	"fmt"

	"github.com/nimble-giant/ailloy/pkg/foundry/index"
	"github.com/nimble-giant/ailloy/pkg/styles"
	"github.com/spf13/cobra"
)

var (
	foundryTemperOffline   bool
	foundryTemperNoRecurse bool
)

var foundryTemperCmd = &cobra.Command{
	Use:     "temper [path]",
	Aliases: []string{"validate"},
	Short:   "Validate a foundry index file",
	Long: `Validate a foundry.yaml file.

Checks performed:
  - YAML parse and schema validation (apiVersion, kind, name, required mold/foundry fields)
  - Each direct mold's source resolves on its remote (git ls-remote, no clone)
  - Each nested foundry resolves and is itself valid (recursively, with cycle protection)

Use --offline to skip network checks (schema only). Use --no-recurse to
validate only this index — skipping descent into nested foundries — while
still verifying the immediate mold sources.`,
	Args:         cobra.MaximumNArgs(1),
	SilenceUsage: true,
	RunE:         runFoundryTemper,
}

func init() {
	foundryCmd.AddCommand(foundryTemperCmd)
	foundryTemperCmd.Flags().BoolVar(&foundryTemperOffline, "offline", false, "skip network checks (schema only)")
	foundryTemperCmd.Flags().BoolVar(&foundryTemperNoRecurse, "no-recurse", false, "skip descent into nested foundries")
}

func runFoundryTemper(_ *cobra.Command, args []string) error {
	path := "foundry.yaml"
	if len(args) > 0 {
		path = args[0]
	}

	fmt.Println(styles.WorkingBanner("Tempering " + path + "..."))
	fmt.Println()

	res, err := index.Temper(path, index.TemperOptions{
		Offline:   foundryTemperOffline,
		NoRecurse: foundryTemperNoRecurse,
	})
	if err != nil {
		return err
	}

	if res.Index != nil && res.Index.Name != "" {
		summary := fmt.Sprintf("Index: %s (%d molds, %d nested foundries)",
			res.Index.Name, len(res.Index.Molds), len(res.Index.Foundries))
		fmt.Println(styles.InfoStyle.Render(summary))
		fmt.Println()
	}

	if !res.HasErrors() {
		fmt.Println(styles.SuccessStyle.Render("Foundry index is valid."))
		return nil
	}

	for _, f := range res.Findings {
		header := styles.ErrorStyle.Render("error")
		if f.Path != "" {
			header += " " + styles.AccentStyle.Render(f.Path)
		}
		fmt.Println("  " + header)
		if f.Source != "" {
			fmt.Println(styles.SubtleStyle.Render("    source: " + f.Source))
		}
		fmt.Println(styles.SubtleStyle.Render("    " + f.Err.Error()))
	}

	return fmt.Errorf("%d finding(s) reported", len(res.Findings))
}
