package commands

import (
	"fmt"
	"log"
	"slices"
	"strings"

	"github.com/nimble-giant/ailloy/internal/tui/ceremony"
	"github.com/nimble-giant/ailloy/pkg/blanks"
	"github.com/nimble-giant/ailloy/pkg/foundry"
	"github.com/nimble-giant/ailloy/pkg/styles"
	"github.com/spf13/cobra"
)

var recastCmd = &cobra.Command{
	Use:     "recast [name]",
	Aliases: []string{"upgrade"},
	Short:   "Re-resolve installed molds to newer versions",
	Long: `Re-resolve installed molds to newer versions and re-render their content (alias: upgrade).

Reads .ailloy/installed.yaml, refreshes each mold to its latest matching
version, and re-runs the cast pipeline so files on disk reflect the new
version. If ailloy.lock exists, it is updated in lockstep.

Flags supplied here layer on top of the options the original cast recorded
in installed.yaml. --set and --values overrides are persisted back; the
recovery flag --force-replace-on-parse-error is not.

Use --dry-run to preview which molds will move, without re-rendering.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runRecast,
}

var (
	recastDryRun        bool
	recastGlobal        bool
	recastSetFlags      []string
	recastValFiles      []string
	recastWithWorkflows bool
	recastForceReplace  bool
	// recastFrozen mirrors --frozen on cast: fail (do not auto-install) on
	// any declared ingot/ore dep that's missing from .ailloy/.
	recastFrozen bool
)

// recastCLIOptions holds the option-shaped flags supplied for THIS recast run.
// Distinct from foundry.CastOptionsRecord (which is the persisted form) so the
// merge algorithm has explicit "recorded" and "this-run" inputs.
type recastCLIOptions struct {
	WithWorkflows            bool
	ValueFiles               []string
	SetOverrides             []string
	ForceReplaceOnParseError bool // run-time only, never persisted
}

// hasOverrides reports whether any persistable CLI flag was supplied. Used by
// the loop to decide whether a same-version mold should still be re-rendered.
// ForceReplaceOnParseError is intentionally excluded — a recovery flag alone
// should not force a re-render of an already-up-to-date mold.
func (o recastCLIOptions) hasOverrides() bool {
	return o.WithWorkflows || len(o.ValueFiles) > 0 || len(o.SetOverrides) > 0
}

// mergeRecastOptions composes the persisted (recorded) options with this run's
// CLI flags. CLI flags layer on top of recorded options:
//
//   - WithWorkflows is OR'd (CLI cannot turn off a recorded true).
//   - ValueFiles: recorded first, CLI appended; dedupe on exact path.
//   - SetOverrides: recorded first, CLI appended; if a CLI override has the
//     same dotted key as a recorded entry, the recorded entry is replaced
//     in place rather than duplicated.
//
// The returned record is what we persist back to the manifest after a
// successful recast. ForceReplaceOnParseError is not part of the result.
func mergeRecastOptions(recorded *foundry.CastOptionsRecord, cli recastCLIOptions) foundry.CastOptionsRecord {
	var rec foundry.CastOptionsRecord
	if recorded != nil {
		rec = *recorded
		rec.ValueFiles = append([]string(nil), recorded.ValueFiles...)
		rec.SetOverrides = append([]string(nil), recorded.SetOverrides...)
	}

	rec.WithWorkflows = rec.WithWorkflows || cli.WithWorkflows

	for _, f := range cli.ValueFiles {
		if !slices.Contains(rec.ValueFiles, f) {
			rec.ValueFiles = append(rec.ValueFiles, f)
		}
	}

	for _, kv := range cli.SetOverrides {
		key := setOverrideKey(kv)
		replaced := false
		for i, existing := range rec.SetOverrides {
			if setOverrideKey(existing) == key {
				// Safe: rec.SetOverrides was cloned above, so this does not
				// alias the caller's recorded slice.
				rec.SetOverrides[i] = kv
				replaced = true
				break
			}
		}
		if !replaced {
			rec.SetOverrides = append(rec.SetOverrides, kv)
		}
	}

	return rec
}

// setOverrideKey returns the LHS of a `key=value` --set string, trimmed of
// whitespace. Mirrors the trim behavior in mold.ApplySetOverrides
// (pkg/mold/flux.go) so dedupe matches what the renderer ultimately sees.
// Inputs without "=" return the entire string trimmed (treated as a single-
// key entry).
func setOverrideKey(kv string) string {
	parts := strings.SplitN(kv, "=", 2)
	return strings.TrimSpace(parts[0])
}

func init() {
	rootCmd.AddCommand(recastCmd)
	recastCmd.Flags().BoolVar(&recastDryRun, "dry-run", false, "preview changes without applying")
	recastCmd.Flags().BoolVarP(&recastGlobal, "global", "g", false, "operate on the global manifest/lock under ~/")
	recastCmd.Flags().BoolVar(&recastWithWorkflows, "with-workflows", false, "include GitHub Actions workflow blanks (OR'd with the recorded value)")
	recastCmd.Flags().StringArrayVar(&recastSetFlags, "set", nil, "override flux variable (key=value, repeatable; supports dotted keys)")
	recastCmd.Flags().StringArrayVarP(&recastValFiles, "values", "f", nil, "flux value file (repeatable; later files override earlier)")
	recastCmd.Flags().BoolVar(&recastForceReplace, "force-replace-on-parse-error", false, "replace unparseable merge-strategy destinations instead of erroring")
	recastCmd.Flags().BoolVar(&recastFrozen, "frozen", false, "fail (do not auto-install) when a declared ingot/ore dep is missing from .ailloy/; intended for CI")
}

type recastChange struct {
	Name           string
	Source         string
	OldVersion     string
	OldCommit      string
	NewVersion     string
	NewCommit      string
	Effective      foundry.CastOptionsRecord // applied options for this run
	OptionsChanged bool                      // true when CLI overrode recorded options
}

func runRecast(cmd *cobra.Command, args []string) error {
	manifestPath := manifestPathFor(recastGlobal)

	manifest, err := foundry.ReadInstalledManifest(manifestPath)
	if err != nil {
		return fmt.Errorf("reading installed manifest: %w", err)
	}
	if manifest == nil || len(manifest.Molds) == 0 {
		return fmt.Errorf("no installed manifest at %s — run %s first",
			styles.CodeStyle.Render(manifestPath),
			styles.CodeStyle.Render("ailloy cast"))
	}

	entries := manifest.Molds
	if len(args) == 1 {
		match := manifest.FindByName(args[0])
		if match == nil {
			return fmt.Errorf("mold %q not found in installed manifest", args[0])
		}
		entries = []foundry.InstalledEntry{*match}
	}

	cli := recastCLIOptions{
		WithWorkflows:            recastWithWorkflows,
		ValueFiles:               recastValFiles,
		SetOverrides:             recastSetFlags,
		ForceReplaceOnParseError: recastForceReplace,
	}

	if recastDryRun {
		fmt.Println(styles.WorkingBanner("Previewing dependency updates (dry run)..."))
		fmt.Println()
	} else {
		ceremony.Open(ceremony.Recast)
	}

	git := foundry.DefaultGitRunner()
	var changes []recastChange
	failures := 0

	for _, entry := range entries {
		ref, refErr := referenceFromInstalledEntry(&entry)
		if refErr != nil {
			fmt.Printf("%s skipping %s: %v\n", styles.WarningStyle.Render("!"), entry.Name, refErr)
			failures++
			continue
		}
		resolved, resolveErr := foundry.ResolveVersion(ref, git)
		if resolveErr != nil {
			fmt.Printf("%s skipping %s: %v\n", styles.WarningStyle.Render("!"), entry.Name, resolveErr)
			failures++
			continue
		}

		versionUnchanged := resolved.Tag == entry.Version && resolved.Commit == entry.Commit
		if versionUnchanged && !cli.hasOverrides() {
			fmt.Println(styles.InfoStyle.Render("  ") + entry.Name + " is already up to date (" +
				styles.CodeStyle.Render(entry.Version) + ")")
			continue
		}

		effective := mergeRecastOptions(entry.CastOptions, cli)
		change := recastChange{
			Name:           entry.Name,
			Source:         entry.Source,
			OldVersion:     entry.Version,
			OldCommit:      entry.Commit,
			NewVersion:     resolved.Tag,
			NewCommit:      resolved.Commit,
			Effective:      effective,
			OptionsChanged: cli.hasOverrides(),
		}

		if recastDryRun {
			changes = append(changes, change)
			continue
		}

		// Build a versioned ref string and re-cast. CastMold owns the
		// installed-manifest update, lock update (when present), state.yaml
		// write, and FileHashes recording — recast does not duplicate any
		// of that work.
		versionedRef := buildVersionedRefString(ref, resolved.Tag)
		castOpts := CastOptions{
			Global:                   recastGlobal,
			WithWorkflows:            effective.WithWorkflows,
			ValueFiles:               effective.ValueFiles,
			SetOverrides:             effective.SetOverrides,
			ForceReplaceOnParseError: cli.ForceReplaceOnParseError,
		}
		if _, castErr := CastMold(cmd.Context(), versionedRef, castOpts); castErr != nil {
			fmt.Printf("%s skipping %s: %v\n", styles.WarningStyle.Render("!"), entry.Name, castErr)
			failures++
			continue
		}

		// Reconcile the freshly resolved mold's dependency graph: install
		// any newly declared deps and prune any that the mold no longer
		// declares. moldKey mirrors the cast-time key (source@subpath when
		// subpath is set) so dependent strings stay consistent. The fetch
		// here hits the foundry cache CastMold just populated, so it's
		// near-free.
		moldKey := entry.Source
		if entry.Subpath != "" {
			moldKey += "@" + entry.Subpath
		}
		if fetcher, ferr := foundry.NewFetcher(git); ferr == nil {
			if fetchedFS, _, fetchErr := fetcher.Fetch(ref, resolved); fetchErr == nil {
				reader := blanks.NewMoldReader(fetchedFS)
				if freshMold, mErr := reader.LoadManifest(); mErr != nil {
					log.Printf("warning: loading fresh mold manifest for %s: %v", entry.Name, mErr)
				} else if freshMold != nil {
					// Auto-install newly declared deps. Recast operates on the
					// project's installed.yaml (or global per --global). Local-path
					// deps are refused because recast walks remote references.
					if err := installDeclaredDeps(freshMold, moldKey, recastGlobal, false, recastFrozen, false, nil); err != nil {
						log.Printf("warning: installing deps for %s: %v", entry.Name, err)
					}
					// Cascade-prune deps the mold no longer declares.
					if err := pruneRemovedDeps(manifestPathFor(recastGlobal), moldKey, freshMold.Dependencies, recastGlobal); err != nil {
						log.Printf("warning: pruning removed deps for %s: %v", entry.Name, err)
					}
				}
			}
		}

		// CastMold has already upserted the manifest entry; now overlay the
		// merged effective options so subsequent recasts replay them.
		if persistErr := persistEffectiveOptions(manifestPath, entry.Source, entry.Subpath, effective); persistErr != nil {
			fmt.Printf("%s warning: %s: failed to persist options: %v\n",
				styles.WarningStyle.Render("!"), entry.Name, persistErr)
			// Non-fatal — files are already updated.
		}

		changes = append(changes, change)
	}

	if len(changes) == 0 && failures == 0 {
		fmt.Println()
		fmt.Println(styles.SuccessStyle.Render("All dependencies are up to date."))
		return nil
	}

	fmt.Println()
	if recastDryRun {
		fmt.Println(styles.InfoStyle.Render("Changes that would be applied:"))
	} else if len(changes) > 0 {
		fmt.Println(styles.SuccessStyle.Render("Updated dependencies:"))
	}
	fmt.Println()
	for _, c := range changes {
		line := fmt.Sprintf("  %s  %s %s %s",
			styles.FoxBullet(c.Name),
			styles.CodeStyle.Render(c.OldVersion),
			styles.InfoStyle.Render("->"),
			styles.CodeStyle.Render(c.NewVersion),
		)
		if c.OptionsChanged {
			line += "  " + styles.InfoStyle.Render("(options overridden)")
		}
		fmt.Println(line)
	}

	if !recastDryRun && len(changes) > 0 {
		fmt.Println()
		fmt.Println(styles.SuccessBanner("Recast complete!"))
		ceremony.Stamp(ceremony.Recast, fmt.Sprintf("%d mold(s) updated", len(changes)))
	}
	if recastDryRun {
		fmt.Println()
		fmt.Println(styles.InfoStyle.Render("Run without --dry-run to apply these changes."))
	}

	if failures > 0 {
		return fmt.Errorf("%d mold(s) failed to recast", failures)
	}
	return nil
}

// buildVersionedRefString turns a Reference plus a resolved tag into the
// canonical ref-string format accepted by CastMold and foundry.ParseReference:
//
//	<host>/<owner>/<repo>@<tag>[//<subpath>]
func buildVersionedRefString(ref *foundry.Reference, tag string) string {
	s := ref.CacheKey()
	if tag != "" {
		s += "@" + tag
	}
	if ref.Subpath != "" {
		s += "//" + ref.Subpath
	}
	return s
}

// persistEffectiveOptions writes the merged CastOptions back onto the manifest
// entry identified by (source, subpath). It re-reads the manifest because
// CastMold has just rewritten it; we layer the options on top of CastMold's
// upsert without otherwise modifying the entry.
func persistEffectiveOptions(manifestPath, source, subpath string, eff foundry.CastOptionsRecord) error {
	manifest, err := foundry.ReadInstalledManifest(manifestPath)
	if err != nil {
		return err
	}
	if manifest == nil {
		return fmt.Errorf("manifest disappeared at %s", manifestPath)
	}
	target := manifest.FindBySource(source, subpath)
	if target == nil {
		return fmt.Errorf("entry %s/%s not found after recast", source, subpath)
	}
	if eff.WithWorkflows || len(eff.ValueFiles) > 0 || len(eff.SetOverrides) > 0 {
		copied := eff
		copied.ValueFiles = append([]string(nil), eff.ValueFiles...)
		copied.SetOverrides = append([]string(nil), eff.SetOverrides...)
		target.CastOptions = &copied
	} else {
		target.CastOptions = nil
	}
	return foundry.WriteInstalledManifest(manifestPath, manifest)
}
