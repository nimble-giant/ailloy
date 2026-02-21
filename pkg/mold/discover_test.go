package mold

import (
	"fmt"
	"testing"
)

func TestDiscoverExecutor_SimpleLineOutput(t *testing.T) {
	d := &DiscoverExecutor{
		RunCmd: func(cmd string) ([]byte, error) {
			return []byte("option1\noption2\noption3\n"), nil
		},
	}

	results, err := d.Run(DiscoverSpec{Command: "echo test"}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	for i, r := range results {
		expected := fmt.Sprintf("option%d", i+1)
		if r.Label != expected || r.Value != expected {
			t.Errorf("result[%d]: expected label=%q value=%q, got label=%q value=%q", i, expected, expected, r.Label, r.Value)
		}
	}
}

func TestDiscoverExecutor_PipeDelimitedOutput(t *testing.T) {
	d := &DiscoverExecutor{
		RunCmd: func(cmd string) ([]byte, error) {
			return []byte("Engineering (#5)|PVT_abc123\nDesign (#8)|PVT_def456\n"), nil
		},
	}

	results, err := d.Run(DiscoverSpec{Command: "echo test"}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].Label != "Engineering (#5)" || results[0].Value != "PVT_abc123" {
		t.Errorf("result[0]: got label=%q value=%q", results[0].Label, results[0].Value)
	}
	if results[1].Label != "Design (#8)" || results[1].Value != "PVT_def456" {
		t.Errorf("result[1]: got label=%q value=%q", results[1].Label, results[1].Value)
	}
}

func TestDiscoverExecutor_WithParseTemplate(t *testing.T) {
	jsonOutput := `{"items":[{"name":"Alpha","id":"1"},{"name":"Beta","id":"2"}]}`

	d := &DiscoverExecutor{
		RunCmd: func(cmd string) ([]byte, error) {
			return []byte(jsonOutput), nil
		},
	}

	spec := DiscoverSpec{
		Command: "echo test",
		Parse:   `{{- range .items -}}{{ .name }}|{{ .id }}{{ "\n" }}{{- end -}}`,
	}

	results, err := d.Run(spec, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].Label != "Alpha" || results[0].Value != "1" {
		t.Errorf("result[0]: got label=%q value=%q", results[0].Label, results[0].Value)
	}
	if results[1].Label != "Beta" || results[1].Value != "2" {
		t.Errorf("result[1]: got label=%q value=%q", results[1].Label, results[1].Value)
	}
}

func TestDiscoverExecutor_ExtraSegments(t *testing.T) {
	d := &DiscoverExecutor{
		RunCmd: func(cmd string) ([]byte, error) {
			return []byte("engineering (#6)|PVT_abc|engineering|6\ndesign (#8)|PVT_def|design|8\n"), nil
		},
	}

	results, err := d.Run(DiscoverSpec{Command: "echo test"}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].Label != "engineering (#6)" || results[0].Value != "PVT_abc" {
		t.Errorf("result[0]: got label=%q value=%q", results[0].Label, results[0].Value)
	}
	if len(results[0].Extra) != 2 || results[0].Extra[0] != "engineering" || results[0].Extra[1] != "6" {
		t.Errorf("result[0].Extra: expected [engineering 6], got %v", results[0].Extra)
	}
	if len(results[1].Extra) != 2 || results[1].Extra[0] != "design" || results[1].Extra[1] != "8" {
		t.Errorf("result[1].Extra: expected [design 8], got %v", results[1].Extra)
	}
}

func TestDiscoverExecutor_CommandFailure(t *testing.T) {
	d := &DiscoverExecutor{
		RunCmd: func(cmd string) ([]byte, error) {
			return nil, fmt.Errorf("command not found")
		},
	}

	results, err := d.Run(DiscoverSpec{Command: "bad-command"}, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if results != nil {
		t.Errorf("expected nil results, got %v", results)
	}
}

func TestDiscoverExecutor_EmptyOutput(t *testing.T) {
	d := &DiscoverExecutor{
		RunCmd: func(cmd string) ([]byte, error) {
			return []byte(""), nil
		},
	}

	results, err := d.Run(DiscoverSpec{Command: "echo"}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestDiscoverExecutor_TemplateExpansion(t *testing.T) {
	var capturedCmd string
	d := &DiscoverExecutor{
		RunCmd: func(cmd string) ([]byte, error) {
			capturedCmd = cmd
			return []byte("result\n"), nil
		},
	}

	flux := map[string]any{
		"project": map[string]any{
			"organization": "acme",
		},
	}

	spec := DiscoverSpec{
		Command: "gh api --org={{.project.organization}}",
	}

	_, err := d.Run(spec, flux)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedCmd != "gh api --org=acme" {
		t.Errorf("expected expanded command, got %q", capturedCmd)
	}
}

func TestDiscoverExecutor_ParseTemplateError(t *testing.T) {
	d := &DiscoverExecutor{
		RunCmd: func(cmd string) ([]byte, error) {
			return []byte(`{"key":"val"}`), nil
		},
	}

	spec := DiscoverSpec{
		Command: "echo test",
		Parse:   "{{.invalid_func_call | bad}}",
	}

	_, err := d.Run(spec, nil)
	if err == nil {
		t.Fatal("expected error for bad parse template")
	}
}

func TestDiscoverExecutor_SkipsBlankLines(t *testing.T) {
	d := &DiscoverExecutor{
		RunCmd: func(cmd string) ([]byte, error) {
			return []byte("opt1\n\n  \nopt2\n"), nil
		},
	}

	results, err := d.Run(DiscoverSpec{Command: "echo test"}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
}

func TestExpandTemplate_GraphQLCurlyBraces(t *testing.T) {
	// Ensure GraphQL queries with single curly braces don't confuse the template engine
	cmd := "gh api graphql -f query='\n  { viewer { organizations(first: 50) { nodes { login name } } } }\n'"
	result, err := expandTemplate(cmd, map[string]any{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != cmd {
		t.Errorf("template mangled GraphQL query:\ngot:  %q\nwant: %q", result, cmd)
	}
}

func TestExpandTemplate(t *testing.T) {
	flux := map[string]any{
		"org": "acme",
		"nested": map[string]any{
			"val": "deep",
		},
	}

	result, err := expandTemplate("hello {{.org}} and {{.nested.val}}", flux)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "hello acme and deep" {
		t.Errorf("expected 'hello acme and deep', got %q", result)
	}
}

func TestExpandTemplate_MissingKey(t *testing.T) {
	result, err := expandTemplate("hello {{.missing}}", map[string]any{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// missingkey=zero renders as zero value (empty string for interface)
	if result != "hello <no value>" {
		t.Errorf("expected missing key to render as zero value, got %q", result)
	}
}
