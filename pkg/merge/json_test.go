package merge

import (
	"strings"
	"testing"
)

func TestLoadJSON_PreservesOrderAndNumbers(t *testing.T) {
	in := []byte(`{"b": 2, "a": 1, "nested": {"y": "Y", "x": "X"}, "items": [1, 2, 3]}`)
	got, err := loadJSON(in)
	if err != nil {
		t.Fatalf("loadJSON: %v", err)
	}
	if got.kind != kindMap {
		t.Fatalf("want kindMap, got %v", got.kind)
	}
	wantKeys := []string{"b", "a", "nested", "items"}
	if len(got.keys) != len(wantKeys) {
		t.Fatalf("keys: want %v, got %v", wantKeys, got.keys)
	}
	for i, k := range wantKeys {
		if got.keys[i] != k {
			t.Fatalf("key[%d]: want %q, got %q", i, k, got.keys[i])
		}
	}
	// Integer should round-trip as json.Number "1", not float64.
	if s := got.fields["a"].scalar; s == nil {
		t.Fatal("a scalar nil")
	}
	if got.fields["items"].kind != kindSeq {
		t.Fatalf("items: want kindSeq, got %v", got.fields["items"].kind)
	}
	if len(got.fields["items"].seq) != 3 {
		t.Fatalf("items length: want 3, got %d", len(got.fields["items"].seq))
	}
}

func TestDumpJSON_RoundTrip(t *testing.T) {
	in := []byte(`{"b":2,"a":1,"nested":{"y":"Y","x":"X"},"items":[1,2,3],"flag":true,"none":null}`)
	n, err := loadJSON(in)
	if err != nil {
		t.Fatal(err)
	}
	out, err := dumpJSON(n)
	if err != nil {
		t.Fatal(err)
	}
	// 2-space indented, key order preserved.
	want := `{
  "b": 2,
  "a": 1,
  "nested": {
    "y": "Y",
    "x": "X"
  },
  "items": [
    1,
    2,
    3
  ],
  "flag": true,
  "none": null
}
`
	if string(out) != want {
		t.Errorf("dumpJSON mismatch.\nwant:\n%s\ngot:\n%s", want, string(out))
	}
}

func TestDumpJSON_StringEscaping(t *testing.T) {
	n := mkMap("msg", mkScalar("he said \"hi\"\nok"))
	out, err := dumpJSON(n)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), `"msg": "he said \"hi\"\nok"`) {
		t.Errorf("escaping wrong, got: %s", string(out))
	}
}

func TestMCPMergeRoundTrip(t *testing.T) {
	base := []byte(`{"mcp":{"outline":{"url":"https://outline"}}}`)
	overlay := []byte(`{"mcp":{"replicated-docs":{"url":"https://docs"}}}`)
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
	if !strings.Contains(got, `"outline"`) {
		t.Errorf("outline missing: %s", got)
	}
	if !strings.Contains(got, `"replicated-docs"`) {
		t.Errorf("replicated-docs missing: %s", got)
	}
	// Order: outline before replicated-docs (base before overlay).
	if strings.Index(got, `"outline"`) > strings.Index(got, `"replicated-docs"`) {
		t.Errorf("expected outline before replicated-docs, got: %s", got)
	}
}
