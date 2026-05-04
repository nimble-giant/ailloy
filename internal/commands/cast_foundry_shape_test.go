package commands

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/nimble-giant/ailloy/pkg/blanks"
	"github.com/nimble-giant/ailloy/pkg/mold"
)

// These tests replicate the exact mold shape used by
// github.com/kriscoleman/replicated-foundry's "wiki" and "docs" molds, which
// is the literal real-world scenario that motivated issue #171:
//
//   - Both molds declare an `mcp` output as a list of two entries (one for
//     each agent target — claude, opencode), each setting
//     `agent.current_target` so a per-target template gate selects the right
//     payload at render time.
//   - Both molds output to dest `.` (project root), producing `.mcp.json`
//     for claude and `opencode.json` for opencode.
//   - Casting both molds into the same project means the second cast writes
//     the SAME files the first did. Without strategy: merge, the second
//     mold's content clobbers the first.
//
// We replicate the shape as fstest.MapFS rather than fetching the live
// foundry: the foundry's molds don't yet declare `strategy: merge` (this PR
// introduces the field), and we want a deterministic test that exercises the
// merge feature against a realistic shape on every CI run.

// foundryWikiFlux mirrors molds/wiki/flux.yaml from
// kriscoleman/replicated-foundry, with `strategy: merge` added on each MCP
// list entry. The skills/mcp_server pieces from the real flux are dropped to
// keep the test focused on the merge surface.
const foundryWikiFlux = `agent:
  targets:
    - claude
    - opencode

output:
  mcp:
    - dest: .
      strategy: merge
      set:
        agent.current_target: claude
    - dest: .
      strategy: merge
      set:
        agent.current_target: opencode
`

const foundryDocsFlux = `agent:
  targets:
    - claude
    - opencode

output:
  mcp:
    - dest: .
      strategy: merge
      set:
        agent.current_target: claude
    - dest: .
      strategy: merge
      set:
        agent.current_target: opencode
`

// foundryWikiMcpClaude mirrors molds/wiki/mcp/.mcp.json from the real foundry.
const foundryWikiMcpClaude = `{{- if and (eq .agent.current_target "claude") (has "claude" .agent.targets) -}}
{
  "mcpServers": {
    "outline": {
      "command": "node",
      "args": [".wiki-mcp-server/dist/index.js"],
      "env": {
        "OUTLINE_API_KEY": ""
      }
    }
  }
}
{{- end -}}`

// foundryWikiMcpOpencode mirrors molds/wiki/mcp/opencode.json.
const foundryWikiMcpOpencode = `{{- if and (eq .agent.current_target "opencode") (has "opencode" .agent.targets) -}}
{
  "mcp": {
    "outline": {
      "type": "local",
      "command": ["node", ".wiki-mcp-server/dist/index.js"],
      "enabled": true,
      "environment": {
        "OUTLINE_API_KEY": ""
      }
    }
  }
}
{{- end -}}`

// foundryDocsMcpClaude mirrors molds/docs/mcp/.mcp.json.
const foundryDocsMcpClaude = `{{- if and (eq .agent.current_target "claude") (has "claude" .agent.targets) -}}
{
  "mcpServers": {
    "replicated-docs": {
      "type": "http",
      "url": "https://replicated-docs-mcp.justicepwhite.workers.dev/mcp"
    }
  }
}
{{- end -}}`

// foundryDocsMcpOpencode mirrors molds/docs/mcp/opencode.json.
const foundryDocsMcpOpencode = `{{- if and (eq .agent.current_target "opencode") (has "opencode" .agent.targets) -}}
{
  "mcp": {
    "replicated-docs": {
      "type": "remote",
      "url": "https://replicated-docs-mcp.justicepwhite.workers.dev/mcp",
      "enabled": true
    }
  }
}
{{- end -}}`

func castFoundryMold(t *testing.T, fs fstest.MapFS) {
	t.Helper()
	reader := blanks.NewMoldReader(fs)
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
	if err := copyResolvedFiles(reader, manifest, flux, resolved, copyOpts{}); err != nil {
		t.Fatalf("copy: %v", err)
	}
}

