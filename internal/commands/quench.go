package commands

import (
	"fmt"
	"strings"
	"time"

	"github.com/nimble-giant/ailloy/internal/tui/ceremony"
	"github.com/nimble-giant/ailloy/pkg/foundry"
	"github.com/nimble-giant/ailloy/pkg/styles"
	"github.com/spf13/cobra"
)

var quenchCmd = &cobra.Command{
	Use:     "quench [reference]",
	Aliases: []string{"lock"},
	Short:   "Pin installed molds to exact versions (creates ailloy.lock)",
	Long: `Pin installed molds to exact versions and verify integrity (alias: lock).

Creates or refreshes ailloy.lock by pinning each mold in .ailloy/installed.yaml
to its current resolved version and commit. Once ailloy.lock exists, subsequent
'cast', 'ingot add', and 'recast' will keep it in sync automatically.

Pass a reference to quench a single mold. Use --verify to check existing pins
without writing.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runQuench,
}

var (
	quenchVerify bool
	quenchGlobal bool
)

func init() {
	rootCmd.AddCommand(quenchCmd)
	quenchCmd.Flags().BoolVar(&quenchVerify, "verify", false, "check pins without writing (CI-friendly)")
	quenchCmd.Flags().BoolVarP(&quenchGlobal, "global", "g", false, "operate on the global manifest/lock under ~/")
}

func runQuench(_ *cobra.Command, args []string) error {
	manifestPath := manifestPathFor(quenchGlobal)
	lockPath := lockPathFor(quenchGlobal)

	manifest, err := foundry.ReadInstalledManifest(manifestPath)
	if err != nil {
		return fmt.Errorf("reading installed manifest: %w", err)
	}
	if manifest == nil || len(manifest.Molds) == 0 {
		return fmt.Errorf("no installed manifest found at %s — run %s first",
			styles.CodeStyle.Render(manifestPath),
			styles.CodeStyle.Render("ailloy cast"))
	}

	// Read the existing lock up front so we can both verify against it and
	// reject scoped quench when there's nothing to scope into.
	existingLock, _ := foundry.ReadLockFile(lockPath)

	// Filter to a single ref if provided. Scoped quench requires an existing
	// lock — otherwise we'd silently drop every other manifest entry by writing
	// a single-entry lock. Tell the user to run `ailloy quench` first.
	entries := manifest.Molds
	if len(args) == 1 {
		ref, err := foundry.ParseReference(args[0])
		if err != nil {
			return fmt.Errorf("parsing reference: %w", err)
		}
		match := manifest.FindBySource(ref.CacheKey(), ref.Subpath)
		if match == nil {
			return fmt.Errorf("mold %q not found in installed manifest — run %s first",
				args[0],
				styles.CodeStyle.Render("ailloy cast "+args[0]))
		}
		if existingLock == nil {
			return fmt.Errorf("no %s present — run %s (without arguments) first to opt in",
				styles.CodeStyle.Render(lockPath),
				styles.CodeStyle.Render("ailloy quench"))
		}
		entries = []foundry.InstalledEntry{*match}
	}

	if quenchVerify {
		// Verify mode keeps the plain banner — it's a check, not a freeze.
		fmt.Println(styles.WorkingBanner("Verifying pins..."))
		fmt.Println()
	} else {
		ceremony.Open(ceremony.Quench)
	}

	// Verification phase: manifest <-> lock consistency.
	failures := verifyManifestAgainstLock(entries, existingLock)
	for _, msg := range failures {
		fmt.Printf("  %s %s\n", styles.WarningStyle.Render("!"), msg)
	}
	if len(failures) > 0 && quenchVerify {
		fmt.Println()
		return fmt.Errorf("verification failed (%d issue(s))", len(failures))
	}

	if quenchVerify {
		fmt.Println()
		fmt.Println(styles.SuccessStyle.Render("All pins verified."))
		return nil
	}

	// Resolve current versions and write a fresh lock.
	git := foundry.DefaultGitRunner()
	newLock := &foundry.LockFile{APIVersion: "v1"}
	for _, entry := range entries {
		ref, err := referenceFromInstalledEntry(&entry)
		if err != nil {
			fmt.Printf("  %s skipping %s: %v\n", styles.WarningStyle.Render("!"), entry.Name, err)
			continue
		}
		resolved, err := foundry.ResolveVersion(ref, git)
		if err != nil {
			fmt.Printf("  %s skipping %s: %v\n", styles.WarningStyle.Render("!"), entry.Name, err)
			continue
		}
		newLock.UpsertEntry(foundry.LockEntry{
			Name:      entry.Name,
			Source:    entry.Source,
			Version:   resolved.Tag,
			Commit:    resolved.Commit,
			Subpath:   entry.Subpath,
			Timestamp: time.Now().UTC(),
		})
		fmt.Printf("  %s  %s @ %s\n",
			styles.FoxBullet(entry.Name),
			styles.CodeStyle.Render(resolved.Tag),
			styles.CodeStyle.Render(shortSHA(resolved.Commit)),
		)
	}

	// Pin every installed ingot and ore alongside the molds. Artifacts already
	// carry resolved version + commit in the manifest (recorded at install
	// time) so we don't re-resolve here. Scoped quench (single mold ref) still
	// pins all artifacts because they're shared infrastructure: skipping them
	// would leave the lock partial and mid-cycle the next `cast` would have to
	// re-pin them anyway.
	for _, ig := range manifest.Ingots {
		newLock.UpsertArtifactLock("ingot", artifactToLock(ig))
	}
	for _, or := range manifest.Ores {
		newLock.UpsertArtifactLock("ore", artifactToLock(or))
	}

	// If quench was scoped to a single ref AND a lock already exists, merge into it.
	if len(args) == 1 && existingLock != nil {
		for _, e := range newLock.Molds {
			existingLock.UpsertEntry(e)
		}
		for _, e := range newLock.Ingots {
			existingLock.UpsertArtifactLock("ingot", e)
		}
		for _, e := range newLock.Ores {
			existingLock.UpsertArtifactLock("ore", e)
		}
		newLock = existingLock
	}

	if err := foundry.WriteLockFile(lockPath, newLock); err != nil {
		return fmt.Errorf("writing lock file: %w", err)
	}

	fmt.Println()
	fmt.Printf("%s %d dependencies pinned in %s.\n",
		styles.SuccessStyle.Render("Locked:"),
		len(entries),
		styles.CodeStyle.Render(lockPath),
	)
	fmt.Printf("Run %s to update to newer versions.\n", styles.CodeStyle.Render("ailloy recast"))
	ceremony.Stamp(ceremony.Quench, fmt.Sprintf("%d dependency pin(s) frozen", len(entries)))
	return nil
}

// verifyManifestAgainstLock returns human-readable failure messages.
func verifyManifestAgainstLock(entries []foundry.InstalledEntry, lock *foundry.LockFile) []string {
	var failures []string
	if lock == nil {
		return nil // no lock to verify against — first quench scenario
	}
	for _, entry := range entries {
		locked := lock.FindEntry(entry.Source, entry.Subpath)
		if locked == nil {
			failures = append(failures, fmt.Sprintf("%s is in installed manifest but missing from lock", entry.Name))
			continue
		}
		if locked.Version == "" || locked.Commit == "" {
			failures = append(failures, fmt.Sprintf("%s lock entry is missing version or commit", entry.Name))
		} else if locked.Commit != entry.Commit {
			failures = append(failures,
				fmt.Sprintf("%s commit drift: manifest=%s lock=%s",
					entry.Name,
					shortSHA(entry.Commit),
					shortSHA(locked.Commit),
				))
		}
	}
	return failures
}

func shortSHA(s string) string {
	if len(s) > 7 {
		return s[:7]
	}
	return s
}

// referenceFromInstalledEntry reconstructs a Latest-typed Reference for re-resolution.
func referenceFromInstalledEntry(entry *foundry.InstalledEntry) (*foundry.Reference, error) {
	parts := strings.SplitN(entry.Source, "/", 3)
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid source %q: expected host/owner/repo", entry.Source)
	}
	return &foundry.Reference{
		Host:    parts[0],
		Owner:   parts[1],
		Repo:    parts[2],
		Subpath: entry.Subpath,
		Type:    foundry.Latest,
	}, nil
}
