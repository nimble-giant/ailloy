package fluxpicker

import (
	"strings"
	"testing"

	"github.com/nimble-giant/ailloy/internal/tui/foundries/data"
	"github.com/nimble-giant/ailloy/pkg/mold"
)

func TestView_RendersHeaderAndKeys(t *testing.T) {
	schema := []mold.FluxVar{
		{Name: "agents.targets", Type: "list"},
		{Name: "agents.parallel", Type: "bool", Default: "true"},
	}
	defaults := map[string]any{"agents": map[string]any{"parallel": true}}
	m := New().OpenFor("official/agents", data.ScopeProject, schema, defaults)
	m.width, m.height = 80, 24

	out := m.View()
	if !strings.Contains(out, "official/agents") {
		t.Fatalf("expected mold ref in header, got:\n%s", out)
	}
	if !strings.Contains(out, "agents.targets") {
		t.Fatalf("expected agents.targets in list, got:\n%s", out)
	}
	if !strings.Contains(out, "agents.parallel") {
		t.Fatalf("expected agents.parallel in list, got:\n%s", out)
	}
}

func TestView_BadgesReflectState(t *testing.T) {
	schema := []mold.FluxVar{{Name: "k", Type: "string"}}
	m := New().OpenFor("ref", data.ScopeProject, schema, nil).SetOverride("k", "v")
	m.width, m.height = 80, 24
	out := m.View()
	if !strings.Contains(out, "●") {
		t.Fatalf("expected '●' badge for set key, got:\n%s", out)
	}
}

func TestView_HiddenWhenClosed(t *testing.T) {
	m := New()
	if m.View() != "" {
		t.Fatalf("expected empty view when closed, got %q", m.View())
	}
}

func TestView_RendersSavePromptWhenActive(t *testing.T) {
	schema := []mold.FluxVar{{Name: "k"}}
	m := New().OpenFor("ref", data.ScopeProject, schema, nil)
	m.saving = saveState{active: true}
	m.focus = focusSavePrompt
	out := m.View()
	if !strings.Contains(out, "[p]") || !strings.Contains(out, "[g]") || !strings.Contains(out, "[o]") {
		t.Fatalf("expected save prompt options, got:\n%s", out)
	}
}

func TestView_ShowsErrorBanner(t *testing.T) {
	m := New().OpenFor("ref", data.ScopeProject, nil, nil)
	m.err = errSentinel
	out := m.View()
	if !strings.Contains(out, "boom") {
		t.Fatalf("expected error message in view, got:\n%s", out)
	}
}

type _e struct{}

func (_e) Error() string { return "boom" }

var errSentinel = _e{}

func TestViewFoundryModeShowsConflictMarker(t *testing.T) {
	per := map[string][]mold.FluxVar{
		"alpha": {{Name: "theme", Type: "string", Default: "dark"}},
		"beta":  {{Name: "theme", Type: "string", Default: "light"}},
	}
	m := New().OpenForFoundry("brand", data.ScopeProject, per, nil)
	out := m.View()

	if !strings.Contains(out, "Foundry: brand") {
		t.Errorf("expected foundry header, got:\n%s", out)
	}
	if !strings.Contains(out, "theme") {
		t.Errorf("expected theme row, got:\n%s", out)
	}
	if !strings.Contains(out, "conflicts") {
		t.Errorf("expected conflict marker for theme, got:\n%s", out)
	}
}
