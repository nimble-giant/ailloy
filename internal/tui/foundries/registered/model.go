package registered

import (
	"errors"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nimble-giant/ailloy/internal/tui/foundries/data"
	"github.com/nimble-giant/ailloy/pkg/foundry/index"
	"github.com/nimble-giant/ailloy/pkg/styles"
)

var (
	headingStyle   = lipgloss.NewStyle().Foreground(styles.Primary1).Bold(true)
	cursorStyle    = lipgloss.NewStyle().Foreground(styles.Accent1).Bold(true)
	foundryNameSty = lipgloss.NewStyle().Foreground(styles.Primary1).Bold(true)
	verifiedStyle  = lipgloss.NewStyle().Foreground(styles.Success).Bold(true)
	typeStyle      = lipgloss.NewStyle().Foreground(styles.Info)
	urlStyle       = lipgloss.NewStyle().Foreground(styles.Gray)
	builtInStyle   = lipgloss.NewStyle().Foreground(styles.LightGray).Italic(true)
	flashOK        = lipgloss.NewStyle().Foreground(styles.Success)
	flashErr       = lipgloss.NewStyle().Foreground(styles.Error).Bold(true)
	flashInfo      = lipgloss.NewStyle().Foreground(styles.Info)
	metaStyle      = lipgloss.NewStyle().Foreground(styles.Gray)
)

// AddFoundryResult mirrors the parent's type so this package doesn't import commands.
type AddFoundryResult struct {
	Entry         index.FoundryEntry
	AlreadyExists bool
	MoldCount     int
}

// UpdateFoundryReport mirrors the parent's type.
type UpdateFoundryReport struct {
	Name      string
	URL       string
	MoldCount int
	Persisted bool
	Err       error
}

// InstallReport mirrors the parent's per-mold install result.
type InstallReport struct {
	Name    string
	Source  string
	Skipped bool
	Err     error
	Version string
}

// AddFn / RemoveFn / UpdateFn / InstallFn are injected operations.
type AddFn func(cfg *index.Config, url string) (AddFoundryResult, error)
type RemoveFn func(cfg *index.Config, nameOrURL string) (index.FoundryEntry, error)
type UpdateFn func(cfg *index.Config) ([]UpdateFoundryReport, error)
type InstallFn func(cfg *index.Config, nameOrURL string) ([]InstallReport, error)

// ErrCannotRemoveDefault must be returned by RemoveFn when the user tries
// to remove the virtual official foundry.
var ErrCannotRemoveDefault = errors.New("cannot remove the default verified foundry; it is built in")

// Model is the Foundries tab.
type Model struct {
	cfg     *index.Config
	add     AddFn
	remove  RemoveFn
	update  UpdateFn
	install InstallFn
	cursor  int
	addMode bool
	addInp  textinput.Model
	flash   string
}

type updateDoneMsg struct {
	reports []UpdateFoundryReport
	err     error
}

type addDoneMsg struct {
	res AddFoundryResult
	err error
}

type removeDoneMsg struct {
	name string
	err  error
}

type installDoneMsg struct {
	name    string
	reports []InstallReport
	err     error
}

func New(cfg *index.Config, add AddFn, remove RemoveFn, update UpdateFn, install InstallFn) Model {
	ti := textinput.New()
	ti.Placeholder = "https://github.com/owner/foundry"
	ti.Prompt = "URL: "
	return Model{cfg: cfg, add: add, remove: remove, update: update, install: install, addInp: ti}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case updateDoneMsg:
		if msg.err != nil {
			m.flash = "update error: " + msg.err.Error()
		} else {
			ok := 0
			for _, r := range msg.reports {
				if r.Err == nil {
					ok++
				}
			}
			m.flash = fmt.Sprintf("updated %d/%d foundries", ok, len(msg.reports))
			_ = index.SaveConfig(m.cfg)
		}
		return m, nil
	case addDoneMsg:
		switch {
		case msg.err != nil:
			m.flash = "add error: " + msg.err.Error()
		case msg.res.AlreadyExists:
			m.flash = "already registered"
		default:
			_ = index.SaveConfig(m.cfg)
			m.flash = fmt.Sprintf("added %s (%d molds)", msg.res.Entry.Name, msg.res.MoldCount)
		}
		m.addMode = false
		m.addInp.SetValue("")
		return m, nil
	case removeDoneMsg:
		switch {
		case msg.err != nil && errors.Is(msg.err, ErrCannotRemoveDefault):
			m.flash = "cannot remove the default verified foundry"
		case msg.err != nil:
			m.flash = "remove error: " + msg.err.Error()
		default:
			_ = index.SaveConfig(m.cfg)
			m.flash = "removed " + msg.name
		}
		return m, nil
	case tea.KeyMsg:
		if m.addMode {
			switch msg.String() {
			case "esc":
				m.addMode = false
				return m, nil
			case "enter":
				url := strings.TrimSpace(m.addInp.Value())
				if url == "" {
					m.addMode = false
					return m, nil
				}
				return m, m.addCmd(url)
			}
			var cmd tea.Cmd
			m.addInp, cmd = m.addInp.Update(msg)
			return m, cmd
		}
		switch msg.String() {
		case "j", "down":
			if m.cursor < len(m.cfg.EffectiveFoundries())-1 {
				m.cursor++
			}
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case "a":
			m.addMode = true
			m.addInp.Focus()
			return m, textinput.Blink
		case "d":
			eff := m.cfg.EffectiveFoundries()
			if m.cursor < len(eff) {
				return m, m.removeCmd(eff[m.cursor].URL)
			}
		case "r":
			return m, m.updateCmd()
		case "i":
			eff := m.cfg.EffectiveFoundries()
			if m.cursor < len(eff) {
				m.flash = "installing every mold from " + eff[m.cursor].Name + "..."
				return m, m.installCmd(eff[m.cursor].Name)
			}
		}
	case installDoneMsg:
		switch {
		case msg.err != nil:
			m.flash = "install error: " + msg.err.Error()
		default:
			ok, skipped, failed := 0, 0, 0
			for _, r := range msg.reports {
				switch {
				case r.Err != nil:
					failed++
				case r.Skipped:
					skipped++
				default:
					ok++
				}
			}
			m.flash = fmt.Sprintf("installed %d · skipped %d · failed %d (from %s)", ok, skipped, failed, msg.name)
		}
		return m, nil
	}
	return m, nil
}

