package mold

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"text/template"
)

// DiscoverResult represents a single discovered option from a discovery command.
// Extra holds additional pipe-delimited segments beyond label|value for also_sets.
type DiscoverResult struct {
	Label string
	Value string
	Extra []string // additional segments at indices 2, 3, ... for also_sets
}

// DiscoverExecutor runs discovery commands and parses their output.
type DiscoverExecutor struct {
	// RunCmd executes a shell command and returns its stdout.
	// Injectable for testing; defaults to real shell execution.
	RunCmd func(command string) ([]byte, error)
}

// NewDiscoverExecutor creates a DiscoverExecutor that uses the real shell.
func NewDiscoverExecutor() *DiscoverExecutor {
	return &DiscoverExecutor{
		RunCmd: func(command string) ([]byte, error) {
			cmd := exec.Command("sh", "-c", command) // #nosec G204 -- discovery commands are authored by mold creators (trusted)
			var stderr bytes.Buffer
			cmd.Stderr = &stderr
			out, err := cmd.Output()
			if err != nil && stderr.Len() > 0 {
				return out, fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
			}
			return out, err
		},
	}
}

// Run executes a DiscoverSpec and returns parsed options.
// The flux map is used to template-expand variables in the command string.
// On failure, returns nil results and the error (caller decides fallback).
func (d *DiscoverExecutor) Run(spec DiscoverSpec, flux map[string]any) ([]DiscoverResult, error) {
	// Template-expand the command against current flux values
	expandedCmd, err := expandTemplate(spec.Command, flux)
	if err != nil {
		return nil, fmt.Errorf("expanding discover command template: %w", err)
	}

	// Execute the command
	output, err := d.RunCmd(expandedCmd)
	if err != nil {
		return nil, fmt.Errorf("running discover command: %w", err)
	}

	// Parse the output
	if spec.Parse == "" {
		return parseLinePerOption(output), nil
	}
	return parseJSONWithTemplate(output, spec.Parse)
}

// expandTemplate applies Go template expansion to a string using flux values.
func expandTemplate(tmplStr string, flux map[string]any) (string, error) {
	tmpl, err := template.New("discover").Option("missingkey=zero").Parse(tmplStr)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, flux); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// parseLinePerOption treats each non-empty line as both label and value.
// Lines in "label|value|extra1|extra2|..." format are split accordingly.
// Extra segments beyond label|value are stored in DiscoverResult.Extra for also_sets.
func parseLinePerOption(output []byte) []DiscoverResult {
	var results []DiscoverResult
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, "|")
		switch len(parts) {
		case 1:
			results = append(results, DiscoverResult{Label: line, Value: line})
		default:
			r := DiscoverResult{
				Label: strings.TrimSpace(parts[0]),
				Value: strings.TrimSpace(parts[1]),
			}
			for _, p := range parts[2:] {
				r.Extra = append(r.Extra, strings.TrimSpace(p))
			}
			results = append(results, r)
		}
	}
	return results
}

// parseJSONWithTemplate decodes stdout as JSON, then applies a Go template
// that should produce lines in "label|value" or "value" format.
func parseJSONWithTemplate(output []byte, parseTmpl string) ([]DiscoverResult, error) {
	var data any
	if err := json.Unmarshal(output, &data); err != nil {
		return nil, fmt.Errorf("parsing discover output as JSON: %w", err)
	}

	tmpl, err := template.New("parse").Option("missingkey=zero").Parse(parseTmpl)
	if err != nil {
		return nil, fmt.Errorf("parsing discover.parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("executing discover.parse template: %w", err)
	}

	return parseLinePerOption(buf.Bytes()), nil
}
