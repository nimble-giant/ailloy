package commands

import (
	"fmt"

	"github.com/nimble-giant/ailloy/pkg/selfupdate"
	"github.com/nimble-giant/ailloy/pkg/styles"
	"github.com/spf13/cobra"
)

var updateCheckOnly bool

var updateCmd = &cobra.Command{
	Use:     "update",
	Aliases: []string{"self-update", "upgrade"},
	Short:   "Update ailloy to the latest version",
	Long: `Check for and install the latest version of ailloy.

By default the command downloads and replaces the current binary.
Use --check to only check whether a newer version is available without updating.`,
	RunE: runUpdate,
}

func init() {
	rootCmd.AddCommand(updateCmd)
	updateCmd.Flags().BoolVar(&updateCheckOnly, "check", false, "only check for updates, do not install")
}

func runUpdate(_ *cobra.Command, _ []string) error {
	current := RawVersion()

	fmt.Println(styles.WorkingBanner("Checking for updates..."))
	fmt.Println()

	result, err := selfupdate.Check(current)
	if err != nil {
		return fmt.Errorf("update check failed: %w", err)
	}

	if result.DevBuild {
		fmt.Println(styles.WarningStyle.Render("Running a development build — cannot determine if an update is available."))
		fmt.Println(styles.InfoStyle.Render("Latest release: " + result.Latest))
		fmt.Println(styles.SubtleStyle.Render("Install a released version to enable update checks."))
		return nil
	}

	if result.UpToDate {
		fmt.Println(styles.SuccessStyle.Render("Already up to date! ") + styles.SubtleStyle.Render("("+current+")"))
		return nil
	}

	// An update is available.
	fmt.Println(styles.InfoStyle.Render("Current version: ") + styles.SubtleStyle.Render(result.Current))
	fmt.Println(styles.AccentStyle.Render("Latest version:  ") + result.Latest)
	fmt.Println()

	if updateCheckOnly {
		fmt.Println(styles.WarningStyle.Render("Update available! ") + styles.SubtleStyle.Render("Run 'ailloy update' to install."))
		return nil
	}

	fmt.Println(styles.WorkingBanner("Downloading " + result.Latest + "..."))
	fmt.Println()

	if err := selfupdate.Update(result.Release); err != nil {
		return fmt.Errorf("update failed: %w", err)
	}

	fmt.Println(styles.SuccessBanner("Updated to " + result.Latest))
	return nil
}
