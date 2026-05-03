// Package clidocs embeds the project's markdown documentation so it can be
// rendered inside the ailloy CLI via `ailloy docs` and the `--docs` flag.
//
// The same files in this directory serve double duty: they are browsed on
// GitHub as the project's documentation site, and at build time they are
// embedded into the binary for terminal rendering through glamour.
package clidocs

import (
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strings"
)

//go:embed *.md
var docsFS embed.FS

// Topic describes a single embedded documentation topic.
type Topic struct {
	// Slug is the lowercase identifier used by the user (e.g. "anneal").
	Slug string
	// Title is the document's H1 heading, used for listings.
	Title string
	// Summary is a short one-line description for the topic listing.
	Summary string
	// File is the source filename inside the embedded FS (e.g. "anneal.md").
	File string
}

// summaries provides curated one-line descriptions per topic. Falls back to
// the document's first non-heading paragraph when a slug isn't listed here.
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
}

// CommandTopic maps a cobra command name (or path) to the topic slug rendered
// when `--docs` is passed to that command. Multi-word command paths use a
// space as separator (e.g. "foundry add").
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
}

// FS exposes the embedded filesystem for advanced consumers (e.g. tests).
func FS() fs.FS { return docsFS }

// List returns the available topics sorted alphabetically by slug, with
// "getting-started" always first if present.
func List() []Topic {
	entries, err := docsFS.ReadDir(".")
	if err != nil {
		return nil
	}
	topics := make([]Topic, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		slug := strings.TrimSuffix(e.Name(), ".md")
		if strings.EqualFold(slug, "README") {
			continue
		}
		slug = strings.ToLower(slug)
		title, summary := metadataFor(e.Name(), slug)
		topics = append(topics, Topic{
			Slug:    slug,
			Title:   title,
			Summary: summary,
			File:    e.Name(),
		})
	}

	sort.Slice(topics, func(i, j int) bool {
		if topics[i].Slug == "getting-started" {
			return true
		}
		if topics[j].Slug == "getting-started" {
			return false
		}
		return topics[i].Slug < topics[j].Slug
	})
	return topics
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

// metadataFor extracts the H1 title and short summary from an embedded file.
// The summary prefers a curated value from the summaries map when one exists,
// falling back to the file's first body paragraph.
func metadataFor(filename, slug string) (title, summary string) {
	data, err := docsFS.ReadFile(filename)
	if err != nil {
		return slug, summaries[slug]
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
		title = slug
	}
	if s, ok := summaries[slug]; ok {
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
