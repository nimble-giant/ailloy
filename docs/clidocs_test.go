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
