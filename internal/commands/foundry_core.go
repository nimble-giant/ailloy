package commands

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/nimble-giant/ailloy/pkg/foundry"
	"github.com/nimble-giant/ailloy/pkg/foundry/index"
	"github.com/nimble-giant/ailloy/pkg/mold"
)

// splitFluxKeysForMold partitions user-supplied --set keys into those declared
// by the mold's flux schema and those that aren't. Returns nil/nil when there
// are no overrides. The mold's schema is loaded from flux.schema.yaml at
// source, and (for local-path sources) merged with any inline `flux:` block
// declared in mold.yaml. Remote refs only see the flux.schema.yaml entries —
// merging both for remote sources would require pkg/mold changes and is filed
// as a follow-up.
func splitFluxKeysForMold(ctx context.Context, source string, setOverrides []string) (applied, skipped []string) {
	if len(setOverrides) == 0 {
		return nil, nil
	}
	keys := make([]string, 0, len(setOverrides))
	for _, kv := range setOverrides {
		if eq := strings.IndexByte(kv, '='); eq != -1 {
			keys = append(keys, kv[:eq])
		}
	}
	// Schema fetch tolerates local dirs and remote refs. A real fetch error
	// is distinct from "no schema declared" — log the former so operators
	// can diagnose, but fall through with an empty schema so any inline
	// mold.yaml flux still gets a chance to declare keys.
	schema, _, err := mold.FetchSchemaFromSource(ctx, source)
	if err != nil {
		log.Printf("warning: fetch flux schema for %s: %v", source, err)
		schema = nil
	}
	declared := map[string]struct{}{}
	for _, v := range schema {
		declared[v.Name] = struct{}{}
	}
	// Inline flux declarations in mold.yaml — only readable for local-path
	// sources. Remote refs go through FetchSchemaFromSource which would need
	// pkg/mold changes to merge inline schema at fetch time.
	if info, serr := os.Stat(source); serr == nil && info.IsDir() {
		if m, lerr := mold.LoadMold(filepath.Join(source, "mold.yaml")); lerr == nil && m != nil {
			for _, v := range m.Flux {
				declared[v.Name] = struct{}{}
			}
		}
	}
	for _, k := range keys {
		if _, ok := declared[k]; ok {
			applied = append(applied, k)
		} else {
			skipped = append(skipped, k)
		}
	}
	return applied, skipped
}

// formatFoundryFluxSummary renders a per-key apply/skip report based on the
// per-mold results returned by InstallFoundryCore. Empty when keys is empty.
func formatFoundryFluxSummary(reports []InstallFoundryReport, keys []string) string {
	if len(keys) == 0 || len(reports) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("Applied flux overrides:\n")
	for _, k := range keys {
		var skipped []string
		applied := 0
		for _, r := range reports {
			if slices.Contains(r.FluxSkipped, k) {
				skipped = append(skipped, r.Name)
				continue
			}
			applied++
		}
		fmt.Fprintf(&b, "  %s → %d/%d molds", k, applied, len(reports))
		if len(skipped) > 0 {
			fmt.Fprintf(&b, " (skipped: %s — key not in schema)", strings.Join(skipped, ", "))
		}
		b.WriteString("\n")
	}
	return b.String()
}

// AddFoundryResult reports the outcome of an add operation.
type AddFoundryResult struct {
	Entry         index.FoundryEntry
	AlreadyExists bool
	MoldCount     int
}

// AddFoundryCore registers a foundry URL into cfg, fetches its index, and
// returns the result. cfg is mutated in place; the caller is responsible
// for SaveConfig if AlreadyExists=false.
func AddFoundryCore(cfg *index.Config, url string) (AddFoundryResult, error) {
	var res AddFoundryResult

	url = index.NormalizeFoundryURL(url)

	if existing := cfg.FindFoundry(url); existing != nil {
		res.Entry = *existing
		res.AlreadyExists = true
		return res, nil
	}

	entry := index.FoundryEntry{
		Name:   nameFromFoundryURL(url),
		URL:    url,
		Type:   index.DetectType(url),
		Status: "pending",
	}

	git := defaultGitRunner()
	fetcher, err := index.NewFetcher(git)
	if err != nil {
		return res, err
	}
	idx, err := fetcher.FetchIndex(&entry)
	if err != nil {
		return res, fmt.Errorf("fetching foundry index: %w", err)
	}

	cfg.AddFoundry(entry)
	res.Entry = entry
	res.MoldCount = len(idx.Molds)
	return res, nil
}

