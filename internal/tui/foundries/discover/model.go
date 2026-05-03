package discover

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nimble-giant/ailloy/internal/tui/foundries/data"
	"github.com/nimble-giant/ailloy/pkg/foundry"
	"github.com/nimble-giant/ailloy/pkg/foundry/index"
	"github.com/nimble-giant/ailloy/pkg/styles"
)

var (
	headingStyle    = lipgloss.NewStyle().Foreground(styles.Primary1).Bold(true)
	cursorStyle     = lipgloss.NewStyle().Foreground(styles.Accent1).Bold(true)
	moldNameStyle   = lipgloss.NewStyle().Foreground(styles.Accent1).Bold(true)
	verifiedStyle   = lipgloss.NewStyle().Foreground(styles.Success).Bold(true)
	installedStyle  = lipgloss.NewStyle().Foreground(styles.Info).Bold(true)
	upToDateStyle   = lipgloss.NewStyle().Foreground(styles.Success).Bold(true)
	updateStyle     = lipgloss.NewStyle().Foreground(styles.Warning).Bold(true)
	descStyle       = lipgloss.NewStyle().Foreground(styles.LightGray)
	metaStyle       = lipgloss.NewStyle().Foreground(styles.Gray)
	statusOKStyle   = lipgloss.NewStyle().Foreground(styles.Success)
	statusErrStyle  = lipgloss.NewStyle().Foreground(styles.Error).Bold(true)
	statusWaitStyle = lipgloss.NewStyle().Foreground(styles.Warning)
)

// CastOptions decouples this package from internal/commands. The parent
// translates between this and commands.CastOptions.
type CastOptions struct {
	Global        bool
	WithWorkflows bool
	ValueFiles    []string
	SetOverrides  []string
}

// CastFunc is the install operation injected by the parent.
type CastFunc func(ctx context.Context, source string, opts CastOptions) (foundry.InstalledFile, error)

// CastResultFn signature matches what the parent provides; we discard the
// detail and just look at err for status display.
type CastFn func(ctx context.Context, source string, opts CastOptions) error

// UpdateFn fetches every effective foundry's index into the local cache.
// Used to bootstrap an empty Discover view (e.g. on a fresh install where
// the verified default has never been fetched).
type UpdateFn func(cfg *index.Config) error

// installedInfo is what we know about a mold present in the installed manifest.
// Source/Subpath are kept so we can build a foundry.Reference for resolving the
// latest upstream version.
type installedInfo struct {
	Version string
	Commit  string
	Source  string
	Subpath string
}

// Model is the Discover tab.
type Model struct {
	cfg            *index.Config
	cast           CastFn
	update         UpdateFn
	git            foundry.GitRunner // injected for tests; nil falls back to DefaultGitRunner
	catalog        []data.CatalogEntry
	filtered       []data.CatalogEntry
	installed      map[string]installedInfo           // canonical join key → installed info
	latest         map[string]foundry.ResolvedVersion // canonical join key → latest upstream
	resolving      map[string]bool                    // canonical join key → "checking…" in flight
	selected       map[string]bool
	cursor         int
	filter         textinput.Model
	loading        bool
	loadErr        error
	castStat       map[string]string
	autoFetched    bool                // guard so we only auto-fetch once per session
	pending        map[string][]string // moldRef → encoded "--set" overrides
	wantLatestScan bool                // user pressed `r`; resolve latest after inventory reloads
}

type catalogLoadedMsg struct {
	catalog []data.CatalogEntry
	err     error
}

type inventoryLoadedMsg struct {
	installed map[string]installedInfo // canonical join key → info
}

type catalogFetchedMsg struct{ err error }

type castDoneMsg struct {
	source string
	err    error
}

// latestResolvedMsg reports the result of one upstream version resolve. One
// message fires per installed mold so the UI can update incrementally rather
// than blocking on the slowest network call.
type latestResolvedMsg struct {
	key      string
	resolved *foundry.ResolvedVersion
	err      error
}

