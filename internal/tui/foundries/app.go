package foundries

import (
	"context"
	"errors"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nimble-giant/ailloy/internal/tui/foundries/data"
	"github.com/nimble-giant/ailloy/internal/tui/foundries/discover"
	"github.com/nimble-giant/ailloy/internal/tui/foundries/fluxpicker"
	"github.com/nimble-giant/ailloy/internal/tui/foundries/health"
	"github.com/nimble-giant/ailloy/internal/tui/foundries/installed"
	"github.com/nimble-giant/ailloy/internal/tui/foundries/registered"
	"github.com/nimble-giant/ailloy/pkg/foundry/index"
	"github.com/nimble-giant/ailloy/pkg/mold"
)

// MoldContexter is implemented by tabs that can identify a "currently
// highlighted" mold. The flux picker uses this to scope its operations.
type MoldContexter interface {
	CurrentMold() (ref string, scope data.Scope, ok bool)
}

type discoverCtx = context.Context

var errMissingDep = errors.New("operation not configured for this TUI build")

// App is the root tea.Model. It owns the active tab, shared config, and
// one sub-model per tab. Sub-models are addressed by value (Bubble Tea's
// Elm-style update returns a new value), so we re-assign on every Update.
type App struct {
	active     Tab
	width      int
	height     int
	cfg        *index.Config
	discover   discover.Model
	installed  installed.Model
	registered registered.Model
	health     health.Model
	picker     fluxpicker.Model
}

// New constructs an App. deps wires platform operations (cast/add/remove/update)
// — pass nil-bearing Deps for tests; the TUI will still render but action keys
// will report "no X function configured".
func New(deps Deps) App {
	cfg, _ := index.LoadConfig()
	if cfg == nil {
		cfg = &index.Config{}
	}

	discoverCast := func(ctx discoverCtx, source string, opts discover.CastOptions) error {
		if deps.Cast == nil {
			return errMissingDep
		}
		_, err := deps.Cast(ctx, source, CastOptions{
			Global:        opts.Global,
			WithWorkflows: opts.WithWorkflows,
			ValueFiles:    opts.ValueFiles,
			SetOverrides:  opts.SetOverrides,
		})
		return err
	}
	installedCast := func(ctx discoverCtx, source string, opts installed.CastOptions) error {
		if deps.Cast == nil {
			return errMissingDep
		}
		_, err := deps.Cast(ctx, source, CastOptions{
			Global:        opts.Global,
			WithWorkflows: opts.WithWorkflows,
			ValueFiles:    opts.ValueFiles,
			SetOverrides:  opts.SetOverrides,
		})
		return err
	}
	regAdd := func(cfg *index.Config, url string) (registered.AddFoundryResult, error) {
		if deps.AddFoundry == nil {
			return registered.AddFoundryResult{}, errMissingDep
		}
		r, err := deps.AddFoundry(cfg, url)
		return registered.AddFoundryResult{Entry: r.Entry, AlreadyExists: r.AlreadyExists, MoldCount: r.MoldCount}, err
	}
	regRemove := func(cfg *index.Config, nameOrURL string) (index.FoundryEntry, error) {
		if deps.RemoveFoundry == nil {
			return index.FoundryEntry{}, errMissingDep
		}
		return deps.RemoveFoundry(cfg, nameOrURL)
	}
	regUpdate := func(cfg *index.Config) ([]registered.UpdateFoundryReport, error) {
		if deps.UpdateFoundries == nil {
			return nil, errMissingDep
		}
		reports, err := deps.UpdateFoundries(cfg)
		out := make([]registered.UpdateFoundryReport, 0, len(reports))
		for _, r := range reports {
			out = append(out, registered.UpdateFoundryReport{
				Name: r.Name, URL: r.URL, MoldCount: r.MoldCount, Persisted: r.Persisted, Err: r.Err,
			})
		}
		return out, err
	}
	discoverUpdate := func(cfg *index.Config) error {
		if deps.UpdateFoundries == nil {
			return errMissingDep
		}
		_, err := deps.UpdateFoundries(cfg)
		return err
	}
	regInstall := func(cfg *index.Config, nameOrURL string) ([]registered.InstallReport, error) {
		if deps.InstallFoundry == nil {
			return nil, errMissingDep
		}
		reports, err := deps.InstallFoundry(context.Background(), cfg, nameOrURL, InstallFoundryOptions{})
		out := make([]registered.InstallReport, 0, len(reports))
		for _, r := range reports {
			out = append(out, registered.InstallReport{
				Name: r.Name, Source: r.Source, Skipped: r.Skipped, Err: r.Err, Version: r.Version,
			})
		}
		return out, err
	}

	return App{
		cfg:        cfg,
		discover:   discover.New(cfg, discoverCast, discoverUpdate),
		installed:  installed.New(cfg, installedCast),
		registered: registered.New(cfg, regAdd, regRemove, regUpdate, regInstall),
		health:     health.New(cfg),
		picker:     fluxpicker.New(),
	}
}