// ErrCannotRemoveDefault is returned when the user tries to remove the
// virtual official foundry (which isn't persisted in cfg).
var ErrCannotRemoveDefault = errors.New("cannot remove the default verified foundry; it is built in")

// RemoveFoundryCore removes a foundry by name or URL. Returns the removed
// entry or an error.
func RemoveFoundryCore(cfg *index.Config, nameOrURL string) (index.FoundryEntry, error) {
	entry := cfg.FindFoundry(nameOrURL)
	if entry == nil {
		official := index.OfficialFoundryEntry()
		if strings.EqualFold(nameOrURL, official.Name) || index.IsOfficialFoundry(nameOrURL) {
			return index.FoundryEntry{}, ErrCannotRemoveDefault
		}
		return index.FoundryEntry{}, fmt.Errorf("foundry %q not found", nameOrURL)
	}
	removed := *entry
	cfg.RemoveFoundry(nameOrURL)
	return removed, nil
}

// UpdateFoundryReport is one row of an UpdateFoundriesCore result.
type UpdateFoundryReport struct {
	Name      string
	URL       string
	MoldCount int
	Persisted bool // false for the virtual default
	Err       error
	Warnings  []index.ResolutionWarning
}

// UpdateFoundriesCore fetches every effective foundry, persists status/timestamp
// for entries originating from cfg, and returns a per-foundry report.
func UpdateFoundriesCore(cfg *index.Config) ([]UpdateFoundryReport, error) {
	git := defaultGitRunner()
	fetcher, err := index.NewFetcher(git)
	if err != nil {
		return nil, err
	}

	type target struct {
		entry     *index.FoundryEntry
		persisted bool
	}
	var targets []target
	for i := range cfg.Foundries {
		targets = append(targets, target{entry: &cfg.Foundries[i], persisted: true})
	}
	if !cfg.HasOfficialFoundry() {
		official := index.OfficialFoundryEntry()
		targets = append([]target{{entry: &official, persisted: false}}, targets...)
	}

	// Network-only lookup. Fetcher writes each fetched index to disk on
	// success, so resolving the tree also populates the on-disk cache that
	// `foundry search` reads from.
	networkLookup := func(source string) (*index.Index, error) {
		url := index.NormalizeFoundryURL(source)
		entry := &index.FoundryEntry{URL: url, Type: index.DetectType(url)}
		return fetcher.FetchIndex(entry)
	}

	out := make([]UpdateFoundryReport, 0, len(targets))
	for _, t := range targets {
		report := UpdateFoundryReport{Name: t.entry.Name, URL: t.entry.URL, Persisted: t.persisted}

		r := index.NewResolver(networkLookup)
		root, _, rerr := r.Resolve(t.entry.URL)
		if rerr != nil {
			report.Err = rerr
			out = append(out, report)
			continue
		}
		report.MoldCount = len(root.Index.Molds)
		report.Warnings = r.Warnings()

		// Mirror the metadata Fetcher would have set on a direct call.
		t.entry.LastUpdated = time.Now().UTC()
		t.entry.Status = "ok"

		out = append(out, report)
	}
	return out, nil
}

// InstallFoundryOptions controls a bulk install across every mold in a foundry.
type InstallFoundryOptions struct {
	Global        bool // pass --global to each cast
	WithWorkflows bool // include .github/ workflow blanks
	DryRun        bool // report what would be installed; don't touch disk
	Force         bool // re-cast even if already installed in the target lockfile
	ClaudePlugin  bool // package each mold as a Claude Code plugin
	Shallow       bool // install only the root foundry's molds; skip nested foundries
	// ValueFiles applies the same -f/--values layering used by `cast` to
	// every mold installed by this run. Files are loaded left-to-right;
	// later files override earlier.
	ValueFiles []string
	// SetOverrides applies the same --set layering used by `cast` to every
	// mold. Same precedence as on `cast`: highest layer.
	SetOverrides []string
}

