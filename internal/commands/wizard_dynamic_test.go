package commands

import (
	"fmt"
	"strings"
	"testing"

	"github.com/nimble-giant/ailloy/pkg/mold"
)

func TestGroupPrefix(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"project.organization", "project"},
		{"ore.status.enabled", "ore.status"},
		{"simple", "general"},
		{"a.b.c.d", "a.b.c"},
	}

	for _, tt := range tests {
		got := groupPrefix(tt.input)
		if got != tt.expected {
			t.Errorf("groupPrefix(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestGroupTitle(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"project", "Project"},
		{"ore.status", "Ore > Status"},
		{"general", "General"},
		{"a.b.c", "A > B > C"},
	}

	for _, tt := range tests {
		got := groupTitle(tt.input)
		if got != tt.expected {
			t.Errorf("groupTitle(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestCollectGroups_GroupsByPrefix(t *testing.T) {
	schema := []mold.FluxVar{
		{Name: "project.organization", Type: "string"},
		{Name: "project.board", Type: "string"},
		{Name: "ore.status.enabled", Type: "bool"},
		{Name: "ore.status.field_id", Type: "string"},
		{Name: "ore.priority.enabled", Type: "bool"},
	}

	groups := collectGroups(schema)

	if len(groups) != 3 {
		t.Fatalf("expected 3 groups, got %d", len(groups))
	}

	// Check group names
	if groups[0].name != "project" {
		t.Errorf("group[0].name = %q, want %q", groups[0].name, "project")
	}
	if groups[1].name != "ore.status" {
		t.Errorf("group[1].name = %q, want %q", groups[1].name, "ore.status")
	}
	if groups[2].name != "ore.priority" {
		t.Errorf("group[2].name = %q, want %q", groups[2].name, "ore.priority")
	}

	// Check var counts
	if len(groups[0].vars) != 2 {
		t.Errorf("group[0] should have 2 vars, got %d", len(groups[0].vars))
	}
	if len(groups[1].vars) != 2 {
		t.Errorf("group[1] should have 2 vars, got %d", len(groups[1].vars))
	}
	if len(groups[2].vars) != 1 {
		t.Errorf("group[2] should have 1 var, got %d", len(groups[2].vars))
	}
}

func TestCollectGroups_NoDotsGoToGeneral(t *testing.T) {
	schema := []mold.FluxVar{
		{Name: "simple_var", Type: "string"},
	}

	groups := collectGroups(schema)
	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}
	if groups[0].name != "general" {
		t.Errorf("expected group name 'general', got %q", groups[0].name)
	}
}

func TestCollectGroups_PreservesOrder(t *testing.T) {
	schema := []mold.FluxVar{
		{Name: "b.first", Type: "string"},
		{Name: "a.second", Type: "string"},
		{Name: "b.third", Type: "string"},
	}

	groups := collectGroups(schema)
	if len(groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(groups))
	}
	// b appears first because b.first comes first in schema
	if groups[0].name != "b" {
		t.Errorf("expected first group to be 'b', got %q", groups[0].name)
	}
	if groups[1].name != "a" {
		t.Errorf("expected second group to be 'a', got %q", groups[1].name)
	}
	if len(groups[0].vars) != 2 {
		t.Errorf("group 'b' should have 2 vars, got %d", len(groups[0].vars))
	}
}

func TestFieldTitle(t *testing.T) {
	tests := []struct {
		input    mold.FluxVar
		expected string
	}{
		{mold.FluxVar{Name: "project.organization"}, "Organization"},
		{mold.FluxVar{Name: "ore.status.field_id"}, "Field id"},
		{mold.FluxVar{Name: "simple"}, "Simple"},
		{mold.FluxVar{Name: "ore.status.enabled"}, "Enabled"},
	}

	for _, tt := range tests {
		got := fieldTitle(tt.input)
		if got != tt.expected {
			t.Errorf("fieldTitle(%q) = %q, want %q", tt.input.Name, got, tt.expected)
		}
	}
}

func TestNewDynamicWizard_PrePopulatesValues(t *testing.T) {
	schema := []mold.FluxVar{
		{Name: "project.org", Type: "string"},
		{Name: "enabled", Type: "bool"},
		{Name: "count", Type: "int", Default: "42"},
	}
	flux := map[string]any{
		"project": map[string]any{"org": "acme"},
		"enabled": true,
	}

	w := newDynamicWizard(schema, flux)

	// String value pre-populated from flux
	if *w.values["project.org"] != "acme" {
		t.Errorf("expected project.org = 'acme', got %q", *w.values["project.org"])
	}

	// Bool value pre-populated from flux
	if !*w.boolVals["enabled"] {
		t.Error("expected enabled = true")
	}

	// Int value pre-populated from default
	if *w.values["count"] != "42" {
		t.Errorf("expected count = '42', got %q", *w.values["count"])
	}
}

func TestNewDynamicWizard_BoolDefault(t *testing.T) {
	schema := []mold.FluxVar{
		{Name: "feat.on", Type: "bool", Default: "true"},
		{Name: "feat.off", Type: "bool", Default: "false"},
		{Name: "feat.unset", Type: "bool"},
	}

	w := newDynamicWizard(schema, map[string]any{})

	if !*w.boolVals["feat.on"] {
		t.Error("expected feat.on = true from default")
	}
	if *w.boolVals["feat.off"] {
		t.Error("expected feat.off = false from default")
	}
	if *w.boolVals["feat.unset"] {
		t.Error("expected feat.unset = false (zero value)")
	}
}

func TestDynamicWizard_BuildSummary(t *testing.T) {
	schema := []mold.FluxVar{
		{Name: "project.org", Type: "string"},
		{Name: "enabled", Type: "bool"},
		{Name: "tags", Type: "list"},
	}

	w := newDynamicWizard(schema, map[string]any{})
	*w.values["project.org"] = "acme"
	*w.boolVals["enabled"] = true
	*w.textVals["tags"] = "a,b,c"

	summary := w.buildSummary()

	if !strings.Contains(summary, "project.org: acme") {
		t.Error("expected project.org in summary")
	}
	if !strings.Contains(summary, "enabled: true") {
		t.Error("expected enabled in summary")
	}
	if !strings.Contains(summary, "tags: a,b,c") {
		t.Error("expected tags in summary")
	}
}

func TestDynamicWizard_BuildSummary_SkipsEmpty(t *testing.T) {
	schema := []mold.FluxVar{
		{Name: "project.org", Type: "string"},
		{Name: "project.board", Type: "string"},
	}

	w := newDynamicWizard(schema, map[string]any{})
	*w.values["project.org"] = "acme"
	// project.board left empty

	summary := w.buildSummary()

	if !strings.Contains(summary, "project.org: acme") {
		t.Error("expected project.org in summary")
	}
	if strings.Contains(summary, "project.board") {
		t.Error("expected project.board to be skipped when empty")
	}
}

func TestDynamicWizard_CurrentFlux(t *testing.T) {
	schema := []mold.FluxVar{
		{Name: "project.org", Type: "string"},
		{Name: "enabled", Type: "bool"},
	}

	w := newDynamicWizard(schema, map[string]any{
		"existing": "value",
	})
	*w.values["project.org"] = "acme"
	*w.boolVals["enabled"] = true

	flux := w.currentFlux()

	// Check new values
	orgVal, ok := mold.GetNestedAny(flux, "project.org")
	if !ok || orgVal != "acme" {
		t.Errorf("expected project.org = 'acme', got %v", orgVal)
	}
	enabledVal, ok := mold.GetNestedAny(flux, "enabled")
	if !ok || enabledVal != true {
		t.Errorf("expected enabled = true, got %v", enabledVal)
	}

	// Check existing value preserved
	existingVal, ok := flux["existing"]
	if !ok || existingVal != "value" {
		t.Errorf("expected existing = 'value', got %v", existingVal)
	}
}

func TestDynamicWizard_BuildGroups_GeneratesGroups(t *testing.T) {
	schema := []mold.FluxVar{
		{Name: "project.organization", Type: "string", Required: true, Description: "Org name"},
		{Name: "project.board", Type: "string", Default: "Engineering"},
		{Name: "feature.enabled", Type: "bool", Default: "true"},
	}

	w := newDynamicWizard(schema, map[string]any{})
	groups := w.buildGroups()

	if len(groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(groups))
	}
}

func TestDynamicWizard_BuildGroups_ConditionalFieldsSplit(t *testing.T) {
	schema := []mold.FluxVar{
		{Name: "ore.status.enabled", Type: "bool", Default: "false"},
		{Name: "ore.status.field_id", Type: "string", Discover: &mold.DiscoverSpec{
			Command: "echo test",
			Prompt:  "select",
		}},
	}

	w := newDynamicWizard(schema, map[string]any{})
	groups := w.buildGroups()

	// enabled goes in main group, field_id goes in conditional group = 2 groups
	if len(groups) != 2 {
		t.Fatalf("expected 2 groups (main + conditional), got %d", len(groups))
	}
}

func TestDynamicWizard_SiblingEnabledHideFunc(t *testing.T) {
	schema := []mold.FluxVar{
		{Name: "ore.status.enabled", Type: "bool", Default: "false"},
		{Name: "ore.status.field_id", Type: "string"},
	}

	w := newDynamicWizard(schema, map[string]any{})

	// field_id should have a hide func since ore.status.enabled exists
	hideFunc := w.siblingEnabledHideFunc("ore.status.field_id")
	if hideFunc == nil {
		t.Fatal("expected non-nil hideFunc for ore.status.field_id")
	}

	// enabled=false -> hide=true
	if !hideFunc() {
		t.Error("expected hidden when enabled=false")
	}

	// enabled=true -> hide=false
	*w.boolVals["ore.status.enabled"] = true
	if hideFunc() {
		t.Error("expected visible when enabled=true")
	}

	// No sibling enabled -> nil
	if w.siblingEnabledHideFunc("project.organization") != nil {
		t.Error("expected nil hideFunc for field without sibling enabled")
	}
}

func TestDiscoverCommandRefs(t *testing.T) {
	tests := []struct {
		cmd      string
		expected []string
	}{
		{"echo hello", nil},
		{"gh api -f org='{{.project.organization}}'", []string{"project.organization"}},
		{"gh api -f org='{{.project.organization}}' -F n={{.project.number}}", []string{"project.organization", "project.number"}},
		{"{{.project.org}} and {{.project.org}}", []string{"project.org"}}, // deduped
	}

	for _, tt := range tests {
		got := discoverCommandRefs(tt.cmd)
		if len(got) != len(tt.expected) {
			t.Errorf("discoverCommandRefs(%q) = %v, want %v", tt.cmd, got, tt.expected)
			continue
		}
		for i := range got {
			if got[i] != tt.expected[i] {
				t.Errorf("discoverCommandRefs(%q)[%d] = %q, want %q", tt.cmd, i, got[i], tt.expected[i])
			}
		}
	}
}

func TestDynamicWizard_RunDiscovery_Success(t *testing.T) {
	schema := []mold.FluxVar{
		{Name: "project.id", Type: "string", Discover: &mold.DiscoverSpec{
			Command: "echo test",
			Prompt:  "select",
		}},
	}

	w := newDynamicWizard(schema, map[string]any{})
	w.discovery = &mold.DiscoverExecutor{
		RunCmd: func(cmd string) ([]byte, error) {
			return []byte("Board A|id_a\nBoard B|id_b\n"), nil
		},
	}

	opts := w.runDiscovery(schema[0])

	// Should have skip option + 2 discovered options
	if len(opts) != 3 {
		t.Fatalf("expected 3 options (skip + 2 discovered), got %d", len(opts))
	}
}

func TestDynamicWizard_RunDiscovery_Failure(t *testing.T) {
	schema := []mold.FluxVar{
		{Name: "project.id", Type: "string", Discover: &mold.DiscoverSpec{
			Command: "false",
			Prompt:  "select",
		}},
	}

	w := newDynamicWizard(schema, map[string]any{})
	w.discovery = &mold.DiscoverExecutor{
		RunCmd: func(cmd string) ([]byte, error) {
			return nil, fmt.Errorf("command failed")
		},
	}

	opts := w.runDiscovery(schema[0])

	// Should have a single error option
	if len(opts) != 1 {
		t.Fatalf("expected 1 fallback option, got %d", len(opts))
	}
}

func TestDynamicWizard_RunDiscovery_WaitingForDeps(t *testing.T) {
	schema := []mold.FluxVar{
		{Name: "project.organization", Type: "string"},
		{Name: "project.id", Type: "string", Discover: &mold.DiscoverSpec{
			Command: "gh api graphql -f org='{{.project.organization}}'",
			Prompt:  "select",
		}},
	}

	// Organization is empty — discovery should not run
	w := newDynamicWizard(schema, map[string]any{})
	cmdRan := false
	w.discovery = &mold.DiscoverExecutor{
		RunCmd: func(cmd string) ([]byte, error) {
			cmdRan = true
			return []byte("Board A|id_a\n"), nil
		},
	}

	opts := w.runDiscovery(schema[1])

	if cmdRan {
		t.Error("discovery command should not have run with empty dependencies")
	}
	if len(opts) != 1 {
		t.Fatalf("expected 1 waiting option, got %d", len(opts))
	}
}

func TestDynamicWizard_RunDiscovery_DepsPopulated(t *testing.T) {
	schema := []mold.FluxVar{
		{Name: "project.organization", Type: "string"},
		{Name: "project.id", Type: "string", Discover: &mold.DiscoverSpec{
			Command: "gh api graphql -f org='{{.project.organization}}'",
			Prompt:  "select",
		}},
	}

	// Organization is populated — discovery should run
	w := newDynamicWizard(schema, map[string]any{
		"project": map[string]any{"organization": "acme"},
	})
	w.discovery = &mold.DiscoverExecutor{
		RunCmd: func(cmd string) ([]byte, error) {
			return []byte("Board A|id_a\n"), nil
		},
	}

	opts := w.runDiscovery(schema[1])

	// Should have skip + 1 discovered option
	if len(opts) != 2 {
		t.Fatalf("expected 2 options, got %d", len(opts))
	}
}

func TestDynamicWizard_AlsoSets(t *testing.T) {
	schema := []mold.FluxVar{
		{Name: "project.id", Type: "string", Discover: &mold.DiscoverSpec{
			Command: "echo test",
			Prompt:  "select",
			AlsoSets: map[string]int{
				"project.board":  0,
				"project.number": 1,
			},
		}},
	}

	w := newDynamicWizard(schema, map[string]any{})
	w.discovery = &mold.DiscoverExecutor{
		RunCmd: func(cmd string) ([]byte, error) {
			return []byte("engineering (#6)|PVT_abc|engineering|6\n"), nil
		},
	}

	// Run discovery to populate results
	opts := w.runDiscovery(schema[0])
	if len(opts) != 2 { // skip + 1 result
		t.Fatalf("expected 2 options, got %d", len(opts))
	}

	// Simulate user selecting the project
	*w.values["project.id"] = "PVT_abc"

	// Build flux — also_sets should propagate
	flux := w.currentFlux()

	boardVal, ok := mold.GetNestedAny(flux, "project.board")
	if !ok || boardVal != "engineering" {
		t.Errorf("expected project.board = 'engineering', got %v", boardVal)
	}

	numberVal, ok := mold.GetNestedAny(flux, "project.number")
	if !ok || numberVal != "6" {
		t.Errorf("expected project.number = '6', got %v", numberVal)
	}
}

func TestMissingTemplateDeps_AllMissing(t *testing.T) {
	tmpl := "gh api -f org='{{.project.organization}}' -F number={{.project.id}}"
	flux := map[string]any{}

	missing := missingTemplateDeps(tmpl, flux)
	if len(missing) != 2 {
		t.Fatalf("expected 2 missing deps, got %d: %v", len(missing), missing)
	}
}

func TestMissingTemplateDeps_AllPresent(t *testing.T) {
	tmpl := "gh api -f org='{{.project.organization}}'"
	flux := map[string]any{
		"project": map[string]any{"organization": "acme"},
	}

	missing := missingTemplateDeps(tmpl, flux)
	if len(missing) != 0 {
		t.Errorf("expected no missing deps, got %v", missing)
	}
}

func TestMissingTemplateDeps_NoDottedRefs(t *testing.T) {
	tmpl := "echo hello"
	missing := missingTemplateDeps(tmpl, map[string]any{})
	if len(missing) != 0 {
		t.Errorf("expected no missing deps for simple command, got %v", missing)
	}
}

func TestMissingTemplateDeps_Deduplicates(t *testing.T) {
	tmpl := "{{.project.org}} and again {{.project.org}}"
	missing := missingTemplateDeps(tmpl, map[string]any{})
	if len(missing) != 1 {
		t.Errorf("expected 1 deduped missing dep, got %d: %v", len(missing), missing)
	}
}

func TestExtractDottedRefs(t *testing.T) {
	tests := []struct {
		expr     string
		expected []string
	}{
		{".project.organization", []string{"project.organization"}},
		{" .project.org ", []string{"project.org"}},
		{"range .data.org.nodes", []string{"data.org.nodes"}},
		{".simple", nil}, // no dot in ref, ignored
		{"", nil},
	}

	for _, tt := range tests {
		got := extractDottedRefs(tt.expr)
		if len(got) != len(tt.expected) {
			t.Errorf("extractDottedRefs(%q) = %v, want %v", tt.expr, got, tt.expected)
			continue
		}
		for i := range got {
			if got[i] != tt.expected[i] {
				t.Errorf("extractDottedRefs(%q)[%d] = %q, want %q", tt.expr, i, got[i], tt.expected[i])
			}
		}
	}
}

func TestLookupNestedString(t *testing.T) {
	m := map[string]any{
		"project": map[string]any{
			"organization": "acme",
		},
		"count": 42,
	}

	if v := lookupNestedString(m, "project.organization"); v != "acme" {
		t.Errorf("expected 'acme', got %q", v)
	}
	if v := lookupNestedString(m, "missing.path"); v != "" {
		t.Errorf("expected empty for missing path, got %q", v)
	}
	if v := lookupNestedString(m, "count"); v != "42" {
		t.Errorf("expected '42' for int value, got %q", v)
	}
}
