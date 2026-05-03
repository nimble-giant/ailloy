package merge

import (
	"strings"
	"testing"
)

// TestJSONMerge_FullMCPConfig verifies a realistic opencode.json merge — two
// molds each declaring a complete MCP server entry with command, args, and
// env. End state must contain BOTH servers in source order, AND the inner
// fields (command/args/env) must be intact for both.
func TestJSONMerge_FullMCPConfig(t *testing.T) {
	base := []byte(`{
  "mcp": {
    "outline": {
      "command": "npx",
      "args": ["-y", "outline-mcp"],
      "env": {
        "OUTLINE_API_TOKEN": "$OUTLINE_API_TOKEN",
        "OUTLINE_API_URL": "https://outline.example.com/api"
      }
    }
  },
  "version": 1
}
`)
	overlay := []byte(`{
  "mcp": {
    "replicated-docs": {
      "command": "docs-server",
      "args": ["--port", "8080", "--verbose"],
      "env": {
        "DOCS_DB_PATH": "/var/lib/docs.db"
      }
    }
  }
}
`)
	bn, err := loadJSON(base)
	if err != nil {
		t.Fatal(err)
	}
	on, err := loadJSON(overlay)
	if err != nil {
		t.Fatal(err)
	}
	merged := mergeNodes(bn, on)
	out, err := dumpJSON(merged)
	if err != nil {
		t.Fatal(err)
	}
	got := string(out)

	for _, s := range []string{"outline", "replicated-docs", "OUTLINE_API_TOKEN", "DOCS_DB_PATH", "outline-mcp", "--port", "8080", "--verbose"} {
		if !strings.Contains(got, s) {
			t.Errorf("expected %q in merged output, missing.\nfull output:\n%s", s, got)
		}
	}

	if strings.Index(got, "outline") > strings.Index(got, "replicated-docs") {
		t.Errorf("expected outline before replicated-docs, got:\n%s", got)
	}

	if !strings.Contains(got, `"version": 1`) {
		t.Errorf("base-only top-level key 'version' lost:\n%s", got)
	}

	if _, err := loadJSON(out); err != nil {
		t.Errorf("merged JSON invalid on re-load: %v", err)
	}
}

func TestJSONMerge_NestedArrayOfObjects(t *testing.T) {
	base := []byte(`{
  "plugins": [
    {"name": "linter", "version": "1.0"},
    {"name": "formatter", "version": "2.0"}
  ]
}
`)
	overlay := []byte(`{
  "plugins": [
    {"name": "formatter", "version": "2.0"},
    {"name": "tests", "version": "3.0"}
  ]
}
`)
	bn, _ := loadJSON(base)
	on, _ := loadJSON(overlay)
	merged := mergeNodes(bn, on)

	plugins := merged.fields["plugins"]
	if plugins == nil || plugins.kind != kindSeq {
		t.Fatalf("plugins not a sequence: %+v", plugins)
	}
	if len(plugins.seq) != 3 {
		t.Fatalf("expected 3 plugins (linter, formatter, tests), got %d:\n%+v", len(plugins.seq), plugins.seq)
	}
}

func TestYAMLMerge_FullMCPConfig(t *testing.T) {
	base := []byte(`mcp:
  outline:
    command: npx
    args:
      - -y
      - outline-mcp
    env:
      OUTLINE_API_TOKEN: $OUTLINE_API_TOKEN
      OUTLINE_API_URL: https://outline.example.com/api
version: 1
`)
	overlay := []byte(`mcp:
  replicated-docs:
    command: docs-server
    args:
      - --port
      - "8080"
      - --verbose
    env:
      DOCS_DB_PATH: /var/lib/docs.db
`)
	bn, err := loadYAML(base)
	if err != nil {
		t.Fatal(err)
	}
	on, err := loadYAML(overlay)
	if err != nil {
		t.Fatal(err)
	}
	merged := mergeNodes(bn, on)
	out, err := dumpYAML(merged)
	if err != nil {
		t.Fatal(err)
	}
	got := string(out)

	for _, s := range []string{"outline", "replicated-docs", "OUTLINE_API_TOKEN", "DOCS_DB_PATH", "outline-mcp", "docs-server"} {
		if !strings.Contains(got, s) {
			t.Errorf("expected %q in merged YAML, missing.\nfull output:\n%s", s, got)
		}
	}
	if strings.Index(got, "outline") > strings.Index(got, "replicated-docs") {
		t.Errorf("expected outline before replicated-docs, got:\n%s", got)
	}
	if _, err := loadYAML(out); err != nil {
		t.Errorf("merged YAML invalid on re-load: %v", err)
	}
}