// New initializes the Discover model and kicks off catalog loading. cast is
// the install operation, update is the foundry-index fetch operation; pass
// nil for either during tests.
func New(cfg *index.Config, cast CastFn, update UpdateFn) Model {
	ti := textinput.New()
	ti.Placeholder = "filter (name / desc / tag / source)"
	ti.Prompt = "/ "
	return Model{
		cfg:       cfg,
		cast:      cast,
		update:    update,
		installed: map[string]installedInfo{},
		latest:    map[string]foundry.ResolvedVersion{},
		resolving: map[string]bool{},
		selected:  map[string]bool{},
		filter:    ti,
		loading:   true,
		castStat:  map[string]string{},
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(loadCatalogCmd(m.cfg), loadInventoryCmd(m.cfg))
}

func loadCatalogCmd(cfg *index.Config) tea.Cmd {
	return func() tea.Msg {
		c, err := data.LoadCatalog(cfg)
		return catalogLoadedMsg{catalog: c, err: err}
	}
}

func loadInventoryCmd(cfg *index.Config) tea.Cmd {
	return func() tea.Msg {
		items, _ := data.LoadInventory(cfg)
		out := make(map[string]installedInfo, len(items))
		for _, it := range items {
			// Key by the canonical "source[//subpath]" form so subpath-bearing
			// catalog entries (e.g. "github.com/x/y//molds/foo") match installed
			// entries (which store subpath separately). First scope wins —
			// project entries take precedence over global ones.
			key := data.MoldIdentity(it.Entry.Source, it.Entry.Subpath)
			if _, exists := out[key]; !exists {
				out[key] = installedInfo{
					Version: it.Entry.Version,
					Commit:  it.Entry.Commit,
					Source:  it.Entry.Source,
					Subpath: it.Entry.Subpath,
				}
			}
		}
		return inventoryLoadedMsg{installed: out}
	}
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case catalogLoadedMsg:
		m.loading = false
		m.catalog = msg.catalog
		m.loadErr = msg.err
		m.applyFilter()
		// Auto-bootstrap: if the catalog is empty and we have an update fn,
		// fetch all foundry indexes once and reload. Common on first run
		// where the verified default has never been cached.
		if len(m.catalog) == 0 && m.loadErr == nil && m.update != nil && !m.autoFetched {
			m.autoFetched = true
			m.loading = true
			return m, m.fetchCmd()
		}
		return m, nil
	case catalogFetchedMsg:
		// After fetch (success or error), reload from cache.
		return m, loadCatalogCmd(m.cfg)
	case inventoryLoadedMsg:
		m.installed = msg.installed
		// Drop any stale "latest" entries for molds that are no longer installed
		// so the next render doesn't claim "up to date" for things we don't have.
		for k := range m.latest {
			if _, ok := m.installed[k]; !ok {
				delete(m.latest, k)
			}
		}
		if m.wantLatestScan {
			m.wantLatestScan = false
			return m, m.scanLatestCmd()
		}
		return m, nil
	case latestResolvedMsg:
		delete(m.resolving, msg.key)
		if msg.err == nil && msg.resolved != nil {
			m.latest[msg.key] = *msg.resolved
		}
		return m, nil
	case castDoneMsg:
		if msg.err != nil {
			m.castStat[msg.source] = "err: " + msg.err.Error()
		} else {
			m.castStat[msg.source] = "ok"
			delete(m.pending, msg.source)
		}
		// Refresh inventory so the "installed" badge updates immediately, and
		// re-scan latest so an updated mold's "update available" tag clears.
		m.wantLatestScan = true
		return m, loadInventoryCmd(m.cfg)
	case tea.KeyMsg:
		if m.filter.Focused() {
			switch msg.String() {
			case "esc", "enter":
				m.filter.Blur()
				return m, nil
			}
			var cmd tea.Cmd
			m.filter, cmd = m.filter.Update(msg)
			m.applyFilter()
			return m, cmd
		}
		switch msg.String() {
		case "/":
			m.filter.Focus()
			return m, textinput.Blink
		case "j", "down":
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
			}
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case " ":
			if m.cursor < len(m.filtered) {
				src := m.filtered[m.cursor].Source
				if m.selected[src] {
					delete(m.selected, src)
				} else {
					m.selected[src] = true
				}
			}
		case "c":
			m.selected = map[string]bool{}
		case "enter":
			cmds := []tea.Cmd{}
			for src := range m.selected {
				m.castStat[src] = "casting"
				cmds = append(cmds, m.castCmd(src))
			}
			return m, tea.Batch(cmds...)
		case "r":
			m.loading = true
			m.wantLatestScan = true
			return m, tea.Batch(loadCatalogCmd(m.cfg), loadInventoryCmd(m.cfg))
		}
	}
	return m, nil
}

func (m Model) fetchCmd() tea.Cmd {
	update := m.update
	cfg := m.cfg
	return func() tea.Msg {
		if update == nil {
			return catalogFetchedMsg{}
		}
		return catalogFetchedMsg{err: update(cfg)}
	}
}

