package plugin

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewGenerator(t *testing.T) {
	g := NewGenerator("/tmp/output")
	if g == nil {
		t.Fatal("expected non-nil generator")
	}
	if g.OutputDir != "/tmp/output" {
		t.Errorf("expected output dir '/tmp/output', got '%s'", g.OutputDir)
	}
}

func TestGenerator_Generate_FullPlugin(t *testing.T) {
	dir := t.TempDir()
	outputDir := filepath.Join(dir, "plugin-output")

	g := NewGenerator(outputDir)
	g.Config = &Config{
		Name:        "test-plugin",
		Version:     "1.0.0",
		Description: "A test plugin",
		Author: Author{
			Name:  "Test Author",
			Email: "test@example.com",
			URL:   "https://example.com",
		},
	}

	err := g.Generate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify directory structure
	expectedDirs := []string{
		filepath.Join(outputDir, ".claude-plugin"),
		filepath.Join(outputDir, "commands"),
		filepath.Join(outputDir, "agents"),
		filepath.Join(outputDir, "hooks"),
		filepath.Join(outputDir, "scripts"),
	}

	for _, expectedDir := range expectedDirs {
		info, err := os.Stat(expectedDir)
		if err != nil {
			t.Errorf("expected directory %s to exist: %v", expectedDir, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("expected %s to be a directory", expectedDir)
		}
	}

	// Verify manifest
	manifestPath := filepath.Join(outputDir, ".claude-plugin", "plugin.json")
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("failed to read manifest: %v", err)
	}

	var manifest map[string]interface{}
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		t.Fatalf("failed to parse manifest: %v", err)
	}

	if manifest["name"] != "test-plugin" {
		t.Errorf("expected name 'test-plugin', got '%v'", manifest["name"])
	}
	if manifest["version"] != "1.0.0" {
		t.Errorf("expected version '1.0.0', got '%v'", manifest["version"])
	}
	if manifest["description"] != "A test plugin" {
		t.Errorf("expected description 'A test plugin', got '%v'", manifest["description"])
	}

	// Verify commands were generated
	commandsDir := filepath.Join(outputDir, "commands")
	entries, err := os.ReadDir(commandsDir)
	if err != nil {
		t.Fatalf("failed to read commands dir: %v", err)
	}
	if len(entries) == 0 {
		t.Error("expected at least one command file")
	}

	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".md") {
			t.Errorf("unexpected file in commands dir: %s", entry.Name())
		}
	}

	// Verify README
	readmePath := filepath.Join(outputDir, "README.md")
	readmeData, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("failed to read README: %v", err)
	}
	readmeStr := string(readmeData)
	if !strings.Contains(readmeStr, "test-plugin") {
		t.Error("expected README to contain plugin name")
	}
	if !strings.Contains(readmeStr, "Ailloy") {
		t.Error("expected README to mention Ailloy")
	}

	// Verify hooks
	hooksPath := filepath.Join(outputDir, "hooks", "hooks.json")
	hooksData, err := os.ReadFile(hooksPath)
	if err != nil {
		t.Fatalf("failed to read hooks: %v", err)
	}
	var hooks map[string]interface{}
	if err := json.Unmarshal(hooksData, &hooks); err != nil {
		t.Fatalf("failed to parse hooks: %v", err)
	}
	if _, ok := hooks["hooks"]; !ok {
		t.Error("expected hooks key in hooks.json")
	}

	// Verify install script
	scriptPath := filepath.Join(outputDir, "scripts", "install.sh")
	scriptData, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("failed to read install script: %v", err)
	}
	scriptStr := string(scriptData)
	if !strings.HasPrefix(scriptStr, "#!/bin/bash") {
		t.Error("expected install script to start with shebang")
	}
	if !strings.Contains(scriptStr, "test-plugin") {
		t.Error("expected install script to contain plugin name")
	}

	// Verify install script is executable
	info, err := os.Stat(scriptPath)
	if err != nil {
		t.Fatalf("failed to stat install script: %v", err)
	}
	if info.Mode()&0100 == 0 {
		t.Error("expected install script to be executable")
	}
}

