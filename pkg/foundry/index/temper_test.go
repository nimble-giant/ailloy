package index

import (
	"fmt"
	"strings"
	"testing"
)

func TestTemperBytes_OfflineSchemaPasses(t *testing.T) {
	yaml := []byte(`apiVersion: v1
kind: foundry-index
name: ok
molds:
  - name: alpha
    source: github.com/x/alpha
`)
	res, err := TemperBytes(yaml, TemperOptions{Offline: true})
	if err != nil {
		t.Fatalf("TemperBytes: %v", err)
	}
	if res.HasErrors() {
		t.Fatalf("unexpected findings: %+v", res.Findings)
	}
	if res.Index.Name != "ok" {
		t.Errorf("Index.Name = %q, want ok", res.Index.Name)
	}
}

func TestTemperBytes_OfflineSchemaFails(t *testing.T) {
	// Missing apiVersion + missing mold source.
	yaml := []byte(`kind: foundry-index
name: bad
molds:
  - name: m1
`)
	res, err := TemperBytes(yaml, TemperOptions{Offline: true})
	if err != nil {
		t.Fatalf("TemperBytes: %v", err)
	}
	if !res.HasErrors() {
		t.Fatal("expected schema errors, got none")
	}
	if got := res.Findings[0].Err.Error(); !strings.Contains(got, "apiVersion") {
		t.Errorf("first finding %q should mention apiVersion", got)
	}
}

func TestTemperBytes_VerifiesMoldSources(t *testing.T) {
	yaml := []byte(`apiVersion: v1
kind: foundry-index
name: ok
molds:
  - name: good
    source: github.com/x/good
  - name: bad
    source: github.com/x/bad
`)
	called := map[string]int{}
	verifier := func(source string) error {
		called[source]++
		if source == "github.com/x/bad" {
			return fmt.Errorf("not found")
		}
		return nil
	}
	res, err := TemperBytes(yaml, TemperOptions{
		MoldVerifier:   verifier,
		FoundryFetcher: func(string) (*Index, error) { t.Fatal("nested fetch not expected"); return nil, nil },
	})
	if err != nil {
		t.Fatalf("TemperBytes: %v", err)
	}
	if called["github.com/x/good"] != 1 || called["github.com/x/bad"] != 1 {
		t.Errorf("verifier call counts = %v, want each once", called)
	}
	if !res.HasErrors() {
		t.Fatalf("expected errors, got none")
	}
	if len(res.Findings) != 1 {
		t.Fatalf("got %d findings, want 1", len(res.Findings))
	}
	if res.Findings[0].Source != "github.com/x/bad" {
		t.Errorf("finding.Source = %q, want github.com/x/bad", res.Findings[0].Source)
	}
}

func TestTemperBytes_RecursesIntoNestedFoundries(t *testing.T) {
	yaml := []byte(`apiVersion: v1
kind: foundry-index
name: parent
molds:
  - name: parent-mold
    source: github.com/x/parent-mold
foundries:
  - name: child
    source: github.com/x/child
`)
	moldsCalled := []string{}
	verifier := func(source string) error {
		moldsCalled = append(moldsCalled, source)
		return nil
	}
	fetcher := func(source string) (*Index, error) {
		if source != "github.com/x/child" {
			return nil, fmt.Errorf("unexpected: %s", source)
		}
		return &Index{
			APIVersion: "v1", Kind: "foundry-index", Name: "child",
			Molds: []MoldEntry{{Name: "child-mold", Source: "github.com/x/child-mold"}},
		}, nil
	}
	res, err := TemperBytes(yaml, TemperOptions{
		MoldVerifier:   verifier,
		FoundryFetcher: fetcher,
	})
	if err != nil {
		t.Fatalf("TemperBytes: %v", err)
	}
	if res.HasErrors() {
		t.Fatalf("unexpected findings: %+v", res.Findings)
	}
	want := []string{"github.com/x/parent-mold", "github.com/x/child-mold"}
	if len(moldsCalled) != len(want) {
		t.Fatalf("verifier called for %v, want %v", moldsCalled, want)
	}
	for i, w := range want {
		if moldsCalled[i] != w {
			t.Errorf("call[%d] = %q, want %q", i, moldsCalled[i], w)
		}
	}
}

func TestTemperBytes_NoRecurseSkipsChildren(t *testing.T) {
	yaml := []byte(`apiVersion: v1
kind: foundry-index
name: parent
molds: []
foundries:
  - name: child
    source: github.com/x/child
`)
	fetcher := func(string) (*Index, error) {
		t.Fatal("FoundryFetcher should not be invoked when NoRecurse is true")
		return nil, nil
	}
	res, err := TemperBytes(yaml, TemperOptions{
		NoRecurse:      true,
		MoldVerifier:   func(string) error { return nil },
		FoundryFetcher: fetcher,
	})
	if err != nil {
		t.Fatalf("TemperBytes: %v", err)
	}
	if res.HasErrors() {
		t.Fatalf("unexpected findings: %+v", res.Findings)
	}
}

func TestTemperBytes_NestedFetchFailureIsReported(t *testing.T) {
	yaml := []byte(`apiVersion: v1
kind: foundry-index
name: parent
molds: []
foundries:
  - name: broken
    source: github.com/x/broken
`)
	res, err := TemperBytes(yaml, TemperOptions{
		MoldVerifier:   func(string) error { return nil },
		FoundryFetcher: func(string) (*Index, error) { return nil, fmt.Errorf("network down") },
	})
	if err != nil {
		t.Fatalf("TemperBytes: %v", err)
	}
	if !res.HasErrors() {
		t.Fatal("expected an error finding for broken nested foundry")
	}
	if !strings.Contains(res.Findings[0].Err.Error(), "network down") {
		t.Errorf("finding.Err = %v, want it to mention network down", res.Findings[0].Err)
	}
}

func TestTemperBytes_Cycles(t *testing.T) {
	yaml := []byte(`apiVersion: v1
kind: foundry-index
name: a
molds: []
foundries:
  - name: b
    source: github.com/x/b
`)
	calls := 0
	fetcher := func(source string) (*Index, error) {
		calls++
		switch canonicalizeSource(source) {
		case "github.com/x/b":
			return &Index{
				APIVersion: "v1", Kind: "foundry-index", Name: "b",
				Foundries: []FoundryRef{{Name: "a", Source: "github.com/x/a"}},
			}, nil
		case "github.com/x/a":
			return &Index{
				APIVersion: "v1", Kind: "foundry-index", Name: "a",
				Foundries: []FoundryRef{{Name: "b", Source: "github.com/x/b"}},
			}, nil
		}
		return nil, fmt.Errorf("unexpected: %s", source)
	}
	res, err := TemperBytes(yaml, TemperOptions{
		MoldVerifier:   func(string) error { return nil },
		FoundryFetcher: fetcher,
	})
	if err != nil {
		t.Fatalf("TemperBytes: %v", err)
	}
	if res.HasErrors() {
		t.Fatalf("cycle should not produce errors: %+v", res.Findings)
	}
	if calls != 2 {
		t.Errorf("fetcher called %d times, want 2 (each child fetched once)", calls)
	}
}
