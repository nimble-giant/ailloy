package registered

import (
	"reflect"
	"testing"

	"github.com/nimble-giant/ailloy/internal/tui/foundries/data"
	"github.com/nimble-giant/ailloy/pkg/foundry/index"
)

func TestCurrentFoundryReturnsHighlight(t *testing.T) {
	cfg := &index.Config{
		Foundries: []index.FoundryEntry{
			{Name: "alpha", URL: "https://github.com/x/alpha"},
			{Name: "beta", URL: "https://github.com/x/beta"},
		},
	}
	m := New(cfg, nil, nil, nil, nil)
	// EffectiveFoundries prepends the official one when not present, so we
	// can't assume cursor 0 is "alpha". Walk the slice to find it.
	for i, e := range cfg.EffectiveFoundries() {
		if e.Name == "alpha" {
			m.cursor = i
			break
		}
	}

	name, scope, ok := m.CurrentFoundry()
	if !ok {
		t.Fatalf("CurrentFoundry ok = false, want true")
	}
	if name != "alpha" {
		t.Errorf("CurrentFoundry name = %q, want alpha", name)
	}
	if scope != data.ScopeProject {
		t.Errorf("CurrentFoundry scope = %q, want project", scope)
	}
}

func TestApplyFoundrySessionOverridesStoresPerMold(t *testing.T) {
	cfg := &index.Config{}
	m := New(cfg, nil, nil, nil, nil)

	overrides := map[string]map[string]any{
		"alpha": {"agents.targets": []any{"claude"}},
		"beta":  {"theme": "midnight"},
	}
	m = m.ApplyFoundrySessionOverrides("nimble-mold", overrides)

	got := m.pendingFoundry["nimble-mold"]
	if !reflect.DeepEqual(got["alpha"], []string{"agents.targets=[claude]"}) {
		t.Errorf("alpha pending = %v, want [agents.targets=[claude]]", got["alpha"])
	}
	if !reflect.DeepEqual(got["beta"], []string{"theme=midnight"}) {
		t.Errorf("beta pending = %v, want [theme=midnight]", got["beta"])
	}
}

func TestInstallCmdForwardsPendingOverrides(t *testing.T) {
	cfg := &index.Config{}
	var captured map[string][]string
	install := func(cfg *index.Config, name string, perMold map[string][]string) ([]InstallReport, error) {
		captured = perMold
		return nil, nil
	}
	m := New(cfg, nil, nil, nil, install).
		ApplyFoundrySessionOverrides("nimble-mold", map[string]map[string]any{
			"alpha": {"agents.targets": []any{"claude"}},
		})

	cmd := m.installCmd("nimble-mold")
	cmd() // run the tea.Cmd inline

	if got := captured["alpha"]; !reflect.DeepEqual(got, []string{"agents.targets=[claude]"}) {
		t.Errorf("captured alpha = %v, want [agents.targets=[claude]]", got)
	}
}