func (m Model) installCmd(name string) tea.Cmd {
	install := m.install
	cfg := m.cfg
	return func() tea.Msg {
		if install == nil {
			return installDoneMsg{name: name, err: fmt.Errorf("no install function configured")}
		}
		reports, err := install(cfg, name)
		return installDoneMsg{name: name, reports: reports, err: err}
	}
}

func (m Model) updateCmd() tea.Cmd {
	update := m.update
	cfg := m.cfg
	return func() tea.Msg {
		if update == nil {
			return updateDoneMsg{err: fmt.Errorf("no update function configured")}
		}
		reports, err := update(cfg)
		return updateDoneMsg{reports: reports, err: err}
	}
}

func (m Model) addCmd(url string) tea.Cmd {
	add := m.add
	cfg := m.cfg
	return func() tea.Msg {
		if add == nil {
			return addDoneMsg{err: fmt.Errorf("no add function configured")}
		}
		res, err := add(cfg, url)
		return addDoneMsg{res: res, err: err}
	}
}

func (m Model) removeCmd(urlOrName string) tea.Cmd {
	remove := m.remove
	cfg := m.cfg
	return func() tea.Msg {
		if remove == nil {
			return removeDoneMsg{name: urlOrName, err: fmt.Errorf("no remove function configured")}
		}
		_, err := remove(cfg, urlOrName)
		return removeDoneMsg{name: urlOrName, err: err}
	}
}

// CurrentMold returns ok=false; the foundries tab has no per-mold context.
func (m Model) CurrentMold() (ref string, scope data.Scope, ok bool) {
	return "", "", false
}

func (m Model) View() string {
	var b strings.Builder
	if m.flash != "" {
		style := flashInfo
		switch {
		case strings.Contains(m.flash, "error") || strings.Contains(m.flash, "cannot"):
			style = flashErr
		case strings.HasPrefix(m.flash, "added") || strings.HasPrefix(m.flash, "removed") || strings.HasPrefix(m.flash, "updated"):
			style = flashOK
		}
		b.WriteString(style.Render(m.flash) + "\n\n")
	}
	if m.addMode {
		b.WriteString(m.addInp.View() + "\n\n" + metaStyle.Render("(enter to add, esc to cancel)") + "\n")
		return b.String()
	}
	b.WriteString(headingStyle.Render("Registered foundries:") + "\n\n")
	hasOfficial := m.cfg.HasOfficialFoundry()
	for i, e := range m.cfg.EffectiveFoundries() {
		caret := "  "
		if i == m.cursor {
			caret = cursorStyle.Render("▶ ")
		}
		verified := ""
		if index.IsOfficialFoundry(e.URL) {
			verified = " " + verifiedStyle.Render("✓ verified")
		}
		builtIn := ""
		if !hasOfficial && index.IsOfficialFoundry(e.URL) {
			builtIn = "  " + builtInStyle.Render("(built-in default)")
		}
		fmt.Fprintf(&b, "%s%s%s  %s  %s%s\n",
			caret,
			foundryNameSty.Render(e.Name),
			verified,
			typeStyle.Render("["+e.Type+"]"),
			urlStyle.Render(e.URL),
			builtIn)
	}
	b.WriteString("\n" + metaStyle.Render("a add · d remove · r refresh · i install all · j/k move") + "\n")
	return b.String()
}
