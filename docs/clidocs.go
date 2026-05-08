// Package clidocs embeds the project's markdown documentation so it can be
// rendered inside the ailloy CLI via `ailloy docs` and the `--docs` flag.
//
// The same files in this directory serve double duty: they are browsed on
// GitHub as the project's documentation site, and at build time they are
// embedded into the binary for terminal rendering through glamour.
//
// Discovery is recursive: any *.md file added under docs/ — including in
// subdirectories — is automatically picked up at build time and surfaced
// in the CLI without further wiring.
package clidocs

import (
	"embed"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"
)

// Embedded docs come from two roots:
//   - top-level docs/*.md  — canonical entry points (also browsed on GitHub)
//   - docs/topics/**       — a recursive subtree for grouped or nested docs
//
// The recursive embed (`all:topics`) discovers any depth of subdirectory and
// any new *.md file added under it automatically; no code changes required.
//
//go:embed *.md
//go:embed all:topics
var docsFS embed.FS

// Topic describes a single embedded documentation topic.
type Topic struct {
	// Slug is the user-facing identifier. For nested files it includes the
	// directory path (e.g. "guides/quickstart"). Always lowercase.
	Slug string
	// Title is the document's H1 heading, used for listings.
	Title string
	// Summary is a short one-line description for the topic listing.
	Summary string
	// File is the source path inside the embedded FS (e.g. "anneal.md" or
	// "guides/quickstart.md").
	File string
	// Dir is the parent directory inside the embedded FS, "" for top-level.
	Dir string
}

// summaries provides curated one-line descriptions per slug. The map falls
// back to the document's first non-heading paragraph when a slug is missing.
// New docs do NOT have to be listed here — they will get an auto-generated
// summary — but listing them yields tighter prose.
var summaries = map[string]string{
	"getting-started":    "Quickstart: install ailloy and cast your first mold",
	"blanks":             "Blanks: commands, skills, and workflow templates",
	"anneal":             "Configure flux variables interactively (alias: configure)",
	"flux":               "Template variable system, schemas, and value layering",
	"foundry":            "Resolve molds from git foundries and manage indexes",
	"smelt":              "Package molds into distributable tarballs or binaries",
	"temper":             "Validate molds and ingot packages",
	"assay":              "Lint AI instruction files against best practices",
	"plugin":             "Generate plugins from molds (Claude Code)",
	"ingots":             "Reusable template components",
	"agents-md":          "Tool-agnostic agent instructions in molds",
	"cast-claude-plugin": "Cast a mold as a Claude Code plugin",
	"helm-users":         "Concept map for Helm users coming to Ailloy",
	"cache":              "Clear ailloy's on-disk cache (mold artifacts and foundry indexes)",
}

// CommandTopic maps a cobra command name to the topic slug rendered when
// `--docs` is passed to that command. Slugs may include subdirectories.
var CommandTopic = map[string]string{
	"ailloy":  "getting-started",
	"anneal":  "anneal",
	"cast":    "blanks",
	"forge":   "blanks",
	"mold":    "blanks",
	"foundry": "foundry",
	"smelt":   "smelt",
	"temper":  "temper",
	"assay":   "assay",
	"plugin":  "plugin",
	"ingot":   "ingots",
	"cache":   "cache",
}

// FS exposes the embedded filesystem for advanced consumers (e.g. tests).
func FS() fs.FS { return docsFS }

// List returns every embedded topic in alphabetical-by-slug order, with
// "getting-started" pinned first and top-level docs ahead of subdirectory
// docs at the same name. Discovery is fully automatic — adding a new .md
// file (in this directory or any subdirectory) is enough to surface it.
func List() []Topic {
	var topics []Topic
	_ = fs.WalkDir(docsFS, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if !strings.HasSuffix(strings.ToLower(p), ".md") {
			return nil
		}
		base := strings.TrimSuffix(d.Name(), ".md")
		if strings.EqualFold(base, "README") {
			return nil
		}
		dir := path.Dir(p)
		if dir == "." {
			dir = ""
		}
		slug := strings.ToLower(strings.TrimSuffix(p, ".md"))
		title, summary := metadataFor(p, slug)
		topics = append(topics, Topic{
			Slug:    slug,
			Title:   title,
			Summary: summary,
			File:    p,
			Dir:     dir,
		})
		return nil
	})

	sort.Slice(topics, func(i, j int) bool {
		switch {
		case topics[i].Slug == "getting-started":
			return true
		case topics[j].Slug == "getting-started":
			return false
		}
		// Top-level docs first, then sorted by slug.
		iTop := topics[i].Dir == ""
		jTop := topics[j].Dir == ""
		if iTop != jTop {
			return iTop
		}
		return topics[i].Slug < topics[j].Slug
	})
	return topics
}

