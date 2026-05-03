package commands

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/nimble-giant/ailloy/pkg/blanks"
	"github.com/nimble-giant/ailloy/pkg/foundry"
	"github.com/nimble-giant/ailloy/pkg/mold"
)

// nimbleMoldRef is the remote reference used by integration tests.
const nimbleMoldRef = "github.com/nimble-giant/nimble-mold@v0.1.10"

// testMoldReader resolves the nimble-mold from its public repository.
func testMoldReader(t *testing.T) *blanks.MoldReader {
	t.Helper()
	fsys, err := foundry.Resolve(nimbleMoldRef)
	if err != nil {
		t.Fatalf("failed to resolve remote mold: %v", err)
	}
	return blanks.NewMoldReader(fsys)
}

// testFlux loads flux defaults from the mold reader for use in tests.
func testFlux(t *testing.T, reader *blanks.MoldReader) map[string]any {
	t.Helper()
	flux, err := reader.LoadFluxDefaults()
	if err != nil {
		t.Fatalf("failed to load flux defaults: %v", err)
	}
	return flux
}

func TestIntegration_CopyResolvedFiles(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	reader := testMoldReader(t)
	flux := testFlux(t, reader)

	manifest, err := reader.LoadManifest()
	if err != nil {
		t.Fatalf("failed to load manifest: %v", err)
	}

	resolved, err := mold.ResolveFiles(flux["output"], reader.FS())
	if err != nil {
		t.Fatalf("failed to resolve files: %v", err)
	}

	if len(resolved) == 0 {
		t.Fatal("expected resolved files, got none")
	}

	err = copyResolvedFiles(reader, manifest, flux, resolved, false)
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
	flux := testFlux(t, reader)

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

	resolved, err := mold.ResolveFiles(flux["output"], reader.FS())
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

	err = copyResolvedFiles(reader, manifest, flux, processable, false)
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
	flux := testFlux(t, reader)

	manifest, err := reader.LoadManifest()
	if err != nil {
		t.Fatalf("failed to load manifest: %v", err)
	}

	resolved, err := mold.ResolveFiles(flux["output"], reader.FS())
	if err != nil {
		t.Fatalf("failed to resolve files: %v", err)
	}

	err = copyResolvedFiles(reader, manifest, flux, resolved, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Load flux defaults matching the layered flow in copyResolvedFiles
	fluxForTemplate := mold.ApplyFluxDefaults(manifest.Flux, make(map[string]any))
	if fluxDefaults, err := reader.LoadFluxDefaults(); err == nil {
		fluxForTemplate = mold.ApplyFluxFileDefaults(fluxDefaults, fluxForTemplate)
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
			expectedContent, err = mold.ProcessTemplate(string(moldContent), fluxForTemplate)
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

func TestCopyResolvedFiles_SkipsEmptyRenderedFiles(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	moldFS := fstest.MapFS{
		"commands/empty.md": &fstest.MapFile{
			Data: []byte("{{- if false -}}content{{- end -}}"),
		},
		"commands/nonempty.md": &fstest.MapFile{
			Data: []byte("# Hello\nThis has content."),
		},
	}
	reader := blanks.NewMoldReader(moldFS)

	resolved := []mold.ResolvedFile{
		{SrcPath: "commands/empty.md", DestPath: filepath.Join(tmpDir, ".claude/commands/empty.md"), Process: true},
		{SrcPath: "commands/nonempty.md", DestPath: filepath.Join(tmpDir, ".claude/commands/nonempty.md"), Process: true},
	}

	err := copyResolvedFiles(reader, nil, map[string]any{}, resolved, false)
	if err != nil {
		t.Fatalf("copyResolvedFiles failed: %v", err)
	}

	// The empty-rendering file should NOT exist
	emptyPath := filepath.Join(tmpDir, ".claude/commands/empty.md")
	if _, err := os.Stat(emptyPath); err == nil {
		t.Error("expected empty-rendering file to be skipped, but it was written")
	}

	// The non-empty file SHOULD exist
	nonEmptyPath := filepath.Join(tmpDir, ".claude/commands/nonempty.md")
	info, err := os.Stat(nonEmptyPath)
	if err != nil {
		t.Fatalf("expected non-empty file to be written: %v", err)
	}
	if info.Size() == 0 {
		t.Error("non-empty file should have content")
	}
}

func TestCleanupEmptyDirs_RemovesEmptyAndAncestors(t *testing.T) {
	tmp := t.TempDir()

	emptyLeaf := filepath.Join(tmp, ".claude", "agents")
	keepLeaf := filepath.Join(tmp, ".opencode", "agents")
	if err := os.MkdirAll(emptyLeaf, 0750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.MkdirAll(keepLeaf, 0750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(keepLeaf, "agent.md"), []byte("hi"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	dirs := []string{emptyLeaf, keepLeaf}
	remaining := cleanupEmptyDirs(dirs, tmp)

	if _, err := os.Stat(emptyLeaf); !os.IsNotExist(err) {
		t.Errorf("expected empty leaf %s to be removed, got err=%v", emptyLeaf, err)
	}
	if _, err := os.Stat(filepath.Join(tmp, ".claude")); !os.IsNotExist(err) {
		t.Errorf("expected empty ancestor .claude to be removed")
	}
	if _, err := os.Stat(keepLeaf); err != nil {
		t.Errorf("expected non-empty leaf %s to remain: %v", keepLeaf, err)
	}
	if _, err := os.Stat(tmp); err != nil {
		t.Errorf("expected destPrefix %s to remain: %v", tmp, err)
	}

	if len(remaining) != 1 || remaining[0] != keepLeaf {
		t.Errorf("expected remaining=[%s], got %v", keepLeaf, remaining)
	}
}

func TestCleanupEmptyDirs_PreservesNonEmptyAncestors(t *testing.T) {
	tmp := t.TempDir()

	leaf := filepath.Join(tmp, ".claude", "agents")
	if err := os.MkdirAll(leaf, 0750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	// Pre-existing sibling content under .claude/ that the cast did not produce.
	sibling := filepath.Join(tmp, ".claude", "settings.json")
	if err := os.WriteFile(sibling, []byte("{}"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	cleanupEmptyDirs([]string{leaf}, tmp)

	if _, err := os.Stat(leaf); !os.IsNotExist(err) {
		t.Errorf("expected empty leaf to be removed")
	}
	if _, err := os.Stat(filepath.Join(tmp, ".claude")); err != nil {
		t.Errorf("expected non-empty ancestor .claude to remain: %v", err)
	}
}

func TestCopyResolvedFiles_RemovesEmptyMultiDestDirs(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	moldFS := fstest.MapFS{
		"agents/foo.md": &fstest.MapFile{
			Data: []byte("{{- if eq .target \"opencode\" -}}# foo{{- end -}}"),
		},
	}
	reader := blanks.NewMoldReader(moldFS)

	resolved := []mold.ResolvedFile{
		{
			SrcPath:  "agents/foo.md",
			DestPath: ".claude/agents/foo.md",
			Process:  true,
			Set:      map[string]any{"target": "claude"},
		},
		{
			SrcPath:  "agents/foo.md",
			DestPath: ".opencode/agents/foo.md",
			Process:  true,
			Set:      map[string]any{"target": "opencode"},
		},
	}

	dirs := []string{".claude/agents", ".opencode/agents"}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0750); err != nil {
			t.Fatalf("mkdir %s: %v", d, err)
		}
	}

	if err := copyResolvedFiles(reader, nil, map[string]any{}, resolved, false); err != nil {
		t.Fatalf("copyResolvedFiles failed: %v", err)
	}

	remaining := cleanupEmptyDirs(dirs, "")

	if _, err := os.Stat(".claude"); !os.IsNotExist(err) {
		t.Errorf("expected .claude to be removed (all renders empty)")
	}
	if _, err := os.Stat(".opencode/agents/foo.md"); err != nil {
		t.Errorf("expected .opencode/agents/foo.md to be written: %v", err)
	}

	if len(remaining) != 1 || remaining[0] != ".opencode/agents" {
		t.Errorf("expected remaining=[.opencode/agents], got %v", remaining)
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

	err := castProject(reader, "")
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
	flux := testFlux(t, reader)

	manifest, err := reader.LoadManifest()
	if err != nil {
		t.Fatalf("failed to load manifest: %v", err)
	}

	resolved, err := mold.ResolveFiles(flux["output"], reader.FS())
	if err != nil {
		t.Fatalf("failed to resolve files: %v", err)
	}

	err = copyResolvedFiles(reader, manifest, flux, resolved, false)
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

	err := castProject(reader, "")
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

	err := castProject(reader, "")
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
	flux := testFlux(t, reader)
	withWorkflows = true

	manifest, err := reader.LoadManifest()
	if err != nil {
		t.Fatalf("failed to load manifest: %v", err)
	}

	resolved, err := mold.ResolveFiles(flux["output"], reader.FS())
	if err != nil {
		t.Fatalf("failed to resolve files: %v", err)
	}

	err = copyResolvedFiles(reader, manifest, flux, resolved, false)
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

func TestIntegration_MergeStrategy_JSON(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	// Pre-seed an existing opencode.json with the "outline" MCP server.
	preExisting := []byte(`{"mcp":{"outline":{"url":"https://outline"}}}`)
	if err := os.WriteFile("opencode.json", preExisting, 0644); err != nil {
		t.Fatalf("seed: %v", err)
	}

	// Build a synthetic mold that produces opencode.json with strategy: merge.
	moldFS := fstest.MapFS{
		"mold.yaml": &fstest.MapFile{Data: []byte("apiVersion: v1\nkind: Mold\nname: test\nversion: 0.1.0\n")},
		"flux.yaml": &fstest.MapFile{Data: []byte(`output:
  config/opencode.json:
    dest: opencode.json
    strategy: merge
`)},
		"config/opencode.json": &fstest.MapFile{Data: []byte(`{"mcp":{"replicated-docs":{"url":"https://docs"}}}`)},
	}

	reader := blanks.NewMoldReader(moldFS)
	manifest, err := reader.LoadManifest()
	if err != nil {
		t.Fatalf("load manifest: %v", err)
	}
	flux, err := reader.LoadFluxDefaults()
	if err != nil {
		t.Fatalf("load flux: %v", err)
	}
	resolved, err := mold.ResolveFiles(flux["output"], reader.FS())
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if err := copyResolvedFiles(reader, manifest, flux, resolved, false); err != nil {
		t.Fatalf("copy: %v", err)
	}

	got, err := os.ReadFile("opencode.json")
	if err != nil {
		t.Fatalf("readback: %v", err)
	}
	gs := string(got)
	if !strings.Contains(gs, `"outline"`) {
		t.Errorf("outline missing after merge:\n%s", gs)
	}
	if !strings.Contains(gs, `"replicated-docs"`) {
		t.Errorf("replicated-docs missing after merge:\n%s", gs)
	}
	// Order: outline comes first (from pre-existing file).
	if strings.Index(gs, "outline") > strings.Index(gs, "replicated-docs") {
		t.Errorf("expected outline before replicated-docs, got:\n%s", gs)
	}
}

func TestIntegration_MergeStrategy_TwoMolds(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	moldA := fstest.MapFS{
		"mold.yaml": &fstest.MapFile{Data: []byte("apiVersion: v1\nkind: Mold\nname: a\nversion: 0.1.0\n")},
		"flux.yaml": &fstest.MapFile{Data: []byte(`output:
  config/opencode.json:
    dest: opencode.json
    strategy: merge
`)},
		"config/opencode.json": &fstest.MapFile{Data: []byte(`{"mcp":{"outline":{"url":"https://outline"}}}`)},
	}
	moldB := fstest.MapFS{
		"mold.yaml": &fstest.MapFile{Data: []byte("apiVersion: v1\nkind: Mold\nname: b\nversion: 0.1.0\n")},
		"flux.yaml": &fstest.MapFile{Data: []byte(`output:
  config/opencode.json:
    dest: opencode.json
    strategy: merge
`)},
		"config/opencode.json": &fstest.MapFile{Data: []byte(`{"mcp":{"replicated-docs":{"url":"https://docs"}}}`)},
	}

	for _, m := range []fstest.MapFS{moldA, moldB} {
		reader := blanks.NewMoldReader(m)
		manifest, err := reader.LoadManifest()
		if err != nil {
			t.Fatalf("load manifest: %v", err)
		}
		flux, err := reader.LoadFluxDefaults()
		if err != nil {
			t.Fatalf("load flux: %v", err)
		}
		resolved, err := mold.ResolveFiles(flux["output"], reader.FS())
		if err != nil {
			t.Fatalf("resolve: %v", err)
		}
		if err := copyResolvedFiles(reader, manifest, flux, resolved, false); err != nil {
			t.Fatalf("copy: %v", err)
		}
	}

	got, err := os.ReadFile("opencode.json")
	if err != nil {
		t.Fatalf("readback: %v", err)
	}
	gs := string(got)
	if !strings.Contains(gs, `"outline"`) {
		t.Errorf("mold A's outline lost:\n%s", gs)
	}
	if !strings.Contains(gs, `"replicated-docs"`) {
		t.Errorf("mold B's replicated-docs missing:\n%s", gs)
	}
}

func TestIntegration_MergeStrategy_ForceReplaceOnParseError(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	// Pre-seed an unparseable opencode.json (simulating user hand-edits).
	if err := os.WriteFile("opencode.json", []byte("not json{"), 0644); err != nil {
		t.Fatalf("seed: %v", err)
	}

	moldFS := fstest.MapFS{
		"mold.yaml": &fstest.MapFile{Data: []byte("apiVersion: v1\nkind: Mold\nname: test\nversion: 0.1.0\n")},
		"flux.yaml": &fstest.MapFile{Data: []byte(`output:
  config/opencode.json:
    dest: opencode.json
    strategy: merge
`)},
		"config/opencode.json": &fstest.MapFile{Data: []byte(`{"mcp":{"replicated-docs":{"url":"https://docs"}}}`)},
	}
	reader := blanks.NewMoldReader(moldFS)
	manifest, err := reader.LoadManifest()
	if err != nil {
		t.Fatalf("load manifest: %v", err)
	}
	flux, err := reader.LoadFluxDefaults()
	if err != nil {
		t.Fatalf("load flux: %v", err)
	}
	resolved, err := mold.ResolveFiles(flux["output"], reader.FS())
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}

	// Without force-replace: should fail with ParseError-derived message.
	err = copyResolvedFiles(reader, manifest, flux, resolved, false)
	if err == nil {
		t.Fatal("expected error without force-replace, got nil")
	}
	if !strings.Contains(err.Error(), "force-replace-on-parse-error") {
		t.Errorf("expected error to mention the flag; got: %v", err)
	}

	// With force-replace: should clobber the unparseable file.
	if err := copyResolvedFiles(reader, manifest, flux, resolved, true); err != nil {
		t.Fatalf("force-replace should succeed; got: %v", err)
	}
	got, _ := os.ReadFile("opencode.json")
	if !strings.Contains(string(got), "replicated-docs") {
		t.Errorf("force-replace should write new content; got: %s", got)
	}
}

func TestIntegration_Forge_MergeStrategy(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	outputDir := filepath.Join(tmpDir, "preview")
	if err := os.MkdirAll(outputDir, 0750); err != nil {
		t.Fatal(err)
	}
	// Pre-seed an existing opencode.json under the output dir to simulate an
	// iterative forge preview.
	dest := filepath.Join(outputDir, "opencode.json")
	if err := os.WriteFile(dest, []byte(`{"mcp":{"outline":{"url":"https://outline"}}}`), 0644); err != nil {
		t.Fatal(err)
	}

	files := []renderedFile{
		{
			destPath: "opencode.json",
			content:  `{"mcp":{"replicated-docs":{"url":"https://docs"}}}`,
			strategy: "merge",
		},
	}
	if err := writeForgeFiles(files, outputDir, false); err != nil {
		t.Fatalf("writeForgeFiles: %v", err)
	}
	got, _ := os.ReadFile(dest)
	gs := string(got)
	if !strings.Contains(gs, "outline") || !strings.Contains(gs, "replicated-docs") {
		t.Errorf("forge merge lost an entry, got: %s", gs)
	}
}
