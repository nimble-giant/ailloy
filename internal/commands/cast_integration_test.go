package commands

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/nimble-giant/ailloy/pkg/blanks"
	"github.com/nimble-giant/ailloy/pkg/mold"
)

// nimbleMoldDir returns the absolute path to the nimble-mold/ directory.
func nimbleMoldDir(t *testing.T) string {
	t.Helper()
	_, filename, _, _ := runtime.Caller(0)
	// filename = .../internal/commands/cast_integration_test.go
	repoRoot := filepath.Join(filepath.Dir(filename), "..", "..")
	moldDir := filepath.Join(repoRoot, "nimble-mold")
	if _, err := os.Stat(moldDir); err != nil {
		t.Fatalf("nimble-mold directory not found at %s: %v", moldDir, err)
	}
	return moldDir
}

// testMoldReader creates a MoldReader from the nimble-mold/ directory.
func testMoldReader(t *testing.T) *blanks.MoldReader {
	t.Helper()
	reader, err := blanks.NewMoldReaderFromPath(nimbleMoldDir(t))
	if err != nil {
		t.Fatalf("failed to create mold reader: %v", err)
	}
	return reader
}

func TestIntegration_CopyBlankFiles(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	reader := testMoldReader(t)

	// Create required directory structure
	if err := os.MkdirAll(".claude/commands", 0750); err != nil {
		t.Fatalf("failed to create dirs: %v", err)
	}
	if err := os.MkdirAll(".claude/skills", 0750); err != nil {
		t.Fatalf("failed to create skills dir: %v", err)
	}

	err := copyBlankFiles(reader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = copySkillFiles(reader)
	if err != nil {
		t.Fatalf("unexpected error copying skills: %v", err)
	}

	// Verify all expected blanks were created (from manifest)
	manifest, err := reader.LoadManifest()
	if err != nil {
		t.Fatalf("failed to load manifest: %v", err)
	}

	for _, tmpl := range manifest.Commands {
		path := filepath.Join(".claude", "commands", tmpl)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("expected blank %s to be created: %v", tmpl, err)
			continue
		}

		content, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("failed to read %s: %v", tmpl, err)
			continue
		}
		if len(content) == 0 {
			t.Errorf("blank %s is empty", tmpl)
		}
	}

	for _, skill := range manifest.Skills {
		path := filepath.Join(".claude", "skills", skill)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("expected skill %s to be created: %v", skill, err)
			continue
		}

		content, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("failed to read skill %s: %v", skill, err)
			continue
		}
		if len(content) == 0 {
			t.Errorf("skill %s is empty", skill)
		}
	}
}

func TestIntegration_CopyBlankFiles_WithVariableSubstitution(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() {
		castValFiles = nil
		_ = os.Chdir(origDir)
	}()

	reader := testMoldReader(t)

	// Create a flux values file with overrides
	valuesContent := "project:\n  organization: test-org\n  board: TestBoard\n"
	valuesPath := filepath.Join(tmpDir, "values.yaml")
	if err := os.WriteFile(valuesPath, []byte(valuesContent), 0644); err != nil {
		t.Fatalf("failed to write values file: %v", err)
	}
	castValFiles = []string{valuesPath}

	// Create directory structure
	if err := os.MkdirAll(".claude/commands", 0750); err != nil {
		t.Fatalf("failed to create dirs: %v", err)
	}

	err := copyBlankFiles(reader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check that blanks exist and are non-empty
	entries, err := os.ReadDir(".claude/commands")
	if err != nil {
		t.Fatalf("failed to read commands dir: %v", err)
	}
	if len(entries) == 0 {
		t.Error("expected blanks to be copied")
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		path := filepath.Join(".claude", "commands", entry.Name())
		content, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("failed to read %s: %v", entry.Name(), err)
		}
		if len(content) == 0 {
			t.Errorf("%s is empty", entry.Name())
		}
	}
}

