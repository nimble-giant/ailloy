package assay

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nimble-giant/ailloy/pkg/mold"
)

// --- skills ---

func TestCommandFrontmatterRule_PluginSkill(t *testing.T) {
	ctx := &RuleContext{
		Files: []DetectedFile{{
			Path:      "myplugin/skills/helper.md",
			Platform:  PlatformClaude,
			Content:   []byte("---\nunknown-field: value\n---\n# Skill\n"),
			PluginDir: "myplugin",
		}},
		Config: DefaultConfig(),
	}
	rule := &commandFrontmatterRule{}
	diags := rule.Check(ctx)
	if len(diags) == 0 {
		t.Error("expected diagnostic for unknown field in plugin skill frontmatter")
	}
}

func TestCommandFrontmatterRule_PluginSkillValidFrontmatter(t *testing.T) {
	ctx := &RuleContext{
		Files: []DetectedFile{{
			Path:      "myplugin/skills/helper.md",
			Platform:  PlatformClaude,
			Content:   []byte("---\nallowed-tools: [Read]\n---\n# Skill\n"),
			PluginDir: "myplugin",
		}},
		Config: DefaultConfig(),
	}
	rule := &commandFrontmatterRule{}
	diags := rule.Check(ctx)
	if len(diags) != 0 {
		t.Errorf("expected no diagnostics, got: %v", diags)
	}
}

// --- rules ---

func TestDetectPluginFiles_RulesDir(t *testing.T) {
	dir := t.TempDir()
	manifest := `{"name":"p","version":"1.0.0","description":"A plugin"}`
	pluginRoot := filepath.Join(dir, "my-plugin")
	writeFile(t, pluginRoot, filepath.Join(".claude-plugin", "plugin.json"), manifest)
	writeFile(t, pluginRoot, filepath.Join("rules", "style.md"), "# Style\nFollow these rules.\n")

	files := detectPluginFiles(dir)

	var found bool
	for _, f := range files {
		if filepath.Base(f.Path) == "style.md" {
			found = true
		}
	}
	if !found {
		t.Error("expected rules/style.md to be detected")
	}
}

// --- hooks ---

func TestPluginHooksRule_Valid(t *testing.T) {
	content := `{"hooks":[{"name":"auth-check","event":"command:before","pattern":"/create-issue","action":"validate-auth"}]}`
	ctx := &RuleContext{
		Files: []DetectedFile{{
			Path:      "myplugin/hooks/hooks.json",
			Platform:  PlatformClaude,
			Content:   []byte(content),
			PluginDir: "myplugin",
		}},
		Config: DefaultConfig(),
	}
	rule := &pluginHooksRule{}
	diags := rule.Check(ctx)
	if len(diags) != 0 {
		t.Errorf("expected no diagnostics for valid hooks, got: %v", diags)
	}
}

func TestPluginHooksRule_InvalidJSON(t *testing.T) {
	ctx := &RuleContext{
		Files: []DetectedFile{{
			Path:      "myplugin/hooks/hooks.json",
			Platform:  PlatformClaude,
			Content:   []byte(`{invalid`),
			PluginDir: "myplugin",
		}},
		Config: DefaultConfig(),
	}
	rule := &pluginHooksRule{}
	diags := rule.Check(ctx)
	if len(diags) == 0 {
		t.Error("expected diagnostic for invalid JSON")
	}
	if diags[0].Severity != mold.SeverityError {
		t.Errorf("expected error severity, got %v", diags[0].Severity)
	}
}

func TestPluginHooksRule_MissingHooksKey(t *testing.T) {
	ctx := &RuleContext{
		Files: []DetectedFile{{
			Path:      "myplugin/hooks/hooks.json",
			Platform:  PlatformClaude,
			Content:   []byte(`{"other":"value"}`),
			PluginDir: "myplugin",
		}},
		Config: DefaultConfig(),
	}
	rule := &pluginHooksRule{}
	diags := rule.Check(ctx)
	if len(diags) == 0 {
		t.Error("expected warning for missing hooks key")
	}
}

func TestPluginHooksRule_MissingRequiredHookFields(t *testing.T) {
	content := `{"hooks":[{"name":"my-hook"}]}` // missing "event"
	ctx := &RuleContext{
		Files: []DetectedFile{{
			Path:      "myplugin/hooks/hooks.json",
			Platform:  PlatformClaude,
			Content:   []byte(content),
			PluginDir: "myplugin",
		}},
		Config: DefaultConfig(),
	}
	rule := &pluginHooksRule{}
	diags := rule.Check(ctx)
	if len(diags) == 0 {
		t.Error("expected diagnostic for missing event field")
	}
}

func TestPluginHooksRule_IgnoresNonPluginFiles(t *testing.T) {
	ctx := &RuleContext{
		Files: []DetectedFile{{
			Path:    ".claude/settings.json",
			Content: []byte(`{}`),
			// PluginDir empty
		}},
		Config: DefaultConfig(),
	}
	rule := &pluginHooksRule{}
	diags := rule.Check(ctx)
	if len(diags) != 0 {
		t.Errorf("expected no diagnostics for non-plugin file, got: %v", diags)
	}
}

