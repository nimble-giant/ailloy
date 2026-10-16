package assay

import (
	"reflect"
	"testing"
)

func TestExtractImportRefsSkipsFencedCode(t *testing.T) {
	content := "@AGENTS.md\n" +
		"```python\n" +
		"@pytest.fixture\n" +
		"def client(): ...\n" +
		"```\n" +
		"  @docs/guide.md\n" +
		"~~~~java\n" +
		"@Test\n" +
		"~~~\n" +
		"@Mock still inside: a shorter fence does not close\n" +
		"~~~~\n" +
		"````md\n" +
		"```\n" +
		"@inner.md\n" +
		"```\n" +
		"````\n" +
		"@after.md\n"

	got := extractImportRefs([]byte(content))
	want := []string{"AGENTS.md", "docs/guide.md", "after.md"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("extractImportRefs() = %v, want %v", got, want)
	}
}

func TestExtractImportRefsUnclosedFence(t *testing.T) {
	got := extractImportRefs([]byte("@a.md\n```\n@b.md\n"))
	if want := []string{"a.md"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("extractImportRefs() = %v, want %v", got, want)
	}
}

func TestFenceMarker(t *testing.T) {
	cases := map[string]string{
		"```":      "```",
		"```go":    "```",
		"~~~~":     "~~~~",
		"``":       "",
		"`code`":   "",
		"@x":       "",
		"":         "",
		"`````yml": "`````",
	}
	for in, want := range cases {
		if got := fenceMarker(in); got != want {
			t.Errorf("fenceMarker(%q) = %q, want %q", in, got, want)
		}
	}
}
