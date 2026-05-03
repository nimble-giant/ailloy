// Package splash plays a one-shot cinematic intro for the `ailloy` CLI in
// the alternate screen buffer. When it finishes (or is skipped), control
// returns to the caller, which is expected to print the static help into
// normal scrollback. The splash itself prints nothing permanent.
package splash

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nimble-giant/ailloy/internal/tui/cinematic"
	"github.com/nimble-giant/ailloy/pkg/styles"
)

const frameInterval = 16 * time.Millisecond

// Beat boundaries (cumulative milliseconds since splash start).
const (
	beatIgniteEnd = 400
	beatBloomEnd  = 1100
	beatCoolEnd   = 1350
	beatTitleEnd  = 1850
	beatSettleEnd = 2250
	beatTotal     = 2350
)

type frameMsg time.Time

// Model is a self-contained Bubble Tea model. The intended lifecycle is:
//
//	prog := tea.NewProgram(splash.New(w, h), tea.WithAltScreen())
//	prog.Run()
//
// The model self-quits when the cinematic completes or any key is pressed.
type Model struct {
	width    int
	height   int
	start    time.Time
	foxLines []string
	foxRows  int
	foxCols  int
}

// New constructs a splash Model. The caller passes the terminal dimensions so
// the model can center its scene; if WindowSizeMsg arrives later it will
// adjust. Pass 0,0 if unknown — Bubble Tea will deliver dimensions on Init.
func New(width, height int) Model {
	lines, cols := cinematic.PadFoxLines()
	return Model{
		width:    width,
		height:   height,
		foxLines: lines,
		foxRows:  len(lines),
		foxCols:  cols,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		func() tea.Msg {
			return startMsg{}
		},
		tick(),
	)
}

type startMsg struct{}

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
		// Any key skips the splash. We quit immediately; the wrapping
		// command is responsible for printing the static help to
		// scrollback. This keeps the splash a "movie" — no menu, no
		// state survives.
		return m, tea.Quit
	case frameMsg:
		if m.start.IsZero() {
			return m, tick()
		}
		elapsed := time.Since(m.start)
		if elapsed >= time.Duration(beatTotal)*time.Millisecond {
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
		return placeBlank(m.width, m.height)
	}

	elapsedMs := float64(time.Since(m.start) / time.Millisecond)

	switch {
	case elapsedMs < beatIgniteEnd:
		t := elapsedMs / beatIgniteEnd
		return m.renderIgnite(styles.EaseOutCubic(t))
	case elapsedMs < beatBloomEnd:
		t := (elapsedMs - beatIgniteEnd) / (beatBloomEnd - beatIgniteEnd)
		return m.renderBloom(styles.EaseOutQuad(t))
	case elapsedMs < beatCoolEnd:
		t := (elapsedMs - beatBloomEnd) / (beatCoolEnd - beatBloomEnd)
		return m.renderCool(t)
	case elapsedMs < beatTitleEnd:
		t := (elapsedMs - beatCoolEnd) / (beatTitleEnd - beatCoolEnd)
		return m.renderTitle(t)
	case elapsedMs < beatSettleEnd:
		t := (elapsedMs - beatTitleEnd) / (beatSettleEnd - beatTitleEnd)
		return m.renderSettle(styles.EaseOutCubic(t))
	default:
		return m.renderSettle(1)
	}
}

func placeBlank(w, h int) string {
	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, "")
}
