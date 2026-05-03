// Package evolution plays the retro RPG-style cinematic for `ailloy evolve`.
// The animation runs in the alternate screen buffer after the binary swap
// has already succeeded — the cinematic itself never persists anything; the
// caller is responsible for printing a receipt line to scrollback.
package evolution

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nimble-giant/ailloy/internal/tui/cinematic"
)

const frameInterval = 16 * time.Millisecond

// Beat boundaries (cumulative milliseconds since program start).
const (
	beatHeadlineEnd = 800
	beatHoldEnd     = 1300
	beatChargeEnd   = 2000
	beatFlashEnd    = 3200
	beatCrackEnd    = 3450
	beatRevealEnd   = 4400
	beatFanfareEnd  = 5300
	beatTotal       = 5700
)

type frameMsg time.Time
type startMsg struct{}

// Model is the evolve cinematic. Lifecycle:
//
//	prog := tea.NewProgram(evolution.New(w, h, "v0.6.21", "v0.7.0"), tea.WithAltScreen())
//	prog.Run()
//
// The model self-quits at beatTotal or on any keypress.
type Model struct {
	width    int
	height   int
	compact  bool // true when the terminal can't fit the full fox-sized layout
	start    time.Time
	frame    int
	foxLines []string
	foxRows  int
	foxCols  int
	sparkles []cinematic.Sparkle
	from     string // current version, e.g. "v0.6.21"
	to       string // target version,  e.g. "v0.7.0"
}

// New constructs an evolution Model. The version strings should be in the
// form `vMAJOR.MINOR.PATCH` (or "dev"); the digit-wheel beat will fall back
// to a static rendering of `to` if `from` and `to` have different shapes.
//
// When the supplied terminal height is below `compactHeight`, the model
// flips to a compact layout: no giant fox, just headline / version chip /
// charge / flash / digit wheel / sparkle fanfare centered tightly. The
// beat structure stays identical so timing and skip behavior are unchanged.
func New(width, height int, currentVersion, targetVersion string) Model {
	lines, cols := cinematic.PadFoxLines()
	// Sparkles seeded by the target version so a given upgrade replays
	// identically. 14 sparkles for the full fox layout; 6 for the
	// compact band so it doesn't look chunky in 256-color terminals.
	seed := stringSeed(targetVersion)
	compact := height < compactHeight
	sparkleCount := 14
	if compact {
		sparkleCount = 6
	}
	return Model{
		width:    width,
		height:   height,
		compact:  compact,
		foxLines: lines,
		foxRows:  len(lines),
		foxCols:  cols,
		sparkles: cinematic.NewSparkleField(sparkleCount, len(lines), cols, seed),
		from:     currentVersion,
		to:       targetVersion,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		func() tea.Msg { return startMsg{} },
		tick(),
	)
}

func tick() tea.Cmd {
	return tea.Tick(frameInterval, func(t time.Time) tea.Msg {
		return frameMsg(t)
	})
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case startMsg:
		m.start = time.Now()
		return m, tick()
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		// Any key skips. Receipt is printed by the caller post-Quit.
		return m, tea.Quit
	case frameMsg:
		if m.start.IsZero() {
			return m, tick()
		}
		m.frame++
		if time.Since(m.start) >= time.Duration(beatTotal)*time.Millisecond {
			return m, tea.Quit
		}
		return m, tick()
	}
	return m, nil
}

func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}
	if m.start.IsZero() {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, "")
	}

	elapsedMs := float64(time.Since(m.start) / time.Millisecond)
	switch {
	case elapsedMs < beatHeadlineEnd:
		return m.renderHeadline(normalize(elapsedMs, 0, beatHeadlineEnd))
	case elapsedMs < beatHoldEnd:
		return m.renderHold(normalize(elapsedMs, beatHeadlineEnd, beatHoldEnd))
	case elapsedMs < beatChargeEnd:
		return m.renderCharge(normalize(elapsedMs, beatHoldEnd, beatChargeEnd))
	case elapsedMs < beatFlashEnd:
		return m.renderFlash(normalize(elapsedMs, beatChargeEnd, beatFlashEnd))
	case elapsedMs < beatCrackEnd:
		return m.renderCrack(normalize(elapsedMs, beatFlashEnd, beatCrackEnd))
	case elapsedMs < beatRevealEnd:
		return m.renderReveal(normalize(elapsedMs, beatCrackEnd, beatRevealEnd))
	case elapsedMs < beatFanfareEnd:
		return m.renderFanfare(normalize(elapsedMs, beatRevealEnd, beatFanfareEnd))
	default:
		return m.renderFanfare(1)
	}
}

func normalize(t, lo, hi float64) float64 {
	if hi <= lo {
		return 1
	}
	v := (t - lo) / (hi - lo)
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// stringSeed turns a string into a stable int64 so the same target version
// always seeds the same sparkle layout. Cheap djb2-like hash.
func stringSeed(s string) int64 {
	var h int64 = 5381
	for _, r := range s {
		h = h*33 + int64(r)
	}
	if h < 0 {
		h = -h
	}
	return h
}
