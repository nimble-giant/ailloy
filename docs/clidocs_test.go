package clidocs

import (
	"strings"
	"testing"
)

func TestList_NotEmpty(t *testing.T) {
	topics := List()
	if len(topics) == 0 {
		t.Fatal("expected at least one embedded topic")
	}
}

func TestList_GettingStartedFirst(t *testing.T) {
	topics := List()
	if topics[0].Slug != "getting-started" {
		t.Errorf("expected getting-started first, got %q", topics[0].Slug)
	}
}

func TestList_TopicsHaveTitleAndSummary(t *testing.T) {
	for _, topic := range List() {
		if topic.Title == "" {
			t.Errorf("topic %q has empty Title", topic.Slug)
		}
		if topic.Summary == "" {
			t.Errorf("topic %q has empty Summary", topic.Slug)
		}
	}
}

func TestFind_CaseInsensitive(t *testing.T) {
	if _, ok := Find("FLUX"); !ok {
		t.Errorf("Find should be case-insensitive")
	}
	if _, ok := Find("  flux  "); !ok {
		t.Errorf("Find should trim whitespace")
	}
}

func TestFind_Unknown(t *testing.T) {
	if _, ok := Find("nope-this-doesnt-exist"); ok {
		t.Errorf("Find should return false for unknown topic")
	}
	if _, ok := Find(""); ok {
		t.Errorf("Find should return false for empty slug")
	}
}

func TestRead_KnownTopic(t *testing.T) {
	body, err := Read("getting-started")
	if err != nil {
		t.Fatalf("Read getting-started: %v", err)
	}
	if !strings.Contains(string(body), "Getting Started") {
		t.Errorf("expected getting-started content; first 60 bytes: %q", string(body[:min(60, len(body))]))
	}
}

func TestList_ExcludesReadme(t *testing.T) {
	for _, topic := range List() {
		if strings.EqualFold(topic.Slug, "readme") {
			t.Errorf("README should not appear in the topic list")
		}
	}
}

func TestCommandTopic_PointsToValidTopics(t *testing.T) {
	for cmdName, slug := range CommandTopic {
		if _, ok := Find(slug); !ok {
			t.Errorf("CommandTopic[%q] = %q is not a known topic", cmdName, slug)
		}
	}
}

func TestList_DiscoversNestedTopics(t *testing.T) {
	// docs/topics/tutorials/first-mold.md ships in the embed; ensure the
	// recursive walk actually surfaces it so future nested docs are auto-
	// discovered without code changes.
	want := "topics/tutorials/first-mold"
	topic, ok := Find(want)
	if !ok {
		t.Fatalf("recursive embed should expose %q via Find", want)
	}
	if topic.Dir != "topics/tutorials" {
		t.Errorf("expected nested Dir 'topics/tutorials', got %q", topic.Dir)
	}
}

func TestTree_HasDirectoriesBeforeFiles(t *testing.T) {
	root := Tree()
	if root == nil || len(root.Children) == 0 {
		t.Fatal("expected a non-empty tree")
	}
	// Find first dir and first file indices.
	firstFile := -1
	firstDir := -1
	for i, c := range root.Children {
		if c.IsDir && firstDir == -1 {
			firstDir = i
		}
		if !c.IsDir && firstFile == -1 {
			firstFile = i
		}
	}
	if firstDir == -1 {
		t.Fatal("expected at least one directory under the tree root (topics/)")
	}
	if firstFile != -1 && firstDir > firstFile {
		t.Errorf("directories should sort before files; firstDir=%d firstFile=%d", firstDir, firstFile)
	}
}

func TestTree_NestedSlugReachable(t *testing.T) {
	root := Tree()
	var walk func(n *Node) bool
	walk = func(n *Node) bool {
		if !n.IsDir && n.Topic.Slug == "topics/tutorials/first-mold" {
			return true
		}
		for _, c := range n.Children {
			if walk(c) {
				return true
			}
		}
		return false
	}
	if !walk(root) {
		t.Fatal("nested tutorial topic should be reachable through Tree()")
	}
}
