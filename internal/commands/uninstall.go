package commands

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/nimble-giant/ailloy/pkg/foundry"
	"github.com/nimble-giant/ailloy/pkg/styles"
	"github.com/spf13/cobra"
)

var (
	uninstallGlobal bool
	uninstallForce  bool
	uninstallDryRun bool
)

var uninstallCmd = &cobra.Command{
	Use:   "uninstall <source>",
	Short: "Remove a casted mold from this project (or ~/ with -g)",
	Long: `Uninstall a previously casted mold by source identifier (e.g. github.com/owner/repo).

Removes the files listed in the mold's lockfile entry, prunes any empty
directories, and drops the entry from ailloy.lock.

Files modified since they were cast are retained unless --force is given.
Files claimed by another casted mold are retained automatically.`,
	Args: cobra.ExactArgs(1),
	RunE: runUninstall,
}

func init() {
	rootCmd.AddCommand(uninstallCmd)
	uninstallCmd.Flags().BoolVarP(&uninstallGlobal, "global", "g", false, "operate on the global lockfile (~/ailloy.lock)")
	uninstallCmd.Flags().BoolVar(&uninstallForce, "force", false, "delete files even if modified since cast")
	uninstallCmd.Flags().BoolVar(&uninstallDryRun, "dry-run", false, "print what would be removed without touching disk")
}

func runUninstall(_ *cobra.Command, args []string) error {
	source := args[0]

	lockPath, err := uninstallLockPath(uninstallGlobal)
	if err != nil {
		return err
	}

	res, err := foundry.UninstallMold(lockPath, source, foundry.UninstallOptions{
		Force:  uninstallForce,
		DryRun: uninstallDryRun,
	})
	if err != nil {
		if errors.Is(err, foundry.ErrLegacyEntry) {
			fmt.Println(styles.WarningStyle.Render("⚠️  ") + err.Error())
			fmt.Println(styles.SubtleStyle.Render("    Run `ailloy cast " + source + "` to backfill the manifest, then retry."))
			return nil
		}
		return err
	}

	header := "Uninstalled"
	if uninstallDryRun {
		header = "Would uninstall (dry-run)"
	}
	fmt.Println(styles.SuccessStyle.Render(header+" ") + styles.AccentStyle.Render(source))

	if len(res.Deleted) > 0 {
		fmt.Println(styles.SubtleStyle.Render(fmt.Sprintf("  Removed:  %d file(s)", len(res.Deleted))))
		for _, f := range res.Deleted {
			fmt.Println(styles.SubtleStyle.Render("    - " + f))
		}
	}
	if len(res.SkippedModified) > 0 {
		fmt.Println(styles.WarningStyle.Render(fmt.Sprintf("  Skipped (modified): %d file(s)", len(res.SkippedModified))))
		for _, f := range res.SkippedModified {
			fmt.Println(styles.SubtleStyle.Render("    - " + f))
		}
		fmt.Println(styles.SubtleStyle.Render("  Re-run with --force to override."))
	}
	if len(res.Retained) > 0 {
		fmt.Println(styles.InfoStyle.Render(fmt.Sprintf("  Retained (claimed by another mold): %d file(s)", len(res.Retained))))
		for _, f := range res.Retained {
			fmt.Println(styles.SubtleStyle.Render("    - " + f))
		}
	}
	if len(res.NotFound) > 0 {
		fmt.Println(styles.SubtleStyle.Render(fmt.Sprintf("  Already absent: %d file(s)", len(res.NotFound))))
	}
	return nil
}

func uninstallLockPath(global bool) (string, error) {
	if !global {
		return foundry.LockFileName, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	return filepath.Join(home, foundry.LockFileName), nil
}
