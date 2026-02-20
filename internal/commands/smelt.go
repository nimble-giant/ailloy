package commands

import (
	"fmt"

	"github.com/nimble-giant/ailloy/pkg/smelt"
	"github.com/nimble-giant/ailloy/pkg/styles"
	"github.com/spf13/cobra"
)

var smeltCmd = &cobra.Command{
	Use:     "smelt",
	Aliases: []string{"package"},
	Short:   "Package a mold into a distributable format",
	Long: `Package a mold into a distributable archive (alias: package).

By default, creates a .tar.gz tarball from the current mold directory.
Use -o binary for self-contained binary output (not yet implemented).`,
	RunE: runSmelt,
}

var (
	smeltOutputFormat string
	smeltOutputPath   string
)

func init() {
	rootCmd.AddCommand(smeltCmd)

	smeltCmd.Flags().StringVarP(&smeltOutputFormat, "output-format", "o", "tar", "output format: tar, binary")
	smeltCmd.Flags().StringVar(&smeltOutputPath, "output", "", "output directory (default: current directory)")
}

func runSmelt(_ *cobra.Command, _ []string) error {
	fmt.Println(styles.WorkingBanner("Smelting mold..."))
	fmt.Println()

	moldDir := "."

	var (
		outputFile string
		size       int64
		err        error
	)

	switch smeltOutputFormat {
	case "tar":
		outputFile, size, err = smelt.PackageTarball(moldDir, smeltOutputPath)
	case "binary":
		outputFile, size, err = smelt.PackageBinary(moldDir, smeltOutputPath)
	default:
		return fmt.Errorf("unknown output format %q (supported: tar, binary)", smeltOutputFormat)
	}

	if err != nil {
		return err
	}

	fmt.Println(styles.SuccessStyle.Render("Smelted: ") + styles.CodeStyle.Render(outputFile) +
		styles.SubtleStyle.Render(fmt.Sprintf(" (%s)", humanSize(size))))
	return nil
}

// humanSize formats a byte count as a human-readable string.
func humanSize(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