// InstallFoundryReport is the per-mold result of an InstallFoundryCore run.
type InstallFoundryReport struct {
	Name    string
	Source  string
	Foundry string   // owning foundry name (e.g., "replicated")
	Chain   []string // resolution chain from root to owning foundry, excluding owner ([] when root-owned)
	Skipped bool     // true when already installed and !Force
	Err     error    // non-nil if cast failed for this mold
	Version string   // populated on success or skip (from CastResult / lockfile)
	// FluxApplied lists user-supplied --set keys that are declared in this
	// mold's flux schema. FluxSkipped lists keys that are not.
	FluxApplied []string
	FluxSkipped []string
}

// ErrFoundryNotFound is returned when nameOrURL doesn't match any effective
// foundry in cfg.
var ErrFoundryNotFound = errors.New("foundry not found")

// InstallFoundryCore casts every mold listed in the named foundry's cached
// index. The foundry lookup is by name or URL against cfg.EffectiveFoundries(),
// so the verified built-in default works without a prior `foundry add`.
//
// Molds already present in the target lockfile are skipped unless
// opts.Force. The first git fetch may take a while; subsequent calls reuse
// the cache.
//
// The returned warnings carry non-fatal sub-foundry resolution failures
// (e.g. private nested foundries the caller cannot access). Molds from those
// inaccessible sub-foundries are silently absent from the report; warnings
// give the CLI a chance to explain why.
func InstallFoundryCore(ctx context.Context, cfg *index.Config, nameOrURL string, opts InstallFoundryOptions) ([]InstallFoundryReport, []index.ResolutionWarning, error) {
	cacheDir, err := index.IndexCacheDir()
	if err != nil {
		return nil, nil, err
	}

	var match *index.FoundryEntry
	for _, e := range cfg.EffectiveFoundries() {
		if strings.EqualFold(e.Name, nameOrURL) || strings.EqualFold(e.URL, nameOrURL) {
			matchCopy := e
			match = &matchCopy
			break
		}
	}
	if match == nil {
		return nil, nil, fmt.Errorf("%w: %q", ErrFoundryNotFound, nameOrURL)
	}

	fetcher, ferr := index.NewFetcher(defaultGitRunner())
	if ferr != nil {
		return nil, nil, ferr
	}
	lookup := index.CacheFirstLookup(cacheDir, fetcher)

	r := index.NewResolver(lookup)
	root, allMolds, err := r.Resolve(match.URL)
	if err != nil {
		return nil, nil, fmt.Errorf("resolving foundry %s: %w", match.Name, err)
	}
	warnings := r.Warnings()

	// Build the working set of molds to install.
	var molds []index.ResolvedMold
	if opts.Shallow {
		for _, m := range allMolds {
			if m.Foundry == root {
				molds = append(molds, m)
			}
		}
	} else {
		molds = allMolds
	}

	// Read target lockfile to spot already-installed sources.
	lockPath := foundry.LockFileName
	if opts.Global {
		if home, herr := os.UserHomeDir(); herr == nil {
			lockPath = filepath.Join(home, foundry.LockFileName)
		}
	}
	installed := map[string]string{}
	if lock, _ := foundry.ReadLockFile(lockPath); lock != nil {
		for _, e := range lock.Molds {
			installed[strings.ToLower(e.Source)] = e.Version
		}
	}

	out := make([]InstallFoundryReport, 0, len(molds))
	for _, m := range molds {
		chain := append([]string(nil), m.Foundry.Parents...)
		report := InstallFoundryReport{
			Name:    m.Entry.Name,
			Source:  m.Entry.Source,
			Foundry: m.Foundry.Index.Name,
			Chain:   chain,
		}
		report.FluxApplied, report.FluxSkipped = splitFluxKeysForMold(ctx, m.Entry.Source, opts.SetOverrides)

		if v, already := installed[strings.ToLower(m.Entry.Source)]; already && !opts.Force {
			report.Skipped = true
			report.Version = v
			out = append(out, report)
			continue
		}
		if opts.DryRun {
			out = append(out, report)
			continue
		}

		castRes, cerr := CastMold(ctx, m.Entry.Source, CastOptions{
			Global:        opts.Global,
			WithWorkflows: opts.WithWorkflows,
			ClaudePlugin:  opts.ClaudePlugin,
			ValueFiles:    opts.ValueFiles,
			SetOverrides:  opts.SetOverrides,
		})
		if cerr != nil {
			report.Err = cerr
		} else {
			report.Version = castRes.MoldName
		}
		out = append(out, report)
	}

	return out, warnings, nil
}
