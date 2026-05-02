package commands

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/nimble-giant/ailloy/pkg/foundry"
	"github.com/nimble-giant/ailloy/pkg/foundry/index"
)

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

	out := make([]UpdateFoundryReport, 0, len(targets))
	for _, t := range targets {
		report := UpdateFoundryReport{Name: t.entry.Name, URL: t.entry.URL, Persisted: t.persisted}
		idx, err := fetcher.FetchIndex(t.entry)
		if err != nil {
			report.Err = err
		} else {
			report.MoldCount = len(idx.Molds)
		}
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
}

// InstallFoundryReport is the per-mold result of an InstallFoundryCore run.
type InstallFoundryReport struct {
	Name    string
	Source  string
	Skipped bool   // true when already installed and !Force
	Err     error  // non-nil if cast failed for this mold
	Version string // populated on success or skip (from CastResult / lockfile)
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
func InstallFoundryCore(ctx context.Context, cfg *index.Config, nameOrURL string, opts InstallFoundryOptions) ([]InstallFoundryReport, error) {
	cacheDir, err := index.IndexCacheDir()
	if err != nil {
		return nil, err
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
		return nil, fmt.Errorf("%w: %q", ErrFoundryNotFound, nameOrURL)
	}

	idx, err := index.LoadCachedIndex(cacheDir, match)
	if err != nil {
		// Try to populate the cache once before giving up.
		git := defaultGitRunner()
		fetcher, ferr := index.NewFetcher(git)
		if ferr != nil {
			return nil, fmt.Errorf("loading cached index for %s: %w", match.Name, err)
		}
		if _, fetchErr := fetcher.FetchIndex(match); fetchErr != nil {
			return nil, fmt.Errorf("fetching foundry index %s: %w", match.Name, fetchErr)
		}
		idx, err = index.LoadCachedIndex(cacheDir, match)
		if err != nil {
			return nil, fmt.Errorf("loading cached index for %s: %w", match.Name, err)
		}
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

	out := make([]InstallFoundryReport, 0, len(idx.Molds))
	for _, m := range idx.Molds {
		report := InstallFoundryReport{Name: m.Name, Source: m.Source}

		if v, already := installed[strings.ToLower(m.Source)]; already && !opts.Force {
			report.Skipped = true
			report.Version = v
			out = append(out, report)
			continue
		}
		if opts.DryRun {
			out = append(out, report)
			continue
		}

		castRes, cerr := CastMold(ctx, m.Source, CastOptions{
			Global:        opts.Global,
			WithWorkflows: opts.WithWorkflows,
			ClaudePlugin:  opts.ClaudePlugin,
		})
		if cerr != nil {
			report.Err = cerr
		} else {
			report.Version = castRes.MoldName // best-effort; real version sits in lockfile
		}
		out = append(out, report)
	}

	return out, nil
}
