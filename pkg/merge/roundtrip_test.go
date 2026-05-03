package merge

import (
	"strings"
	"testing"
)

// jsonFixedPoint asserts that loadJSON → dumpJSON is a fixed point: the second
// dump equals the first. This is the canonical-form property — any valid input
// converges to a unique canonical serialization in one round-trip.
func jsonFixedPoint(t *testing.T, name, input string) {
	t.Helper()
	t.Run(name, func(t *testing.T) {
		n1, err := loadJSON([]byte(input))
		if err != nil {
			t.Fatalf("load1: %v\ninput: %s", err, input)
		}
		out1, err := dumpJSON(n1)
		if err != nil {
			t.Fatalf("dump1: %v", err)
		}
		n2, err := loadJSON(out1)
		if err != nil {
			t.Fatalf("load2: %v\nout1: %s", err, out1)
		}
		out2, err := dumpJSON(n2)
		if err != nil {
			t.Fatalf("dump2: %v", err)
		}
		if string(out1) != string(out2) {
			t.Errorf("not a fixed point.\nout1:\n%s\nout2:\n%s", out1, out2)
		}
	})
}

func yamlFixedPoint(t *testing.T, name, input string) {
	t.Helper()
	t.Run(name, func(t *testing.T) {
		n1, err := loadYAML([]byte(input))
		if err != nil {
			t.Fatalf("load1: %v\ninput: %s", err, input)
		}
		out1, err := dumpYAML(n1)
		if err != nil {
			t.Fatalf("dump1: %v", err)
		}
		n2, err := loadYAML(out1)
		if err != nil {
			t.Fatalf("load2: %v\nout1: %s", err, out1)
		}
		out2, err := dumpYAML(n2)
		if err != nil {
			t.Fatalf("dump2: %v", err)
		}
		if string(out1) != string(out2) {
			t.Errorf("not a fixed point.\nout1:\n%s\nout2:\n%s", out1, out2)
		}
	})
}

func TestJSONFixedPoint(t *testing.T) {
	jsonFixedPoint(t, "empty_object", `{}`)
	jsonFixedPoint(t, "empty_array", `[]`)
	jsonFixedPoint(t, "scalar_string", `"hello"`)
	jsonFixedPoint(t, "scalar_number", `42`)
	jsonFixedPoint(t, "scalar_bool", `true`)
	jsonFixedPoint(t, "scalar_null", `null`)
	jsonFixedPoint(t, "nested_object", `{"a":{"b":{"c":1}}}`)
	jsonFixedPoint(t, "nested_array", `[[[1,2],[3,4]],[[5,6]]]`)
	jsonFixedPoint(t, "mixed_types", `{"s":"x","n":1,"b":true,"z":null,"a":[1,2],"o":{"k":"v"}}`)
	jsonFixedPoint(t, "negative_numbers", `{"n":-42,"f":-3.14}`)
	jsonFixedPoint(t, "large_integer", `{"big":9007199254740993}`) // > 2^53, requires json.Number to survive
	jsonFixedPoint(t, "scientific_notation", `{"e":1.5e10}`)
	jsonFixedPoint(t, "url_with_ampersand", `{"url":"https://x.com/?a=1&b=2"}`)
	jsonFixedPoint(t, "html_chars", `{"html":"<div class=\"x\">&amp;</div>"}`)
	jsonFixedPoint(t, "newline_in_string", `{"s":"line1\nline2"}`)
	jsonFixedPoint(t, "tab_in_string", `{"s":"a\tb"}`)
	jsonFixedPoint(t, "unicode_string", `{"emoji":"hello 🌍"}`)
	jsonFixedPoint(t, "deep_nesting", strings.Repeat(`{"x":`, 20)+`1`+strings.Repeat(`}`, 20))
	jsonFixedPoint(t, "mcp_realistic", `{"mcp":{"outline":{"command":"npx","args":["-y","outline-mcp"],"env":{"OUTLINE_API_TOKEN":"$OUTLINE_API_TOKEN"}},"replicated-docs":{"command":"docs-server","args":["--port","8080"]}}}`)
}

