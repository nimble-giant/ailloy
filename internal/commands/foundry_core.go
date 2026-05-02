package commands

import (
	"errors"
	"fmt"
	"strings"

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
