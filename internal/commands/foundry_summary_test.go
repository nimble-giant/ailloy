package commands

import (
	"strings"
	"testing"
)

func TestFormatFoundryFluxSummary(t *testing.T) {
	reports := []InstallFoundryReport{
		{Name: "alpha", FluxApplied: []string{"agents.targets"}},
		{Name: "beta", FluxApplied: []string{"agents.targets", "theme"}},
		{Name: "gamma", FluxSkipped: []string{"agents.targets"}, FluxApplied: []string{"theme"}},
	}
	keys := []string{"agents.targets", "theme"}

	out := formatFoundryFluxSummary(reports, keys)
	if !strings.Contains(out, "agents.targets") {
		t.Errorf("missing agents.targets in summary: %q", out)
	}
	if !strings.Contains(out, "2/3") {
		t.Errorf("agents.targets should report 2/3 molds: %q", out)
	}
	if !strings.Contains(out, "theme") || !strings.Contains(out, "3/3") {
		t.Errorf("theme should report 3/3 molds: %q", out)
	}
	if !strings.Contains(out, "gamma") {
		t.Errorf("gamma (skipped agents.targets) should appear in skip list: %q", out)
	}
}

func TestFormatFoundryFluxSummaryEmpty(t *testing.T) {
	out := formatFoundryFluxSummary(nil, nil)
	if out != "" {
		t.Errorf("empty inputs should yield empty summary, got %q", out)
	}
}