// scanLatestCmd kicks off one async resolve per installed mold, in parallel.
// Each resolve emits its own latestResolvedMsg so the UI fills in incrementally
// rather than blocking on the slowest network call.
func (m *Model) scanLatestCmd() tea.Cmd {
	if len(m.installed) == 0 {
		return nil
	}
	git := m.git
	if git == nil {
		git = foundry.DefaultGitRunner()
	}
	cmds := make([]tea.Cmd, 0, len(m.installed))
	for key, info := range m.installed {
		if m.resolving[key] {
			continue
		}
		m.resolving[key] = true
		key, info := key, info
		cmds = append(cmds, func() tea.Msg {
			ref, err := referenceFromInstalled(info)
			if err != nil {
				return latestResolvedMsg{key: key, err: err}
			}
			rv, err := foundry.ResolveVersion(ref, git)
			return latestResolvedMsg{key: key, resolved: rv, err: err}
		})
	}
	if len(cmds) == 0 {
		return nil
	}
	return tea.Batch(cmds...)
}

// referenceFromInstalled builds the foundry.Reference needed for a "latest"
// resolve from the source/subpath stored in the installed manifest. Mirrors
// internal/commands.referenceFromInstalledEntry without taking a dependency on
// the commands package.
func referenceFromInstalled(info installedInfo) (*foundry.Reference, error) {
	parts := strings.SplitN(info.Source, "/", 3)
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid source %q: expected host/owner/repo", info.Source)
	}
	return &foundry.Reference{
		Host:    parts[0],
		Owner:   parts[1],
		Repo:    parts[2],
		Subpath: info.Subpath,
		Type:    foundry.Latest,
	}, nil
}

func (m Model) castCmd(source string) tea.Cmd {
	cast := m.cast
	// Capture pending overrides at Cmd build time so concurrent picker edits
	// don't change what's already in flight.
	extra := append([]string(nil), m.pending[source]...)
	return func() tea.Msg {
		if cast == nil {
			return castDoneMsg{source: source, err: fmt.Errorf("no cast function configured")}
		}
		opts := CastOptions{}
		if len(extra) > 0 {
			opts.SetOverrides = append(opts.SetOverrides, extra...)
		}
		return castDoneMsg{source: source, err: cast(context.Background(), source, opts)}
	}
}

func (m *Model) applyFilter() {
	q := strings.ToLower(strings.TrimSpace(m.filter.Value()))
	if q == "" {
		m.filtered = append([]data.CatalogEntry(nil), m.catalog...)
		return
	}
	var out []data.CatalogEntry
	for _, e := range m.catalog {
		if strings.Contains(strings.ToLower(e.Name), q) ||
			strings.Contains(strings.ToLower(e.Description), q) ||
			strings.Contains(strings.ToLower(e.Source), q) {
			out = append(out, e)
			continue
		}
		for _, t := range e.Tags {
			if strings.Contains(strings.ToLower(t), q) {
				out = append(out, e)
				break
			}
		}
	}
	m.filtered = out
	if m.cursor >= len(out) {
		m.cursor = 0
	}
}

// CurrentMold returns the highlighted catalog entry's mold reference and the
// scope to use when casting. Returns ok=false when no entry is highlighted.
func (m Model) CurrentMold() (ref string, scope data.Scope, ok bool) {
	if m.cursor < 0 || m.cursor >= len(m.filtered) {
		return "", "", false
	}
	return m.filtered[m.cursor].Source, data.ScopeProject, true
}

// NewWithFiltered constructs a Model directly with a known filtered slice
// and cursor — exported for cross-package tests in the foundries app.
func NewWithFiltered(entries []data.CatalogEntry, cursor int) Model {
	return Model{filtered: entries, cursor: cursor}
}

// ApplySessionOverrides records session overrides for the given mold ref.
// They will be applied as --set overrides on the next cast of that mold.
// Last-write-wins: re-applying replaces any prior pending overrides for the
// same ref rather than merging.
func (m Model) ApplySessionOverrides(moldRef string, overrides map[string]any) Model {
	if m.pending == nil {
		m.pending = map[string][]string{}
	}
	m.pending[moldRef] = encodeSetOverrides(overrides)
	return m
}