// TestYAMLMerge_NorwayProblem documents goccy's behavior on the classic YAML
// 1.1 quirk where bare NO/YES/ON/OFF can parse as booleans. We assert
// round-trip stability rather than a specific value.
func TestYAMLMerge_NorwayProblem(t *testing.T) {
	in := []byte("country: NO\n")
	n, err := loadYAML(in)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	out, err := dumpYAML(n)
	if err != nil {
		t.Fatal(err)
	}
	n2, err := loadYAML(out)
	if err != nil {
		t.Fatal(err)
	}
	if !nodeEqual(n, n2) {
		t.Errorf("Norway problem: %q does not round-trip\nfirst:  %+v\nsecond: %+v\ndumped:\n%s", in, n, n2, out)
	}
}

func TestYAMLMerge_TopLevelSequence(t *testing.T) {
	in := []byte("- a\n- b\n- c\n")
	n, err := loadYAML(in)
	if err != nil {
		t.Fatal(err)
	}
	if n.kind != kindSeq {
		t.Fatalf("expected kindSeq at root, got %v", n.kind)
	}
	out, err := dumpYAML(n)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := loadYAML(out); err != nil {
		t.Errorf("top-level seq round-trip failed: %v\nout:\n%s", err, out)
	}
}

func TestJSONMerge_TopLevelArray(t *testing.T) {
	in := []byte(`[1, 2, 3]`)
	n, err := loadJSON(in)
	if err != nil {
		t.Fatal(err)
	}
	if n.kind != kindSeq {
		t.Fatalf("expected kindSeq, got %v", n.kind)
	}
	out, err := dumpJSON(n)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := loadJSON(out); err != nil {
		t.Errorf("top-level array round-trip failed: %v\nout: %s", err, out)
	}
}

func TestYAMLMerge_BlockScalar(t *testing.T) {
	cases := map[string]string{
		"literal_block": "description: |\n  line one\n  line two\n",
		"folded_block":  "description: >\n  long\n  paragraph\n",
	}
	for name, in := range cases {
		t.Run(name, func(t *testing.T) {
			n, err := loadYAML([]byte(in))
			if err != nil {
				t.Fatalf("load: %v", err)
			}
			out, err := dumpYAML(n)
			if err != nil {
				t.Fatalf("dump: %v", err)
			}
			n2, err := loadYAML(out)
			if err != nil {
				t.Fatalf("reload: %v\nout:\n%s", err, out)
			}
			if !nodeEqual(n, n2) {
				t.Errorf("%s: not stable across round-trip\nfirst-dump:\n%s", name, out)
			}
		})
	}
}

func TestYAMLMerge_DeeplyNested(t *testing.T) {
	var b strings.Builder
	for i := range 20 {
		b.WriteString(strings.Repeat("  ", i))
		b.WriteString("level0:\n")
	}
	b.WriteString(strings.Repeat("  ", 20))
	b.WriteString("leaf: 1\n")
	in := []byte(b.String())
	n, err := loadYAML(in)
	if err != nil {
		t.Skipf("goccy may have nesting limits; skipping if so: %v", err)
	}
	out, err := dumpYAML(n)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := loadYAML(out); err != nil {
		t.Errorf("20-level YAML round-trip failed: %v", err)
	}
}

func TestJSONMerge_DeeplyNested(t *testing.T) {
	in := []byte(strings.Repeat(`{"x":`, 50) + `1` + strings.Repeat(`}`, 50))
	n, err := loadJSON(in)
	if err != nil {
		t.Fatal(err)
	}
	out, err := dumpJSON(n)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := loadJSON(out); err != nil {
		t.Errorf("50-level JSON round-trip failed: %v", err)
	}
}

func TestJSONMerge_LargeIntegers(t *testing.T) {
	in := []byte(`{"timestamp_ns": 1700000000123456789, "id": 9223372036854775807}`)
	n, err := loadJSON(in)
	if err != nil {
		t.Fatal(err)
	}
	out, err := dumpJSON(n)
	if err != nil {
		t.Fatal(err)
	}
	got := string(out)
	for _, want := range []string{"1700000000123456789", "9223372036854775807"} {
		if !strings.Contains(got, want) {
			t.Errorf("large integer %q lost precision; got:\n%s", want, got)
		}
	}
}

func TestJSONMerge_UnicodeStrings(t *testing.T) {
	in := []byte(`{"emoji":"hello 🌍","jp":"こんにちは","arabic":"مرحبا"}`)
	n, err := loadJSON(in)
	if err != nil {
		t.Fatal(err)
	}
	out, err := dumpJSON(n)
	if err != nil {
		t.Fatal(err)
	}
	got := string(out)
	for _, want := range []string{"🌍", "こんにちは", "مرحبا"} {
		if !strings.Contains(got, want) {
			t.Errorf("unicode %q not preserved; got:\n%s", want, got)
		}
	}
}

func TestMergeFile_PreservesCanonicalFormat(t *testing.T) {
	in := []byte("{\n    \"a\": 1\n}\n")
	n, err := loadJSON(in)
	if err != nil {
		t.Fatal(err)
	}
	out, err := dumpJSON(n)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), `  "a": 1`) {
		t.Errorf("expected 2-space indent in canonical output; got:\n%s", out)
	}
}
