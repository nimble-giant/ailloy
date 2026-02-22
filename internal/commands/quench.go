package commands

import (
	"fmt"

	"github.com/nimble-giant/ailloy/pkg/foundry"
	"github.com/nimble-giant/ailloy/pkg/styles"
	"github.com/spf13/cobra"
)

var quenchCmd = &cobra.Command{
	Use:     "quench",
	Aliases: []string{"lock"},
	Short:   "Pin all dependencies to exact versions",
	Long: `Pin all dependencies to their current exact versions (alias: lock).

Ensures all dependencies in ailloy.lock are pinned to exact version and commit
SHAs. Subsequent cast operations will use only these locked versions until
the lock is updated via recast.`,
	RunE: runQuench,
}

func init() {
	rootCmd.AddCommand(quenchCmd)
}

func runQuench(_ *cobra.Command, _ []string) error {
	lock, err := foundry.ReadLockFile(foundry.LockFileName)
	if err != nil {
		return fmt.Errorf("reading lock file: %w", err)
	}
	if lock == nil || len(lock.Molds) == 0 {
		return fmt.Errorf("no lock file found — run %s first to resolve dependencies", styles.CodeStyle.Render("ailloy cast"))
	}

	fmt.Println(styles.WorkingBanner("Quenching dependencies..."))
	fmt.Println()

	// Verify each entry has exact version and commit pinned.
	allPinned := true
	for _, entry := range lock.Molds {
		if entry.Version == "" || entry.Commit == "" {
			fmt.Printf("  %s %s is missing version or commit pin\n",
				styles.WarningStyle.Render("!"),
				entry.Name,
			)
			allPinned = false
			continue
		}

		commitShort := entry.Commit
		if len(commitShort) > 7 {
			commitShort = commitShort[:7]
		}
		fmt.Printf("  %s  %s @ %s\n",
			styles.FoxBullet(entry.Name),
			styles.CodeStyle.Render(entry.Version),
			styles.CodeStyle.Render(commitShort),
		)
	}

	fmt.Println()

	if !allPinned {
		return fmt.Errorf("some dependencies are not fully pinned — run %s to re-resolve", styles.CodeStyle.Render("ailloy recast"))
	}

	fmt.Printf("%s All %d dependencies are pinned to exact versions.\n",
		styles.SuccessStyle.Render("Locked:"),
		len(lock.Molds),
	)
	fmt.Println()
	fmt.Printf("Run %s to update to newer versions.\n", styles.CodeStyle.Render("ailloy recast"))

	return nil
}