// encodeSetOverrides converts a typed override map into the --set k=v strings
// that CastOptions.SetOverrides expects. Slices are emitted as YAML flow
// sequences ([a,b,c]) so the cast core's YAML-aware --set parser produces a
// real list rather than parsing the Go-default "[a b c]" form as a single
// string. Result is sorted for deterministic ordering.
func encodeSetOverrides(overrides map[string]any) []string {
	out := make([]string, 0, len(overrides))
	for k, v := range overrides {
		out = append(out, fmt.Sprintf("%s=%s", k, formatSetValue(v)))
	}
	sort.Strings(out)
	return out
}

func formatSetValue(v any) string {
	switch x := v.(type) {
	case []string:
		return "[" + strings.Join(x, ",") + "]"
	case []any:
		parts := make([]string, len(x))
		for i, item := range x {
			parts[i] = fmt.Sprintf("%v", item)
		}
		return "[" + strings.Join(parts, ",") + "]"
	default:
		return fmt.Sprintf("%v", v)
	}
}

// renderCastBadge produces the status pill rendered next to a catalog row's
// name. Returns "" when the mold has never been cast (no badge at all);
// otherwise one of:
//
//   - "● up to date {version}"        — installed and matches latest upstream
//   - "● update available {old} → {new}" — installed but upstream has moved on
//   - "● installed {version} (checking…)" — installed, latest still resolving
//   - "● installed {version}"          — installed, latest unknown (resolve failed)
func renderCastBadge(m Model, e data.CatalogEntry) string {
	key := data.MoldIdentity(e.Source, "")
	info, ok := m.installed[key]
	if !ok {
		return ""
	}
	version := info.Version
	if version == "" {
		version = "?"
	}
	if latest, have := m.latest[key]; have {
		if latest.Tag == info.Version && latest.Commit == info.Commit {
			return upToDateStyle.Render("● up to date " + version)
		}
		return updateStyle.Render("● update available " + version + " → " + latest.Tag)
	}
	label := "● installed " + version
	if m.resolving[key] {
		label += " (checking…)"
	}
	return installedStyle.Render(label)
}

func (m Model) View() string {
	if m.loading {
		return metaStyle.Render("Loading catalog…")
	}
	if m.loadErr != nil {
		return statusErrStyle.Render("Error: " + m.loadErr.Error())
	}

	var b strings.Builder
	b.WriteString(m.filter.View() + "\n\n")

	if m.filter.Value() == "" {
		recent := data.Recent(m.catalog, 7*24*time.Hour, 10)
		if len(recent) > 0 {
			b.WriteString(headingStyle.Render("Recent (last 7 days)") + "\n")
			for _, e := range recent {
				fmt.Fprintf(&b, "  %s %s %s\n",
					metaStyle.Render("·"),
					moldNameStyle.Render(e.Name),
					descStyle.Render("— "+e.Description))
			}
			b.WriteString("\n")
		}
	}

	visible := append([]data.CatalogEntry(nil), m.filtered...)
	sort.Slice(visible, func(i, j int) bool { return visible[i].Name < visible[j].Name })
	for i, e := range visible {
		mark := "[ ]"
		if m.selected[e.Source] {
			mark = cursorStyle.Render("[x]")
		}
		caret := "  "
		if i == m.cursor {
			caret = cursorStyle.Render("▶ ")
		}
		verified := ""
		if e.Verified {
			verified = " " + verifiedStyle.Render("✓")
		}
		installedTag := ""
		if badge := renderCastBadge(m, e); badge != "" {
			installedTag = " " + badge
		}
		status := ""
		if s, ok := m.castStat[e.Source]; ok {
			switch s {
			case "ok":
				status = "  " + statusOKStyle.Render("("+s+")")
			case "casting":
				status = "  " + statusWaitStyle.Render("("+s+")")
			default:
				status = "  " + statusErrStyle.Render("("+s+")")
			}
		}
		nested := ""
		if e.IsNested() {
			nested = " " + descStyle.Render("via "+strings.Join(e.OwnerChain, " → "))
		}
		fmt.Fprintf(&b, "%s%s %s%s%s%s %s%s\n",
			caret, mark,
			moldNameStyle.Render(e.Name),
			verified,
			installedTag,
			nested,
			descStyle.Render("— "+e.Description),
			status)
	}

	fmt.Fprintf(&b, "\n%s\n", metaStyle.Render(fmt.Sprintf("%d selected · %d shown · %d total",
		len(m.selected), len(visible), len(m.catalog))))
	b.WriteString(metaStyle.Render("space toggle · enter cast all · / search · c clear · r refresh + check updates · f flux · j/k move") + "\n")
	return b.String()
}
