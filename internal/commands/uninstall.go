package commands

import (
	"errors"
	"fmt"
	"strings"

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
	Use:   "uninstall <source[//subpath]>",
	Short: "Remove a casted mold from this project (or ~/ with -g)",
	Long: `Uninstall a previously casted mold by source identifier (e.g. github.com/owner/repo).

Removes the files listed in the mold's installed manifest entry, prunes any
empty directories, and drops the entry from .ailloy/installed.yaml. If
ailloy.lock exists alongside the manifest, the matching lock entry is also
removed.

A foundry repo may host multiple molds at different subpaths. Pass the full
ref (e.g. github.com/owner/repo//molds/shortcut) to disambiguate. Bare
sources are accepted when only one matching entry exists.

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
	manifestPath := manifestPathFor(uninstallGlobal)
	if manifestPath == "" {
		return fmt.Errorf("cannot determine installed manifest path")
	}

	source, subpath, err := resolveUninstallTarget(manifestPath, args[0])
	if err != nil {
		return err
	}

	res, err := foundry.UninstallMold(manifestPath, source, subpath, foundry.UninstallOptions{
		Force:  uninstallForce,
		DryRun: uninstallDryRun,
	})
	display := source
	if subpath != "" {
		display = source + "//" + subpath
	}
	// Cascade-prune unshared ingots/ores once the mold has been removed.
	// moldKey mirrors the cast-time key (source@subpath when subpath is set).
	if err == nil && !uninstallDryRun {
		moldKey := source
		if subpath != "" {
			moldKey += "@" + subpath
		}
		if cerr := cascadeUninstallArtifacts(manifestPath, moldKey, uninstallGlobal); cerr != nil {
			fmt.Println(styles.WarningStyle.Render("⚠️  ") + "cascade cleanup: " + cerr.Error())
		}
	}
	if err != nil {
		if errors.Is(err, foundry.ErrLegacyEntry) {
			fmt.Println(styles.WarningStyle.Render("⚠️  ") + err.Error())
			fmt.Println(styles.SubtleStyle.Render("    Run `ailloy cast " + display + "` to backfill the manifest, then retry."))
			return nil
		}
		return err
	}

	header := "Uninstalled"
	if uninstallDryRun {
		header = "Would uninstall (dry-run)"
	}
	fmt.Println(styles.SuccessStyle.Render(header+" ") + styles.AccentStyle.Render(display))

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

// resolveUninstallTarget interprets the user's positional argument as a mold
// reference. If the argument carries a //subpath, that wins. Otherwise the
// installed manifest is consulted: a single matching entry is auto-resolved,
// and ambiguous bare sources are rejected with a list so the user can re-run
// with the disambiguating subpath.
func resolveUninstallTarget(manifestPath, raw string) (source, subpath string, err error) {
	ref, perr := foundry.ParseReference(raw)
	if perr != nil {
		// Fall back to treating the raw arg as an opaque source string so
		// pre-existing scripts that pass a bare cache key still work.
		source = raw
	} else {
		source = ref.CacheKey()
		subpath = ref.Subpath
	}

	if subpath != "" {
		return source, subpath, nil
	}

	manifest, mErr := foundry.ReadInstalledManifest(manifestPath)
	if mErr != nil || manifest == nil {
		return source, "", nil
	}
	matches := manifest.FindAllBySource(source)
	switch len(matches) {
	case 0, 1:
		// 0: let UninstallMold report "not found" with its richer message.
		// 1: the bare-source shorthand resolves unambiguously.
		if len(matches) == 1 {
			subpath = matches[0].Subpath
		}
		return source, subpath, nil
	default:
		var lines []string
		for _, m := range matches {
			sp := m.Subpath
			if sp == "" {
				sp = "(no subpath)"
			}
			lines = append(lines, "  - "+source+"//"+sp+" ("+m.Name+")")
		}
		return "", "", fmt.Errorf("multiple installed molds match %q; specify the full ref:\n%s", source, strings.Join(lines, "\n"))
	}
}
