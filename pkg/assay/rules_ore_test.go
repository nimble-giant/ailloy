package assay

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeMoldWithOre lays out a minimal mold tree with an optional packaged ore
// at ores/<oreName>/ and an optional flux.schema.yaml. Both args may be empty
// to skip writing them.
func writeMoldWithOre(t *testing.T, root, oreName, fluxSchema string) {
	t.Helper()
	if err := os.MkdirAll(root, 0750); err != nil {
		t.Fatal(err)
	}
	moldYAML := "apiVersion: v1\nkind: mold\nname: test-mold\nversion: 1.0.0\n"
	if err := os.WriteFile(filepath.Join(root, "mold.yaml"), []byte(moldYAML), 0644); err != nil {
		t.Fatal(err)
	}
	if oreName != "" {
		oreDir := filepath.Join(root, "ores", oreName)
		if err := os.MkdirAll(oreDir, 0750); err != nil {
			t.Fatal(err)
		}
		oreYAML := "apiVersion: v1\nkind: ore\nname: " + oreName + "\nversion: 1.0.0\n"
		if err := os.WriteFile(filepath.Join(oreDir, "ore.yaml"), []byte(oreYAML), 0644); err != nil {
			t.Fatal(err)
		}
		schema := "- name: enabled\n  type: bool\n  default: \"false\"\n"
		if err := os.WriteFile(filepath.Join(oreDir, "flux.schema.yaml"), []byte(schema), 0644); err != nil {
			t.Fatal(err)
		}
		defaults := "enabled: false\n"
		if err := os.WriteFile(filepath.Join(oreDir, "flux.yaml"), []byte(defaults), 0644); err != nil {
			t.Fatal(err)
		}
	}
	if fluxSchema != "" {
		if err := os.WriteFile(filepath.Join(root, "flux.schema.yaml"), []byte(fluxSchema), 0644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestOreShadowingRule_FiresOnCollision(t *testing.T) {
	root := t.TempDir()
	schema := "- name: ore.status.enabled\n  type: bool\n- name: ore.status.field_id\n  type: string\n"
	writeMoldWithOre(t, root, "status", schema)

	rule := &oreShadowingRule{}
	ctx := &RuleContext{RootDir: root, Config: DefaultConfig()}
	diags := rule.Check(ctx)

	if len(diags) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d: %+v", len(diags), diags)
	}
	d := diags[0]
	if d.Rule != "ore-shadowing" {
		t.Errorf("Rule = %q, want ore-shadowing", d.Rule)
	}
	if d.File != "flux.schema.yaml" {
		t.Errorf("File = %q, want flux.schema.yaml", d.File)
	}
	if !strings.Contains(d.Message, "ore.status.") {
		t.Errorf("Message missing prefix: %q", d.Message)
	}
	if !strings.Contains(d.Message, "./ores/status/") {
		t.Errorf("Message missing ore path: %q", d.Message)
	}
}

func TestOreShadowingRule_QuietWhenOnlyOneSide(t *testing.T) {
	t.Run("only packaged ore", func(t *testing.T) {
		root := t.TempDir()
		writeMoldWithOre(t, root, "status", "- name: project.name\n  type: string\n")

		diags := (&oreShadowingRule{}).Check(&RuleContext{RootDir: root, Config: DefaultConfig()})
		if len(diags) != 0 {
			t.Fatalf("expected no diagnostics, got %+v", diags)
		}
	})

	t.Run("only hand-rolled entries", func(t *testing.T) {
		root := t.TempDir()
		schema := "- name: ore.status.enabled\n  type: bool\n"
		writeMoldWithOre(t, root, "", schema)

		diags := (&oreShadowingRule{}).Check(&RuleContext{RootDir: root, Config: DefaultConfig()})
		if len(diags) != 0 {
			t.Fatalf("expected no diagnostics, got %+v", diags)
		}
	})

	t.Run("packaged ore for unrelated namespace", func(t *testing.T) {
		root := t.TempDir()
		// status is packaged, but flux.schema.yaml only has ore.priority entries.
		schema := "- name: ore.priority.enabled\n  type: bool\n"
		writeMoldWithOre(t, root, "status", schema)

		diags := (&oreShadowingRule{}).Check(&RuleContext{RootDir: root, Config: DefaultConfig()})
		if len(diags) != 0 {
			t.Fatalf("expected no diagnostics, got %+v", diags)
		}
	})
}

func TestOreShadowingRule_OnlyFiresInsideMoldTree(t *testing.T) {
	root := t.TempDir()
	// Build the tree but omit mold.yaml so the rule should bail out.
	oreDir := filepath.Join(root, "ores", "status")
	if err := os.MkdirAll(oreDir, 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(oreDir, "ore.yaml"), []byte("apiVersion: v1\nkind: ore\nname: status\nversion: 1.0.0\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "flux.schema.yaml"), []byte("- name: ore.status.enabled\n  type: bool\n"), 0644); err != nil {
		t.Fatal(err)
	}

	diags := (&oreShadowingRule{}).Check(&RuleContext{RootDir: root, Config: DefaultConfig()})
	if len(diags) != 0 {
		t.Fatalf("expected no diagnostics outside a mold tree, got %+v", diags)
	}
}

func TestOreShadowingRule_MultipleNamespaces(t *testing.T) {
	root := t.TempDir()
	writeMoldWithOre(t, root, "status", "")
	// Add a second packaged ore at ores/priority/
	priorityDir := filepath.Join(root, "ores", "priority")
	if err := os.MkdirAll(priorityDir, 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(priorityDir, "ore.yaml"), []byte("apiVersion: v1\nkind: ore\nname: priority\nversion: 1.0.0\n"), 0644); err != nil {
		t.Fatal(err)
	}
	schema := "- name: ore.status.enabled\n  type: bool\n- name: ore.priority.enabled\n  type: bool\n"
	if err := os.WriteFile(filepath.Join(root, "flux.schema.yaml"), []byte(schema), 0644); err != nil {
		t.Fatal(err)
	}

	diags := (&oreShadowingRule{}).Check(&RuleContext{RootDir: root, Config: DefaultConfig()})
	if len(diags) != 2 {
		t.Fatalf("expected 2 diagnostics, got %d: %+v", len(diags), diags)
	}
	// Sorted by install dir: priority comes before status.
	if !strings.Contains(diags[0].Message, "ore.priority.") {
		t.Errorf("first diagnostic should mention priority, got %q", diags[0].Message)
	}
	if !strings.Contains(diags[1].Message, "ore.status.") {
		t.Errorf("second diagnostic should mention status, got %q", diags[1].Message)
	}
}

func TestOreShadowingRule_HonorsAlias(t *testing.T) {
	// When the install-dir name differs from ore.Name (alias case), the
	// install-dir name wins as the namespace.
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "mold.yaml"), []byte("apiVersion: v1\nkind: mold\nname: test-mold\nversion: 1.0.0\n"), 0644); err != nil {
		t.Fatal(err)
	}
	// Install dir is "github_status" but the underlying ore.Name is "status".
	oreDir := filepath.Join(root, "ores", "github_status")
	if err := os.MkdirAll(oreDir, 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(oreDir, "ore.yaml"), []byte("apiVersion: v1\nkind: ore\nname: status\nversion: 1.0.0\n"), 0644); err != nil {
		t.Fatal(err)
	}
	// flux.schema.yaml carries ore.github_status.* — should fire.
	schema := "- name: ore.github_status.enabled\n  type: bool\n- name: ore.status.enabled\n  type: bool\n"
	if err := os.WriteFile(filepath.Join(root, "flux.schema.yaml"), []byte(schema), 0644); err != nil {
		t.Fatal(err)
	}

	diags := (&oreShadowingRule{}).Check(&RuleContext{RootDir: root, Config: DefaultConfig()})
	if len(diags) != 1 {
		t.Fatalf("expected 1 diagnostic for alias, got %d: %+v", len(diags), diags)
	}
	if !strings.Contains(diags[0].Message, "ore.github_status.") {
		t.Errorf("expected diagnostic to use alias namespace, got %q", diags[0].Message)
	}
	if !strings.Contains(diags[0].Message, "./ores/github_status/") {
		t.Errorf("expected diagnostic to reference install dir, got %q", diags[0].Message)
	}
}

func TestOreShadowingRule_RegisteredAndSuppressible(t *testing.T) {
	// The rule must be registered so .ailloyrc.yaml can disable it by name.
	var found *oreShadowingRule
	for _, r := range AllRules() {
		if r.Name() == "ore-shadowing" {
			if cast, ok := r.(*oreShadowingRule); ok {
				found = cast
			}
			break
		}
	}
	if found == nil {
		t.Fatal("ore-shadowing rule not registered in global registry")
	}

	// Verify Config.IsRuleEnabled honors disabled config.
	disabled := false
	cfg := &Config{Rules: map[string]RuleConfig{
		"ore-shadowing": {Enabled: &disabled},
	}}
	if cfg.IsRuleEnabled("ore-shadowing") {
		t.Error("expected ore-shadowing to be disabled by config")
	}
}