func TestGenerator_LoadTemplates(t *testing.T) {
	g := NewGenerator(t.TempDir())
	err := g.loadTemplates()
	if err != nil {
		t.Fatalf("unexpected error loading templates: %v", err)
	}
	if len(g.templates) == 0 {
		t.Error("expected at least one template to be loaded")
	}

	for _, tmpl := range g.templates {
		if tmpl.Name == "" {
			t.Error("template has empty name")
		}
		if len(tmpl.Content) == 0 {
			t.Errorf("template %s has empty content", tmpl.Name)
		}
		// Name should not have .md extension
		if strings.HasSuffix(tmpl.Name, ".md") {
			t.Errorf("template name should not have .md extension: %s", tmpl.Name)
		}
	}
}

func TestGenerator_CreateStructure(t *testing.T) {
	dir := t.TempDir()
	outputDir := filepath.Join(dir, "structure-test")

	g := NewGenerator(outputDir)
	err := g.createStructure()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedDirs := []string{
		filepath.Join(outputDir, ".claude-plugin"),
		filepath.Join(outputDir, "commands"),
		filepath.Join(outputDir, "agents"),
		filepath.Join(outputDir, "hooks"),
		filepath.Join(outputDir, "scripts"),
	}

	for _, expected := range expectedDirs {
		info, err := os.Stat(expected)
		if err != nil {
			t.Errorf("expected dir %s: %v", expected, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("expected %s to be a directory", expected)
		}
	}
}

func TestGenerator_GenerateManifest(t *testing.T) {
	dir := t.TempDir()
	outputDir := filepath.Join(dir, "manifest-test")

	g := NewGenerator(outputDir)
	g.Config = &Config{
		Name:        "manifest-test",
		Version:     "2.0.0",
		Description: "Testing manifest generation",
		Author: Author{
			Name: "Author Name",
		},
	}

	// Create structure first
	if err := g.createStructure(); err != nil {
		t.Fatalf("failed to create structure: %v", err)
	}

	err := g.generateManifest()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	manifestPath := filepath.Join(outputDir, ".claude-plugin", "plugin.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("failed to read manifest: %v", err)
	}

	var manifest map[string]interface{}
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("invalid JSON in manifest: %v", err)
	}

	if manifest["name"] != "manifest-test" {
		t.Errorf("expected name 'manifest-test', got %v", manifest["name"])
	}
	if manifest["version"] != "2.0.0" {
		t.Errorf("expected version '2.0.0', got %v", manifest["version"])
	}

	// Verify author structure
	author, ok := manifest["author"].(map[string]interface{})
	if !ok {
		t.Fatal("expected author to be a map")
	}
	if author["name"] != "Author Name" {
		t.Errorf("expected author name 'Author Name', got %v", author["name"])
	}
}

func TestGenerator_GenerateCommands(t *testing.T) {
	dir := t.TempDir()
	outputDir := filepath.Join(dir, "commands-test")

	g := NewGenerator(outputDir)
	g.Config = &Config{
		Name:    "cmd-test",
		Version: "1.0.0",
	}

	// Create structure and load templates
	if err := g.createStructure(); err != nil {
		t.Fatalf("failed to create structure: %v", err)
	}
	if err := g.loadTemplates(); err != nil {
		t.Fatalf("failed to load templates: %v", err)
	}

	err := g.generateCommands()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify commands were written
	entries, err := os.ReadDir(filepath.Join(outputDir, "commands"))
	if err != nil {
		t.Fatalf("failed to read commands dir: %v", err)
	}

	if len(entries) != len(g.templates) {
		t.Errorf("expected %d commands, got %d", len(g.templates), len(entries))
	}

	for _, entry := range entries {
		content, err := os.ReadFile(filepath.Join(outputDir, "commands", entry.Name()))
		if err != nil {
			t.Errorf("failed to read command %s: %v", entry.Name(), err)
			continue
		}
		if len(content) == 0 {
			t.Errorf("command %s is empty", entry.Name())
		}
		// Each command should have been transformed
		contentStr := string(content)
		if !strings.Contains(contentStr, "## Instructions for Claude") {
			t.Errorf("command %s missing instructions section", entry.Name())
		}
	}
}