// --- prefix matching (isUnderPluginSubdir) ---

func TestIsUnderPluginSubdir(t *testing.T) {
	cases := []struct {
		path, pluginDir, subdir string
		want                    bool
	}{
		{"myplugin/commands/cmd.md", "myplugin", "commands", true},
		{"myplugin/commands/sub/cmd.md", "myplugin", "commands", true},
		{"myplugin/skills/s.md", "myplugin", "commands", false},
		{"other/commands/cmd.md", "myplugin", "commands", false},
	}
	for _, c := range cases {
		got := isUnderPluginSubdir(c.path, c.pluginDir, c.subdir)
		if got != c.want {
			t.Errorf("isUnderPluginSubdir(%q, %q, %q) = %v, want %v", c.path, c.pluginDir, c.subdir, got, c.want)
		}
	}
}

// writePluginStructure creates a minimal Claude plugin directory under dir/pluginName.
func writePluginStructure(t *testing.T, dir, pluginName string, manifest, command string) string {
	t.Helper()
	pluginRoot := filepath.Join(dir, pluginName)

	writeFile(t, pluginRoot, filepath.Join(".claude-plugin", "plugin.json"), manifest)
	writeFile(t, pluginRoot, filepath.Join("commands", "test-cmd.md"), command)

	return pluginRoot
}

func TestDetectPluginFiles_Basic(t *testing.T) {
	dir := t.TempDir()
	manifest := `{"name":"my-plugin","version":"1.0.0","description":"Test plugin"}`
	writePluginStructure(t, dir, "my-plugin", manifest, "# Test Command\nDo the thing.\n")

	files := detectPluginFiles(dir)

	if len(files) == 0 {
		t.Fatal("expected plugin files to be detected")
	}

	var foundManifest, foundCommand bool
	for _, f := range files {
		if f.PluginDir == "" {
			t.Errorf("expected PluginDir to be set for %s", f.Path)
		}
		if filepath.Base(f.Path) == "plugin.json" {
			foundManifest = true
		}
		if filepath.Ext(f.Path) == ".md" {
			foundCommand = true
		}
	}
	if !foundManifest {
		t.Error("expected plugin manifest to be detected")
	}
	if !foundCommand {
		t.Error("expected plugin command file to be detected")
	}
}

func TestDetectPluginFiles_SkipsNonPluginDirs(t *testing.T) {
	dir := t.TempDir()
	// Create a non-plugin json file that shouldn't trigger detection
	writeFile(t, dir, "other/plugin.json", `{"not":"a-plugin"}`)

	files := detectPluginFiles(dir)
	if len(files) != 0 {
		t.Errorf("expected 0 plugin files, got %d", len(files))
	}
}

func TestDetectPluginFiles_Marketplace(t *testing.T) {
	dir := t.TempDir()
	manifest := `{"name":"p","version":"1.0.0","description":"A plugin"}`

	// Three plugins at different depths in a marketplace
	writePluginStructure(t, dir, "plugins/auth-plugin", manifest, "# Auth\n")
	writePluginStructure(t, dir, "plugins/ui-plugin", manifest, "# UI\n")
	writePluginStructure(t, dir, "plugins/nested/deep-plugin", manifest, "# Deep\n")

	files := detectPluginFiles(dir)

	pluginDirs := make(map[string]bool)
	for _, f := range files {
		if f.PluginDir != "" {
			pluginDirs[f.PluginDir] = true
		}
	}

	for _, expected := range []string{
		filepath.Join("plugins", "auth-plugin"),
		filepath.Join("plugins", "ui-plugin"),
		filepath.Join("plugins", "nested", "deep-plugin"),
	} {
		if !pluginDirs[expected] {
			t.Errorf("expected plugin %q to be detected, got dirs: %v", expected, pluginDirs)
		}
	}
}

func TestDetectPluginFiles_NestedCommands(t *testing.T) {
	dir := t.TempDir()
	manifest := `{"name":"p","version":"1.0.0","description":"A plugin"}`
	pluginRoot := filepath.Join(dir, "my-plugin")
	writeFile(t, pluginRoot, filepath.Join(".claude-plugin", "plugin.json"), manifest)
	// Command in a subdirectory of commands/
	writeFile(t, pluginRoot, filepath.Join("commands", "sub", "nested-cmd.md"), "# Nested\n")

	files := detectPluginFiles(dir)

	var found bool
	for _, f := range files {
		if filepath.Base(f.Path) == "nested-cmd.md" {
			found = true
		}
	}
	if !found {
		t.Error("expected nested command file to be detected recursively")
	}
}

func TestDetectPluginFiles_PluginDir(t *testing.T) {
	dir := t.TempDir()
	manifest := `{"name":"pkg","version":"0.1.0","description":"A plugin"}`
	writePluginStructure(t, dir, "myplugin", manifest, "# Cmd\n")

	files := detectPluginFiles(dir)
	for _, f := range files {
		if f.PluginDir != "myplugin" {
			t.Errorf("expected PluginDir=myplugin, got %q for path %s", f.PluginDir, f.Path)
		}
	}
}