func TestIntegration_BlankFilesMatchMold(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	reader := testMoldReader(t)

	// Create directory structure
	if err := os.MkdirAll(".claude/commands", 0750); err != nil {
		t.Fatalf("failed to create dirs: %v", err)
	}
	if err := os.MkdirAll(".claude/skills", 0750); err != nil {
		t.Fatalf("failed to create skills dir: %v", err)
	}

	err := copyBlankFiles(reader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = copySkillFiles(reader)
	if err != nil {
		t.Fatalf("unexpected error copying skills: %v", err)
	}

	// Load flux defaults matching the layered flow in copyBlankFiles
	manifest, err := reader.LoadManifest()
	if err != nil {
		t.Fatalf("failed to load manifest: %v", err)
	}
	flux := mold.ApplyFluxDefaults(manifest.Flux, make(map[string]any))
	if fluxDefaults, err := reader.LoadFluxDefaults(); err == nil {
		flux = mold.ApplyFluxFileDefaults(fluxDefaults, flux)
	}

	// Verify each blank matches what the reader+template engine would produce
	for _, tmplName := range manifest.Commands {
		moldContent, err := reader.GetBlank(tmplName)
		if err != nil {
			t.Errorf("failed to get mold blank %s: %v", tmplName, err)
			continue
		}

		expectedContent, err := mold.ProcessTemplate(string(moldContent), flux)
		if err != nil {
			t.Errorf("failed to process blank %s: %v", tmplName, err)
			continue
		}

		filePath := filepath.Join(".claude", "commands", tmplName)
		fileContent, err := os.ReadFile(filePath)
		if err != nil {
			t.Errorf("failed to read copied blank %s: %v", tmplName, err)
			continue
		}

		if expectedContent != string(fileContent) {
			t.Errorf("blank %s content mismatch between processed mold and copied version", tmplName)
		}
	}

	for _, skillName := range manifest.Skills {
		moldContent, err := reader.GetSkill(skillName)
		if err != nil {
			t.Errorf("failed to get mold skill %s: %v", skillName, err)
			continue
		}

		expectedContent, err := mold.ProcessTemplate(string(moldContent), flux)
		if err != nil {
			t.Errorf("failed to process skill %s: %v", skillName, err)
			continue
		}

		filePath := filepath.Join(".claude", "skills", skillName)
		fileContent, err := os.ReadFile(filePath)
		if err != nil {
			t.Errorf("failed to read copied skill %s: %v", skillName, err)
			continue
		}

		if expectedContent != string(fileContent) {
			t.Errorf("skill %s content mismatch between processed mold and copied version", skillName)
		}
	}
}

func TestIntegration_CastProject_DirectoryCreation(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	dirs := []string{
		".claude",
		".claude/commands",
		".claude/skills",
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0750); err != nil {
			t.Fatalf("failed to create directory %s: %v", dir, err)
		}
	}

	for _, dir := range dirs {
		info, err := os.Stat(dir)
		if err != nil {
			t.Errorf("directory %s not created: %v", dir, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("expected %s to be a directory", dir)
		}
	}
}

func TestIntegration_CastGlobal_DirectoryCreation(t *testing.T) {
	tmpHome := t.TempDir()

	globalDir := filepath.Join(tmpHome, ".ailloy")
	dirs := []string{
		globalDir,
		filepath.Join(globalDir, "blanks"),
		filepath.Join(globalDir, "providers"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0750); err != nil {
			t.Fatalf("failed to create directory %s: %v", dir, err)
		}
	}

	for _, dir := range dirs {
		info, err := os.Stat(dir)
		if err != nil {
			t.Errorf("directory %s not created: %v", dir, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("expected %s to be a directory", dir)
		}
	}

	configPath := filepath.Join(globalDir, "ailloy.yaml")
	configContent := `user:
  name: "Test User"
  email: "test@example.com"
providers:
  claude:
    enabled: true
    api_key_env: "ANTHROPIC_API_KEY"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("failed to create config: %v", err)
	}

	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("config file not created: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("expected config permissions 0600, got %o", info.Mode().Perm())
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}
	if !strings.Contains(string(data), "claude") {
		t.Error("expected config to mention claude provider")
	}
}

func TestIntegration_BlankFilePermissions(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	reader := testMoldReader(t)

	if err := os.MkdirAll(".claude/commands", 0750); err != nil {
		t.Fatalf("failed to create dirs: %v", err)
	}
	if err := os.MkdirAll(".claude/skills", 0750); err != nil {
		t.Fatalf("failed to create skills dir: %v", err)
	}

	err := copyBlankFiles(reader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = copySkillFiles(reader)
	if err != nil {
		t.Fatalf("unexpected error copying skills: %v", err)
	}

	checkPermissions := func(dir string) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("failed to read dir %s: %v", dir, err)
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			path := filepath.Join(dir, entry.Name())
			info, err := os.Stat(path)
			if err != nil {
				t.Errorf("failed to stat %s: %v", entry.Name(), err)
				continue
			}
			perm := info.Mode().Perm()
			if perm != 0644 {
				t.Errorf("expected permissions 0644 for %s, got %o", path, perm)
			}
		}
	}

	checkPermissions(".claude/commands")
	checkPermissions(".claude/skills")
}

func TestIntegration_CastProject_DefaultSkipsWorkflows(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() {
		withWorkflows = false
		_ = os.Chdir(origDir)
	}()

	reader := testMoldReader(t)
	withWorkflows = false

	err := castProject(reader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := os.Stat(".claude/commands"); err != nil {
		t.Error("expected .claude/commands to be created")
	}

	if _, err := os.Stat(".github/workflows"); err == nil {
		t.Error("expected .github/workflows to NOT be created by default")
	}
}

func TestIntegration_CastProject_WithWorkflowsFlag(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() {
		withWorkflows = false
		_ = os.Chdir(origDir)
	}()

	reader := testMoldReader(t)
	withWorkflows = true

	err := castProject(reader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := os.Stat(".claude/commands"); err != nil {
		t.Error("expected .claude/commands to be created")
	}

	if _, err := os.Stat(".github/workflows"); err != nil {
		t.Error("expected .github/workflows to be created with --with-workflows")
	}

	wfPath := filepath.Join(".github", "workflows", "claude-code.yml")
	content, err := os.ReadFile(wfPath)
	if err != nil {
		t.Fatalf("expected claude-code.yml to be created: %v", err)
	}
	if len(content) == 0 {
		t.Error("workflow file is empty")
	}
	if !strings.Contains(string(content), "claude-code-action") {
		t.Error("workflow file should reference claude-code-action")
	}
}

func TestIntegration_CopyWorkflowBlanks(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	reader := testMoldReader(t)

	if err := os.MkdirAll(".github/workflows", 0750); err != nil {
		t.Fatalf("failed to create dirs: %v", err)
	}

	err := copyWorkflowBlanks(reader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	manifest, _ := reader.LoadManifest()
	for _, wf := range manifest.Workflows {
		path := filepath.Join(".github", "workflows", wf)
		content, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("expected workflow %s to be created: %v", wf, err)
			continue
		}
		if len(content) == 0 {
			t.Errorf("workflow %s is empty", wf)
		}
		if !strings.Contains(string(content), "name:") {
			t.Errorf("workflow %s does not contain YAML name field", wf)
		}
	}
}

func TestIntegration_WorkflowFilesMatchMold(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	reader := testMoldReader(t)

	if err := os.MkdirAll(".github/workflows", 0750); err != nil {
		t.Fatalf("failed to create dirs: %v", err)
	}

	err := copyWorkflowBlanks(reader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	manifest, _ := reader.LoadManifest()
	for _, wfName := range manifest.Workflows {
		moldContent, err := reader.GetWorkflowBlank(wfName)
		if err != nil {
			t.Errorf("failed to get mold workflow %s: %v", wfName, err)
			continue
		}

		filePath := filepath.Join(".github", "workflows", wfName)
		fileContent, err := os.ReadFile(filePath)
		if err != nil {
			t.Errorf("failed to read copied workflow %s: %v", wfName, err)
			continue
		}

		if string(moldContent) != string(fileContent) {
			t.Errorf("workflow %s content mismatch between mold and copied version", wfName)
		}
	}
}