// Node is a tree node used by the docs TUI to render directories and files.
// A directory has Children and an empty Topic; a leaf has a populated Topic
// and no Children.
type Node struct {
	Name     string  // display name (last path segment)
	Path     string  // FS path: "" for root, "guides" for a folder, "guides/quickstart.md" for a leaf
	IsDir    bool    // true for directories, false for files
	Topic    Topic   // populated for leaves
	Children []*Node // populated for directories
}

// Tree builds a directory-aware tree of every embedded topic. Directory
// nodes are sorted before file nodes, then each group sorted by name (with
// "getting-started" pinned first at the root). Use this to render a
// collapsible tree-style nav.
func Tree() *Node {
	root := &Node{Name: "docs", IsDir: true}
	for _, t := range List() {
		insertTopic(root, t)
	}
	sortNode(root)
	return root
}

func insertTopic(root *Node, t Topic) {
	cur := root
	if t.Dir != "" {
		segments := strings.Split(t.Dir, "/")
		for i, seg := range segments {
			child := findChild(cur, seg, true)
			if child == nil {
				child = &Node{
					Name:  seg,
					Path:  strings.Join(segments[:i+1], "/"),
					IsDir: true,
				}
				cur.Children = append(cur.Children, child)
			}
			cur = child
		}
	}
	cur.Children = append(cur.Children, &Node{
		Name:  path.Base(t.File),
		Path:  t.File,
		IsDir: false,
		Topic: t,
	})
}

func findChild(parent *Node, name string, dir bool) *Node {
	for _, c := range parent.Children {
		if c.Name == name && c.IsDir == dir {
			return c
		}
	}
	return nil
}

func sortNode(n *Node) {
	sort.SliceStable(n.Children, func(i, j int) bool {
		ci, cj := n.Children[i], n.Children[j]
		// Directories first.
		if ci.IsDir != cj.IsDir {
			return ci.IsDir
		}
		// Pin getting-started at the top of the root level.
		if !ci.IsDir && !cj.IsDir {
			if ci.Topic.Slug == "getting-started" {
				return true
			}
			if cj.Topic.Slug == "getting-started" {
				return false
			}
		}
		return ci.Name < cj.Name
	})
	for _, c := range n.Children {
		if c.IsDir {
			sortNode(c)
		}
	}
}

// Find returns the Topic for the given slug, performing a case-insensitive
// match. Returns false if no topic matches.
func Find(slug string) (Topic, bool) {
	slug = strings.ToLower(strings.TrimSpace(slug))
	if slug == "" {
		return Topic{}, false
	}
	for _, t := range List() {
		if t.Slug == slug {
			return t, true
		}
	}
	return Topic{}, false
}

// Read returns the raw markdown bytes for a topic.
func Read(slug string) ([]byte, error) {
	t, ok := Find(slug)
	if !ok {
		return nil, fmt.Errorf("unknown docs topic %q (run `ailloy docs` to list topics)", slug)
	}
	return docsFS.ReadFile(t.File)
}

// metadataFor extracts the H1 title and short summary from an embedded
// file. The summary prefers a curated value from the summaries map (keyed
// by slug or by basename) and falls back to the file's first body paragraph
// so newly added docs ship with a usable description automatically.
func metadataFor(filename, slug string) (title, summary string) {
	data, err := docsFS.ReadFile(filename)
	if err != nil {
		return slug, lookupSummary(slug)
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if rest, ok := strings.CutPrefix(trimmed, "# "); ok {
			title = strings.TrimSpace(rest)
			break
		}
	}
	if title == "" {
		title = path.Base(slug)
	}
	if s := lookupSummary(slug); s != "" {
		return title, s
	}
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "<") {
			continue
		}
		return title, firstSentence(trimmed)
	}
	return title, ""
}

// lookupSummary checks the curated summaries map by full slug first, then
// by the basename so summaries written for old top-level slugs continue to
// work after a doc is moved into a subdirectory.
func lookupSummary(slug string) string {
	if s, ok := summaries[slug]; ok {
		return s
	}
	if base := path.Base(slug); base != slug {
		if s, ok := summaries[base]; ok {
			return s
		}
	}
	return ""
}

// firstSentence returns up to the first sentence-terminator in s, capped at
// a reasonable length so single-line summaries fit in the topics table.
func firstSentence(s string) string {
	const maxLen = 100
	for i, r := range s {
		if r == '.' || r == '\n' {
			return strings.TrimSpace(s[:i])
		}
		if i >= maxLen {
			return strings.TrimSpace(s[:maxLen]) + "…"
		}
	}
	return s
}