func TestAssay_PluginDirectory(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0755); err != nil {
		t.Fatal(err)
	}

	manifest := `{"name":"myplugin","version":"1.0.0","description":"Test plugin"}`
	writePluginStructure(t, dir, "myplugin", manifest, "# Command\nDo stuff.\n")

	result, err := Assay(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.FilesScanned == 0 {
		t.Error("expected plugin files to be included in scan")
	}
}

func TestPluginManifestRule_Valid(t *testing.T) {
	ctx := &RuleContext{
		Files: []DetectedFile{{
			Path:      "myplugin/.claude-plugin/plugin.json",
			Platform:  PlatformClaude,
			Content:   []byte(`{"name":"myplugin","version":"1.0.0","description":"A plugin"}`),
			PluginDir: "myplugin",
		}},
		Config: DefaultConfig(),
	}
	rule := &pluginManifestRule{}
	diags := rule.Check(ctx)
	if len(diags) != 0 {
		t.Errorf("expected no diagnostics for valid manifest, got: %v", diags)
	}
}

func TestPluginManifestRule_MissingFields(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		wantErrs int
	}{
		{"missing name", `{"version":"1.0.0","description":"A plugin"}`, 1},
		{"missing version", `{"name":"p","description":"A plugin"}`, 1},
		{"missing description", `{"name":"p","version":"1.0.0"}`, 1},
		{"missing all", `{}`, 3},
		{"invalid json", `{invalid`, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &RuleContext{
				Files: []DetectedFile{{
					Path:      "myplugin/.claude-plugin/plugin.json",
					Platform:  PlatformClaude,
					Content:   []byte(tt.content),
					PluginDir: "myplugin",
				}},
				Config: DefaultConfig(),
			}
			rule := &pluginManifestRule{}
			diags := rule.Check(ctx)
			errs := 0
			for _, d := range diags {
				if d.Severity == mold.SeverityError {
					errs++
				}
			}
			if errs != tt.wantErrs {
				t.Errorf("expected %d errors, got %d: %v", tt.wantErrs, errs, diags)
			}
		})
	}
}

func TestPluginManifestRule_IgnoresNonPluginFiles(t *testing.T) {
	ctx := &RuleContext{
		Files: []DetectedFile{{
			Path:     ".claude/settings.json",
			Platform: PlatformClaude,
			Content:  []byte(`{}`),
			// PluginDir is empty
		}},
		Config: DefaultConfig(),
	}
	rule := &pluginManifestRule{}
	diags := rule.Check(ctx)
	if len(diags) != 0 {
		t.Errorf("expected no diagnostics for non-plugin file, got: %v", diags)
	}
}

func TestCommandFrontmatterRule_PluginCommand(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		wantDiag bool
	}{
		{
			"valid plugin command frontmatter",
			"---\nallowed-tools: [Read]\n---\n# Command\n",
			false,
		},
		{
			"unknown field in plugin command",
			"---\nunknown-field: value\n---\n# Command\n",
			true,
		},
		{
			"no frontmatter in plugin command",
			"# Command\nJust content\n",
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &RuleContext{
				Files: []DetectedFile{{
					Path:      "myplugin/commands/test.md",
					Platform:  PlatformClaude,
					Content:   []byte(tt.content),
					PluginDir: "myplugin",
				}},
				Config: DefaultConfig(),
			}
			rule := &commandFrontmatterRule{}
			diags := rule.Check(ctx)
			if tt.wantDiag && len(diags) == 0 {
				t.Error("expected diagnostic, got none")
			}
			if !tt.wantDiag && len(diags) > 0 {
				t.Errorf("expected no diagnostic, got: %v", diags)
			}
		})
	}
}

func TestAgentFrontmatterRule_PluginAgent(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		wantErrs int
		wantWarn int
	}{
		{
			"valid plugin agent",
			"name: my-agent\ndescription: Does things\n",
			0, 0,
		},
		{
			"plugin agent missing name",
			"description: Does things\n",
			1, 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &RuleContext{
				Files: []DetectedFile{{
					Path:      "myplugin/agents/test.yml",
					Platform:  PlatformClaude,
					Content:   []byte(tt.content),
					PluginDir: "myplugin",
				}},
				Config: DefaultConfig(),
			}
			rule := &agentFrontmatterRule{}
			diags := rule.Check(ctx)
			errs, warns := 0, 0
			for _, d := range diags {
				if d.Severity == mold.SeverityError {
					errs++
				}
				if d.Severity == mold.SeverityWarning {
					warns++
				}
			}
			if errs != tt.wantErrs {
				t.Errorf("expected %d errors, got %d", tt.wantErrs, errs)
			}
			if warns != tt.wantWarn {
				t.Errorf("expected %d warnings, got %d", tt.wantWarn, warns)
			}
		})
	}
}
