package commands

import (
	"io/fs"
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

func TestIntegration_CopyResolvedFiles(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	reader := testMoldReader(t)

	manifest, err := reader.LoadManifest()
	if err != nil {
		t.Fatalf("failed to load manifest: %v", err)
	}

	resolved, err := mold.ResolveFiles(manifest, reader.FS())
	if err != nil {
		t.Fatalf("failed to resolve files: %v", err)
	}

	if len(resolved) == 0 {
		t.Fatal("expected resolved files, got none")
	}

	err = copyResolvedFiles(reader, manifest, resolved)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify all resolved files were created
	for _, rf := range resolved {
		if _, err := os.Stat(rf.DestPath); err != nil {
			t.Errorf("expected file %s to be created: %v", rf.DestPath, err)
			continue
		}

		content, err := os.ReadFile(rf.DestPath)
		if err != nil {
			t.Errorf("failed to read %s: %v", rf.DestPath, err)
			continue
		}
		if len(content) == 0 {
			t.Errorf("file %s is empty", rf.DestPath)
		}
	}
}

func TestIntegration_CopyResolvedFiles_WithVariableSubstitution(t *testing.T) {
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

	manifest, err := reader.LoadManifest()
	if err != nil {
		t.Fatalf("failed to load manifest: %v", err)
	}

	resolved, err := mold.ResolveFiles(manifest, reader.FS())
	if err != nil {
		t.Fatalf("failed to resolve files: %v", err)
	}

	// Filter to only processable files for this test
	var processable []mold.ResolvedFile
	for _, rf := range resolved {
		if rf.Process {
			processable = append(processable, rf)
		}
	}

	err = copyResolvedFiles(reader, manifest, processable)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check that files exist and are non-empty
	for _, rf := range processable {
		content, err := os.ReadFile(rf.DestPath)
		if err != nil {
			t.Errorf("failed to read %s: %v", rf.DestPath, err)
			continue
		}
		if len(content) == 0 {
			t.Errorf("%s is empty", rf.DestPath)
		}
	}
}

func TestIntegration_ResolvedFilesMatchMold(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	reader := testMoldReader(t)

	manifest, err := reader.LoadManifest()
	if err != nil {
		t.Fatalf("failed to load manifest: %v", err)
	}

	resolved, err := mold.ResolveFiles(manifest, reader.FS())
	if err != nil {
		t.Fatalf("failed to resolve files: %v", err)
	}

	err = copyResolvedFiles(reader, manifest, resolved)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Load flux defaults matching the layered flow in copyResolvedFiles
	flux := mold.ApplyFluxDefaults(manifest.Flux, make(map[string]any))
	if fluxDefaults, err := reader.LoadFluxDefaults(); err == nil {
		flux = mold.ApplyFluxFileDefaults(fluxDefaults, flux)
	}

	// Verify each file matches what the reader+template engine would produce
	for _, rf := range resolved {
		moldContent, err := fs.ReadFile(reader.FS(), rf.SrcPath)
		if err != nil {
			t.Errorf("failed to read mold file %s: %v", rf.SrcPath, err)
			continue
		}

		var expectedContent string
		if rf.Process {
			expectedContent, err = mold.ProcessTemplate(string(moldContent), flux)
			if err != nil {
				t.Errorf("failed to process %s: %v", rf.SrcPath, err)
				continue
			}
		} else {
			expectedContent = string(moldContent)
		}

		fileContent, err := os.ReadFile(rf.DestPath)
		if err != nil {
			t.Errorf("failed to read copied file %s: %v", rf.DestPath, err)
			continue
		}

		if expectedContent != string(fileContent) {
			t.Errorf("file %s content mismatch between processed mold and copied version", rf.DestPath)
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

	reader := testMoldReader(t)

	// castProject dynamically creates directories from the output mapping
	withWorkflows = true
	defer func() { withWorkflows = false }()

	err := castProject(reader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify expected output directories were created
	expectedDirs := []string{
		".claude/commands",
		".claude/skills",
		".github/workflows",
	}

	for _, dir := range expectedDirs {
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

func TestIntegration_FilePermissions(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	reader := testMoldReader(t)

	manifest, err := reader.LoadManifest()
	if err != nil {
		t.Fatalf("failed to load manifest: %v", err)
	}

	resolved, err := mold.ResolveFiles(manifest, reader.FS())
	if err != nil {
		t.Fatalf("failed to resolve files: %v", err)
	}

	err = copyResolvedFiles(reader, manifest, resolved)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, rf := range resolved {
		info, err := os.Stat(rf.DestPath)
		if err != nil {
			t.Errorf("failed to stat %s: %v", rf.DestPath, err)
			continue
		}
		perm := info.Mode().Perm()
		if perm != 0644 {
			t.Errorf("expected permissions 0644 for %s, got %o", rf.DestPath, perm)
		}
	}
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

func TestIntegration_WorkflowsNotProcessed(t *testing.T) {
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

	manifest, err := reader.LoadManifest()
	if err != nil {
		t.Fatalf("failed to load manifest: %v", err)
	}

	resolved, err := mold.ResolveFiles(manifest, reader.FS())
	if err != nil {
		t.Fatalf("failed to resolve files: %v", err)
	}

	err = copyResolvedFiles(reader, manifest, resolved)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Workflow files should be copied as-is (process: false)
	for _, rf := range resolved {
		if !strings.HasPrefix(rf.DestPath, ".github/workflows") {
			continue
		}

		moldContent, err := fs.ReadFile(reader.FS(), rf.SrcPath)
		if err != nil {
			t.Errorf("failed to read mold file %s: %v", rf.SrcPath, err)
			continue
		}

		fileContent, err := os.ReadFile(rf.DestPath)
		if err != nil {
			t.Errorf("failed to read copied file %s: %v", rf.DestPath, err)
			continue
		}

		// Since process: false, content should match exactly
		if string(moldContent) != string(fileContent) {
			t.Errorf("workflow %s should be copied verbatim (process: false) but content differs", rf.DestPath)
		}
	}
}