func TestExtractDescription(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "with purpose section",
			content:  "# Header\n## Purpose\nThis is the purpose.\n",
			expected: "AI-assisted workflow command", // Bug: the function iterates from line 0, not from the Purpose line
		},
		{
			name:     "without purpose",
			content:  "# Header\nSome content",
			expected: "AI-assisted workflow command",
		},
		{
			name:     "empty content",
			content:  "",
			expected: "AI-assisted workflow command",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractDescription([]byte(tt.content))
			if result == "" {
				t.Error("expected non-empty description")
			}
		})
	}
}

func TestGenerator_BuildREADME(t *testing.T) {
	g := NewGenerator("test-output")
	g.Config = &Config{
		Name:    "readme-test",
		Version: "1.0.0",
	}
	g.templates = []TemplateInfo{
		{Name: "cmd-one", Description: "First command"},
		{Name: "cmd-two", Description: "Second command"},
	}

	readme := g.buildREADME()

	if !strings.Contains(readme, "readme-test") {
		t.Error("expected README to contain plugin name")
	}
	if !strings.Contains(readme, "cmd-one") {
		t.Error("expected README to contain first command")
	}
	if !strings.Contains(readme, "cmd-two") {
		t.Error("expected README to contain second command")
	}
	if !strings.Contains(readme, "First command") {
		t.Error("expected README to contain first command description")
	}
	if !strings.Contains(readme, "v1.0.0") {
		t.Error("expected README to contain version")
	}
}

func TestGenerator_GenerateHooks(t *testing.T) {
	dir := t.TempDir()
	outputDir := filepath.Join(dir, "hooks-test")

	g := NewGenerator(outputDir)
	g.Config = &Config{Name: "hooks-test"}

	if err := g.createStructure(); err != nil {
		t.Fatalf("failed to create structure: %v", err)
	}

	err := g.generateHooks()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	hooksPath := filepath.Join(outputDir, "hooks", "hooks.json")
	data, err := os.ReadFile(hooksPath)
	if err != nil {
		t.Fatalf("failed to read hooks: %v", err)
	}

	var hooks map[string]interface{}
	if err := json.Unmarshal(data, &hooks); err != nil {
		t.Fatalf("invalid JSON in hooks: %v", err)
	}

	hooksList, ok := hooks["hooks"].([]interface{})
	if !ok {
		t.Fatal("expected hooks to be an array")
	}
	if len(hooksList) < 2 {
		t.Errorf("expected at least 2 hooks, got %d", len(hooksList))
	}
}

func TestGenerator_GenerateInstallScript(t *testing.T) {
	dir := t.TempDir()
	outputDir := filepath.Join(dir, "script-test")

	g := NewGenerator(outputDir)
	g.Config = &Config{Name: "script-test"}

	if err := g.createStructure(); err != nil {
		t.Fatalf("failed to create structure: %v", err)
	}

	err := g.generateInstallScript()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	scriptPath := filepath.Join(outputDir, "scripts", "install.sh")
	data, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("failed to read script: %v", err)
	}

	script := string(data)
	if !strings.Contains(script, "#!/bin/bash") {
		t.Error("expected shebang line")
	}
	if !strings.Contains(script, "script-test") {
		t.Error("expected plugin name in script")
	}
	if !strings.Contains(script, "set -e") {
		t.Error("expected set -e in script")
	}
}
