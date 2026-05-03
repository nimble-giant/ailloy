package commands

import (
	"bytes"
	"strings"
	"testing"

	clidocs "github.com/nimble-giant/ailloy/docs"
	"github.com/spf13/cobra"
)

func TestDocsList_IncludesGettingStarted(t *testing.T) {
	topics := clidocs.List()
	if len(topics) == 0 {
		t.Fatal("expected at least one topic, got 0")
	}
	if topics[0].Slug != "getting-started" {
		t.Errorf("expected first topic to be getting-started, got %q", topics[0].Slug)
	}
	for _, want := range []string{"flux", "anneal", "foundry"} {
		if _, ok := clidocs.Find(want); !ok {
			t.Errorf("expected topic %q in embedded docs, missing", want)
		}
	}
}

func TestDocsRead_ReturnsContent(t *testing.T) {
	body, err := clidocs.Read("flux")
	if err != nil {
		t.Fatalf("Read(flux): %v", err)
	}
	if !bytes.Contains(body, []byte("Flux")) {
		t.Errorf("flux topic does not contain 'Flux'; first 80 bytes: %q", string(body[:min(80, len(body))]))
	}
}

func TestDocsRead_UnknownTopicErrors(t *testing.T) {
	_, err := clidocs.Read("not-a-real-topic")
	if err == nil {
		t.Fatal("expected error for unknown topic, got nil")
	}
	if !strings.Contains(err.Error(), "unknown docs topic") {
		t.Errorf("error message should mention 'unknown docs topic', got: %v", err)
	}
}

func TestPrintTopicList_RendersTable(t *testing.T) {
	var buf bytes.Buffer
	if err := printTopicList(&buf); err != nil {
		t.Fatalf("printTopicList: %v", err)
	}
	out := buf.String()
	for _, want := range []string{"Topic", "Description", "getting-started", "flux"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected output to contain %q; got:\n%s", want, out)
		}
	}
}

func TestRenderTopicTo_RendersMarkdown(t *testing.T) {
	var buf bytes.Buffer
	if err := renderTopicTo(&buf, "getting-started"); err != nil {
		t.Fatalf("renderTopicTo: %v", err)
	}
	out := buf.String()
	if out == "" {
		t.Fatal("expected non-empty rendered output")
	}
	if !strings.Contains(out, "Getting Started") {
		t.Errorf("expected rendered topic to mention 'Getting Started', got: %q", out)
	}
}

func TestRenderTopicTo_UnknownTopicErrors(t *testing.T) {
	var buf bytes.Buffer
	err := renderTopicTo(&buf, "definitely-not-a-topic")
	if err == nil {
		t.Fatal("expected error for unknown topic")
	}
}

func TestTopicForCommand_WalksParents(t *testing.T) {
	root := &cobra.Command{Use: "ailloy"}
	foundry := &cobra.Command{Use: "foundry"}
	add := &cobra.Command{Use: "add"}
	root.AddCommand(foundry)
	foundry.AddCommand(add)

	if got := topicForCommand(add); got != "foundry" {
		t.Errorf("expected `foundry add` to fall back to foundry topic, got %q", got)
	}
	if got := topicForCommand(foundry); got != "foundry" {
		t.Errorf("expected foundry topic, got %q", got)
	}
}

func TestTopicForCommand_UnknownReturnsEmpty(t *testing.T) {
	cmd := &cobra.Command{Use: "no-such-cmd"}
	if got := topicForCommand(cmd); got != "" {
		t.Errorf("expected empty topic for unmapped command, got %q", got)
	}
}

func TestCommandTopic_AllSlugsResolve(t *testing.T) {
	for cmdName, slug := range clidocs.CommandTopic {
		if _, ok := clidocs.Find(slug); !ok {
			t.Errorf("CommandTopic[%q]=%q does not resolve to an embedded topic", cmdName, slug)
		}
	}
}

func TestDocsRenderWidth_BoundedByCaps(t *testing.T) {
	// docsRenderWidth doesn't take args; ensure it returns a reasonable value
	// even when the caller is not attached to a TTY (test environment).
	w := docsRenderWidth()
	if w < minDocsWidth || w > maxDocsWidth {
		t.Errorf("docsRenderWidth() = %d; want between %d and %d", w, minDocsWidth, maxDocsWidth)
	}
}