// TestIntegration_ReplicatedFoundryShape_WithMerge: literal acceptance
// criterion of issue #171. Two molds matching the kriscoleman/replicated-foundry
// wiki + docs shape, both casting into the same project with both
// agent.targets enabled. After both casts, .mcp.json and opencode.json must
// each contain both MCP servers (outline AND replicated-docs).
func TestIntegration_ReplicatedFoundryShape_WithMerge(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	wiki := fstest.MapFS{
		"mold.yaml":         &fstest.MapFile{Data: []byte("apiVersion: v1\nkind: Mold\nname: wiki\nversion: 0.1.0\n")},
		"flux.yaml":         &fstest.MapFile{Data: []byte(foundryWikiFlux)},
		"mcp/.mcp.json":     &fstest.MapFile{Data: []byte(foundryWikiMcpClaude)},
		"mcp/opencode.json": &fstest.MapFile{Data: []byte(foundryWikiMcpOpencode)},
	}
	docs := fstest.MapFS{
		"mold.yaml":         &fstest.MapFile{Data: []byte("apiVersion: v1\nkind: Mold\nname: docs\nversion: 0.1.0\n")},
		"flux.yaml":         &fstest.MapFile{Data: []byte(foundryDocsFlux)},
		"mcp/.mcp.json":     &fstest.MapFile{Data: []byte(foundryDocsMcpClaude)},
		"mcp/opencode.json": &fstest.MapFile{Data: []byte(foundryDocsMcpOpencode)},
	}

	// Cast in the order a foundries TUI would: wiki first, docs second.
	castFoundryMold(t, wiki)
	castFoundryMold(t, docs)

	// .mcp.json (claude target) must contain BOTH outline and replicated-docs.
	mcpBytes, err := os.ReadFile(".mcp.json")
	if err != nil {
		t.Fatalf("readback .mcp.json: %v", err)
	}
	mcp := map[string]any{}
	if err := json.Unmarshal(mcpBytes, &mcp); err != nil {
		t.Fatalf(".mcp.json invalid JSON after merge: %v\ncontent:\n%s", err, mcpBytes)
	}
	mcpServers, ok := mcp["mcpServers"].(map[string]any)
	if !ok {
		t.Fatalf(".mcp.json missing mcpServers key: %s", mcpBytes)
	}
	if _, ok := mcpServers["outline"]; !ok {
		t.Errorf("wiki's outline MCP server lost from .mcp.json:\n%s", mcpBytes)
	}
	if _, ok := mcpServers["replicated-docs"]; !ok {
		t.Errorf("docs's replicated-docs MCP server missing from .mcp.json:\n%s", mcpBytes)
	}

	// opencode.json must contain BOTH outline and replicated-docs under "mcp".
	openBytes, err := os.ReadFile("opencode.json")
	if err != nil {
		t.Fatalf("readback opencode.json: %v", err)
	}
	open := map[string]any{}
	if err := json.Unmarshal(openBytes, &open); err != nil {
		t.Fatalf("opencode.json invalid JSON after merge: %v\ncontent:\n%s", err, openBytes)
	}
	openServers, ok := open["mcp"].(map[string]any)
	if !ok {
		t.Fatalf("opencode.json missing mcp key: %s", openBytes)
	}
	if _, ok := openServers["outline"]; !ok {
		t.Errorf("wiki's outline MCP server lost from opencode.json:\n%s", openBytes)
	}
	if _, ok := openServers["replicated-docs"]; !ok {
		t.Errorf("docs's replicated-docs MCP server missing from opencode.json:\n%s", openBytes)
	}

	// Source order: wiki cast first, so outline must precede replicated-docs.
	mcpStr := string(mcpBytes)
	if strings.Index(mcpStr, "outline") > strings.Index(mcpStr, "replicated-docs") {
		t.Errorf("expected outline before replicated-docs in .mcp.json (base before overlay):\n%s", mcpStr)
	}
}