func (a App) Init() tea.Cmd {
	return tea.Batch(
		a.discover.Init(),
		a.installed.Init(),
		a.registered.Init(),
		a.health.Init(),
	)
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m := msg.(type) {
	case tea.WindowSizeMsg:
		a.width, a.height = m.Width, m.Height
		// Picker is sized here; tabs receive WindowSizeMsg via the broadcast
		// loop below.
		a.picker, _ = a.picker.Update(m)
	case fluxpicker.FluxOverridesMsg:
		if m.Target == fluxpicker.SaveTargetSession {
			switch a.active {
			case TabDiscover:
				a.discover = a.discover.ApplySessionOverrides(m.MoldRef, m.Overrides)
			case TabInstalled:
				a.installed = a.installed.ApplySessionOverrides(m.MoldRef, m.Overrides)
			}
		}
		a.picker = a.picker.Close()
		return a, nil
	case tea.KeyMsg:
		// Picker captures all keys while open — including tab-switch keys —
		// so the user can't navigate tabs behind the overlay.
		if a.picker.IsOpen() {
			var cmd tea.Cmd
			a.picker, cmd = a.picker.Update(m)
			return a, cmd
		}
		switch m.String() {
		case "ctrl+c", "q":
			return a, tea.Quit
		case "tab", "right", "l":
			a.active = (a.active + 1) % tabCount
			return a, nil
		case "shift+tab", "left", "h":
			a.active = (a.active + tabCount - 1) % tabCount
			return a, nil
		case "f":
			ref, scope, ok := a.currentMold()
			if !ok {
				return a, nil
			}
			schema, defaults := a.loadSchemaForMold(ref)
			a.picker = a.picker.OpenFor(ref, scope, schema, defaults)
			return a, nil
		}

		// Key events go only to the active tab.
		var cmd tea.Cmd
		switch a.active {
		case TabDiscover:
			a.discover, cmd = a.discover.Update(msg)
		case TabInstalled:
			a.installed, cmd = a.installed.Update(msg)
		case TabFoundries:
			a.registered, cmd = a.registered.Update(msg)
		case TabHealth:
			a.health, cmd = a.health.Update(msg)
		}
		return a, cmd
	}

	// Non-key messages (window size, async results from each tab's Init)
	// broadcast to every sub-model. Each tab filters on its own message
	// types, so cross-talk is harmless and avoids lost loaded-msgs when
	// the user isn't on the originating tab.
	var dCmd, iCmd, rCmd, hCmd tea.Cmd
	a.discover, dCmd = a.discover.Update(msg)
	a.installed, iCmd = a.installed.Update(msg)
	a.registered, rCmd = a.registered.Update(msg)
	a.health, hCmd = a.health.Update(msg)
	return a, tea.Batch(dCmd, iCmd, rCmd, hCmd)
}

func (a App) View() string {
	var b strings.Builder
	for t := Tab(0); t < tabCount; t++ {
		if t == a.active {
			b.WriteString(tabActive.Render(t.String()))
		} else {
			b.WriteString(tabInactive.Render(t.String()))
		}
	}
	b.WriteString("\n\n")
	switch a.active {
	case TabDiscover:
		b.WriteString(bodyBox.Render(a.discover.View()))
	case TabInstalled:
		b.WriteString(bodyBox.Render(a.installed.View()))
	case TabFoundries:
		b.WriteString(bodyBox.Render(a.registered.View()))
	case TabHealth:
		b.WriteString(bodyBox.Render(a.health.View()))
	}
	b.WriteString("\n")
	b.WriteString(statusBar.Render("tab/shift-tab to switch · q quit"))
	if a.picker.IsOpen() {
		b.WriteString("\n")
		b.WriteString(a.picker.View())
	}
	return b.String()
}

// currentMold delegates to the active tab's MoldContexter to identify the
// currently highlighted mold reference and scope.
func (a App) currentMold() (string, data.Scope, bool) {
	switch a.active {
	case TabDiscover:
		return a.discover.CurrentMold()
	case TabInstalled:
		return a.installed.CurrentMold()
	case TabFoundries:
		return a.registered.CurrentMold()
	case TabHealth:
		return a.health.CurrentMold()
	}
	return "", "", false
}

// loadSchemaForMold attempts a synchronous fetch via mold.FetchSchemaFromSource.
// On error returns empty schema/defaults; the picker shows a "no flux variables"
// state. (Task 13 makes this asynchronous.)
func (a App) loadSchemaForMold(ref string) ([]mold.FluxVar, map[string]any) {
	schema, defaults, err := mold.FetchSchemaFromSource(context.Background(), ref)
	if err != nil {
		return nil, nil
	}
	return schema, defaults
}

// Interface compliance — these will fail to compile if any tab drifts.
var (
	_ MoldContexter = discover.Model{}
	_ MoldContexter = installed.Model{}
	_ MoldContexter = registered.Model{}
	_ MoldContexter = health.Model{}
)
