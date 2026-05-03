package merge

import (
	"strings"
	"testing"
)

func FuzzJSONRoundTrip(f *testing.F) {
	// Seed corpus.
	seeds := []string{
		`{}`,
		`[]`,
		`{"a":1}`,
		`{"mcp":{"outline":{"url":"https://outline"}}}`,
		`{"a":{"b":{"c":[1,2,3]}}}`,
		`{"url":"https://x.com/?a=1&b=2"}`,
		`null`,
		`42`,
		`"hello"`,
		`true`,
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, raw string) {
		// Skip absurdly large inputs to keep fuzzing fast.
		if len(raw) > 64*1024 {
			t.Skip()
		}
		n1, err := loadJSON([]byte(raw))
		if err != nil {
			// Loader rejected; that's fine.
			return
		}
		out1, err := dumpJSON(n1)
		if err != nil {
			t.Fatalf("loadJSON accepted but dumpJSON failed: %v\ninput: %q", err, raw)
		}
		n2, err := loadJSON(out1)
		if err != nil {
			t.Fatalf("dumpJSON output not parseable: %v\nout1: %s", err, out1)
		}
		out2, err := dumpJSON(n2)
		if err != nil {
			t.Fatalf("second dump failed: %v", err)
		}
		if string(out1) != string(out2) {
			t.Fatalf("not a fixed point.\ninput: %q\nout1:\n%s\nout2:\n%s", raw, out1, out2)
		}
	})
}

func FuzzYAMLRoundTrip(f *testing.F) {
	seeds := []string{
		"a: 1\n",
		"a:\n  b: 2\n",
		"items:\n  - a\n  - b\n",
		"port: 8080\nname: foo\n",
		"empty: {}\n",
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, raw string) {
		if len(raw) > 64*1024 {
			t.Skip()
		}
		// Skip control chars that would cause goccy/go-yaml lexer issues
		// unrelated to merge logic.
		if strings.ContainsAny(raw, "\x00") {
			t.Skip()
		}
		n1, err := loadYAML([]byte(raw))
		if err != nil {
			return
		}
		out1, err := dumpYAML(n1)
		if err != nil {
			t.Fatalf("loadYAML accepted but dumpYAML failed: %v\ninput: %q", err, raw)
		}
		n2, err := loadYAML(out1)
		if err != nil {
			t.Fatalf("dumpYAML output not parseable: %v\nout1: %s", err, out1)
		}
		out2, err := dumpYAML(n2)
		if err != nil {
			t.Fatalf("second dump failed: %v", err)
		}
		if string(out1) != string(out2) {
			t.Fatalf("not a fixed point.\ninput: %q\nout1:\n%s\nout2:\n%s", raw, out1, out2)
		}
	})
}

// FuzzMergeNodes asserts that merging two parsed trees never crashes and the
// result is dumpable.
func FuzzMergeNodesJSON(f *testing.F) {
	f.Add(`{"a":1}`, `{"b":2}`)
	f.Add(`{"items":[1,2]}`, `{"items":[2,3]}`)
	f.Add(`{}`, `{}`)
	f.Fuzz(func(t *testing.T, baseRaw, overlayRaw string) {
		if len(baseRaw)+len(overlayRaw) > 64*1024 {
			t.Skip()
		}
		base, err := loadJSON([]byte(baseRaw))
		if err != nil {
			return
		}
		overlay, err := loadJSON([]byte(overlayRaw))
		if err != nil {
			return
		}
		merged := mergeNodes(base, overlay)
		if _, err := dumpJSON(merged); err != nil {
			t.Fatalf("dumpJSON of merged tree failed: %v", err)
		}
	})
}