func TestYAMLFixedPoint(t *testing.T) {
	yamlFixedPoint(t, "single_key", "a: 1\n")
	yamlFixedPoint(t, "nested", "a:\n  b:\n    c: 1\n")
	yamlFixedPoint(t, "sequence", "items:\n  - a\n  - b\n  - c\n")
	yamlFixedPoint(t, "mixed", "name: foo\nport: 8080\nenabled: true\nargs:\n  - --verbose\n  - --debug\n")
	yamlFixedPoint(t, "empty_object_value", "empty: {}\n")
	yamlFixedPoint(t, "empty_seq_value", "empty: []\n")
	yamlFixedPoint(t, "string_with_special_chars", "url: https://x.com/?a=1\n")
	yamlFixedPoint(t, "mcp_realistic", `mcp:
  outline:
    command: npx
    args:
      - -y
      - outline-mcp
    env:
      OUTLINE_API_TOKEN: $OUTLINE_API_TOKEN
  replicated-docs:
    command: docs-server
    args:
      - --port
      - "8080"
`)
}

// Cross-property: merging a tree with itself (mergeNodes(n, n)) should be
// idempotent — the result must equal the original by nodeEqual.
func TestMergeNodes_IdempotentJSON(t *testing.T) {
	cases := []string{
		`{"a":1}`,
		`{"mcp":{"outline":{"url":"https://outline"}}}`,
		`{"items":[1,2,3]}`,
		`{"nested":{"a":{"b":{"c":1}}}}`,
	}
	for _, c := range cases {
		t.Run(c, func(t *testing.T) {
			n, err := loadJSON([]byte(c))
			if err != nil {
				t.Fatal(err)
			}
			merged := mergeNodes(n, n)
			if !nodeEqual(n, merged) {
				t.Errorf("merge(n, n) should equal n for %q", c)
			}
		})
	}
}

// mergeNodes(base, empty) and mergeNodes(empty, base) should both equal base.
func TestMergeNodes_IdentityElement(t *testing.T) {
	base, _ := loadJSON([]byte(`{"a":1,"b":[1,2]}`))
	empty := &node{kind: kindMap, fields: map[string]*node{}}

	left := mergeNodes(empty, base)
	right := mergeNodes(base, empty)
	if !nodeEqual(base, left) {
		t.Error("merge(empty, base) should equal base")
	}
	if !nodeEqual(base, right) {
		t.Error("merge(base, empty) should equal base")
	}
}

// Non-mutation: a merge must not change either input.
func TestMergeNodes_DoesNotMutateInputs(t *testing.T) {
	baseInput := `{"a":{"b":1},"items":[1,2]}`
	overlayInput := `{"a":{"c":2},"items":[2,3]}`

	base, _ := loadJSON([]byte(baseInput))
	overlay, _ := loadJSON([]byte(overlayInput))

	_ = mergeNodes(base, overlay)

	// After merge, dumping base/overlay should still produce the original output.
	baseAfter, _ := dumpJSON(base)
	overlayAfter, _ := dumpJSON(overlay)

	baseExpected, _ := loadJSON([]byte(baseInput))
	overlayExpected, _ := loadJSON([]byte(overlayInput))
	baseExpectedDump, _ := dumpJSON(baseExpected)
	overlayExpectedDump, _ := dumpJSON(overlayExpected)

	if string(baseAfter) != string(baseExpectedDump) {
		t.Errorf("base mutated by merge.\nbefore: %s\nafter:  %s", baseExpectedDump, baseAfter)
	}
	if string(overlayAfter) != string(overlayExpectedDump) {
		t.Errorf("overlay mutated by merge.\nbefore: %s\nafter:  %s", overlayExpectedDump, overlayAfter)
	}
}