// TestIntegration_ReplicatedFoundryShape_WithoutMerge_DemonstratesClobbering:
// the SAME mold pair without strategy: merge declared. Documents the
// pre-fix behavior — the second cast clobbers the first — which is the
// regression we're protecting against.
//
// This is a baseline test: it asserts that without the merge feature, the
// motivating bug exists. If a future refactor accidentally enables merge
// implicitly (e.g., by changing the default), this test fails and we
// re-evaluate.
func TestIntegration_ReplicatedFoundryShape_WithoutMerge_DemonstratesClobbering(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	noMergeFlux := strings.ReplaceAll(foundryWikiFlux, "      strategy: merge\n", "")

	wiki := fstest.MapFS{
		"mold.yaml":         &fstest.MapFile{Data: []byte("apiVersion: v1\nkind: Mold\nname: wiki\nversion: 0.1.0\n")},
		"flux.yaml":         &fstest.MapFile{Data: []byte(noMergeFlux)},
		"mcp/.mcp.json":     &fstest.MapFile{Data: []byte(foundryWikiMcpClaude)},
		"mcp/opencode.json": &fstest.MapFile{Data: []byte(foundryWikiMcpOpencode)},
	}
	docs := fstest.MapFS{
		"mold.yaml":         &fstest.MapFile{Data: []byte("apiVersion: v1\nkind: Mold\nname: docs\nversion: 0.1.0\n")},
		"flux.yaml":         &fstest.MapFile{Data: []byte(noMergeFlux)},
		"mcp/.mcp.json":     &fstest.MapFile{Data: []byte(foundryDocsMcpClaude)},
		"mcp/opencode.json": &fstest.MapFile{Data: []byte(foundryDocsMcpOpencode)},
	}

	castFoundryMold(t, wiki)
	castFoundryMold(t, docs)

	mcpBytes, err := os.ReadFile(".mcp.json")
	if err != nil {
		t.Fatalf("readback .mcp.json: %v", err)
	}
	mcp := map[string]any{}
	if err := json.Unmarshal(mcpBytes, &mcp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	mcpServers := mcp["mcpServers"].(map[string]any)

	// Without merge: only the LAST cast's content survives.
	if _, ok := mcpServers["outline"]; ok {
		t.Errorf("regression: without strategy: merge, wiki's outline should have been clobbered, but it's still present:\n%s", mcpBytes)
	}
	if _, ok := mcpServers["replicated-docs"]; !ok {
		t.Errorf("docs's replicated-docs missing from .mcp.json after second cast:\n%s", mcpBytes)
	}
}

// TestIntegration_ReplicatedFoundryShape_SingleTargetOnly: variant where
// only one agent.target is enabled. The opencode list entry's template
// renders to empty content (the if-block is false), so it should be skipped.
// Only .mcp.json is produced. Verifies merge still works in the simpler
// single-target case.
func TestIntegration_ReplicatedFoundryShape_SingleTargetOnly(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	// Same flux but only claude target enabled.
	singleTargetFlux := strings.ReplaceAll(foundryWikiFlux, "    - opencode\n", "")

	wiki := fstest.MapFS{
		"mold.yaml":         &fstest.MapFile{Data: []byte("apiVersion: v1\nkind: Mold\nname: wiki\nversion: 0.1.0\n")},
		"flux.yaml":         &fstest.MapFile{Data: []byte(singleTargetFlux)},
		"mcp/.mcp.json":     &fstest.MapFile{Data: []byte(foundryWikiMcpClaude)},
		"mcp/opencode.json": &fstest.MapFile{Data: []byte(foundryWikiMcpOpencode)},
	}
	docs := fstest.MapFS{
		"mold.yaml":         &fstest.MapFile{Data: []byte("apiVersion: v1\nkind: Mold\nname: docs\nversion: 0.1.0\n")},
		"flux.yaml":         &fstest.MapFile{Data: []byte(singleTargetFlux)},
		"mcp/.mcp.json":     &fstest.MapFile{Data: []byte(foundryDocsMcpClaude)},
		"mcp/opencode.json": &fstest.MapFile{Data: []byte(foundryDocsMcpOpencode)},
	}

	castFoundryMold(t, wiki)
	castFoundryMold(t, docs)

	// Only .mcp.json should exist — opencode.json renders to empty.
	if _, err := os.Stat("opencode.json"); err == nil {
		t.Error("opencode.json should not exist when only claude target enabled")
	}

	mcpBytes, err := os.ReadFile(".mcp.json")
	if err != nil {
		t.Fatalf("readback: %v", err)
	}
	mcp := map[string]any{}
	if err := json.Unmarshal(mcpBytes, &mcp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	mcpServers := mcp["mcpServers"].(map[string]any)
	if _, ok := mcpServers["outline"]; !ok {
		t.Errorf("outline missing in single-target mode:\n%s", mcpBytes)
	}
	if _, ok := mcpServers["replicated-docs"]; !ok {
		t.Errorf("replicated-docs missing in single-target mode:\n%s", mcpBytes)
	}
}

// TestIntegration_MultiDest_DifferentDirs_SkipsEmptyRender: literal
// reproduction of issue #195. A single source file fans out to two
// different destination directories via per-dest `set` overrides; the
// template uses conditionals so only one destination's render produces
// content. The inactive destination's render is empty and must NOT be
// written as a zero-byte file.
func TestIntegration_MultiDest_DifferentDirs_SkipsEmptyRender(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	flux := `agent:
  targets:
    - opencode

output:
  agents:
    - dest: .claude/agents
      set:
        agent.current_target: claude
    - dest: .opencode/agents
      set:
        agent.current_target: opencode
`

	template := `{{- if and (eq .agent.current_target "claude") (has "claude" .agent.targets) -}}
---
name: coding-agent
model: opus
---
claude body
{{- else if and (eq .agent.current_target "opencode") (has "opencode" .agent.targets) -}}
---
description: coding agent
mode: primary
---
opencode body
{{- end -}}`

	moldFS := fstest.MapFS{
		"mold.yaml":              &fstest.MapFile{Data: []byte("apiVersion: v1\nkind: Mold\nname: agents\nversion: 0.1.0\n")},
		"flux.yaml":              &fstest.MapFile{Data: []byte(flux)},
		"agents/coding-agent.md": &fstest.MapFile{Data: []byte(template)},
	}

	castFoundryMold(t, moldFS)

	// Active destination must exist with content.
	openPath := filepath.Join(".opencode", "agents", "coding-agent.md")
	openBytes, err := os.ReadFile(openPath)
	if err != nil {
		t.Fatalf("expected %s to be created, got: %v", openPath, err)
	}
	if !strings.Contains(string(openBytes), "opencode body") {
		t.Errorf("expected opencode content, got:\n%s", openBytes)
	}

	// Inactive destination must NOT be written as a zero-byte file.
	claudePath := filepath.Join(".claude", "agents", "coding-agent.md")
	if info, err := os.Stat(claudePath); err == nil {
		t.Errorf("regression #195: %s should not exist (template renders empty for inactive target), but it does (size=%d)", claudePath, info.Size())
	}
}

// TestIntegration_ReplicatedFoundryShape_ThreeMolds: a more demanding scenario
// — three molds all writing to the same destination files. Verifies merge
// behaves transitively (mold C reads the result of mold A + mold B).
func TestIntegration_ReplicatedFoundryShape_ThreeMolds(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	supportClaude := `{{- if and (eq .agent.current_target "claude") (has "claude" .agent.targets) -}}
{
  "mcpServers": {
    "support": {
      "type": "http",
      "url": "https://support-mcp.example.com/mcp"
    }
  }
}
{{- end -}}`
	supportOpencode := `{{- if and (eq .agent.current_target "opencode") (has "opencode" .agent.targets) -}}
{
  "mcp": {
    "support": {
      "type": "remote",
      "url": "https://support-mcp.example.com/mcp",
      "enabled": true
    }
  }
}
{{- end -}}`

	for _, m := range []fstest.MapFS{
		{
			"mold.yaml":         &fstest.MapFile{Data: []byte("apiVersion: v1\nkind: Mold\nname: wiki\nversion: 0.1.0\n")},
			"flux.yaml":         &fstest.MapFile{Data: []byte(foundryWikiFlux)},
			"mcp/.mcp.json":     &fstest.MapFile{Data: []byte(foundryWikiMcpClaude)},
			"mcp/opencode.json": &fstest.MapFile{Data: []byte(foundryWikiMcpOpencode)},
		},
		{
			"mold.yaml":         &fstest.MapFile{Data: []byte("apiVersion: v1\nkind: Mold\nname: docs\nversion: 0.1.0\n")},
			"flux.yaml":         &fstest.MapFile{Data: []byte(foundryDocsFlux)},
			"mcp/.mcp.json":     &fstest.MapFile{Data: []byte(foundryDocsMcpClaude)},
			"mcp/opencode.json": &fstest.MapFile{Data: []byte(foundryDocsMcpOpencode)},
		},
		{
			"mold.yaml":         &fstest.MapFile{Data: []byte("apiVersion: v1\nkind: Mold\nname: support\nversion: 0.1.0\n")},
			"flux.yaml":         &fstest.MapFile{Data: []byte(foundryDocsFlux)},
			"mcp/.mcp.json":     &fstest.MapFile{Data: []byte(supportClaude)},
			"mcp/opencode.json": &fstest.MapFile{Data: []byte(supportOpencode)},
		},
	} {
		castFoundryMold(t, m)
	}

	mcpBytes, err := os.ReadFile(filepath.Join(".", ".mcp.json"))
	if err != nil {
		t.Fatalf("readback .mcp.json: %v", err)
	}
	mcp := map[string]any{}
	if err := json.Unmarshal(mcpBytes, &mcp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	servers := mcp["mcpServers"].(map[string]any)
	for _, want := range []string{"outline", "replicated-docs", "support"} {
		if _, ok := servers[want]; !ok {
			t.Errorf("server %q missing after three-mold cast:\n%s", want, mcpBytes)
		}
	}

	// Order: wiki (outline) < docs (replicated-docs) < support.
	mcpStr := string(mcpBytes)
	posOutline := strings.Index(mcpStr, "outline")
	posDocs := strings.Index(mcpStr, "replicated-docs")
	posSupport := strings.Index(mcpStr, "support")
	if posOutline >= posDocs || posDocs >= posSupport {
		t.Errorf("expected source order outline < replicated-docs < support, got positions %d/%d/%d in:\n%s", posOutline, posDocs, posSupport, mcpStr)
	}
}
