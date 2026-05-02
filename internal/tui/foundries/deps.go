package foundries

import (
	"context"

	"github.com/nimble-giant/ailloy/pkg/foundry"
	"github.com/nimble-giant/ailloy/pkg/foundry/index"
)

// CastFunc installs a mold by source. Mirrors commands.CastMold's signature.
type CastFunc func(ctx context.Context, source string, opts CastOptions) (CastResult, error)

// CastOptions mirrors commands.CastOptions.
type CastOptions struct {
	Global        bool
	WithWorkflows bool
	ValueFiles    []string
	SetOverrides  []string
}

// CastResult mirrors commands.CastResult.
type CastResult struct {
	Source     string
	MoldName   string
	FilesCast  []foundry.InstalledFile
	Dirs       []string
	GlobalRoot string
}

// AddFoundryResult mirrors commands.AddFoundryResult.
type AddFoundryResult struct {
	Entry         index.FoundryEntry
	AlreadyExists bool
	MoldCount     int
}

// UpdateFoundryReport mirrors commands.UpdateFoundryReport.
type UpdateFoundryReport struct {
	Name      string
	URL       string
	MoldCount int
	Persisted bool
	Err       error
}

// AddFoundryFunc registers a foundry URL into cfg and fetches its index.
type AddFoundryFunc func(cfg *index.Config, url string) (AddFoundryResult, error)

// RemoveFoundryFunc removes a foundry from cfg by name or URL.
type RemoveFoundryFunc func(cfg *index.Config, nameOrURL string) (index.FoundryEntry, error)

// UpdateFoundriesFunc fetches every effective foundry's index.
type UpdateFoundriesFunc func(cfg *index.Config) ([]UpdateFoundryReport, error)

// Deps wires platform-level operations into the TUI without forcing the
// tui package to import internal/commands (which would create a cycle).
// commands/foundries.go populates this when constructing the App.
type Deps struct {
	Cast            CastFunc
	AddFoundry      AddFoundryFunc
	RemoveFoundry   RemoveFoundryFunc
	UpdateFoundries UpdateFoundriesFunc
}
