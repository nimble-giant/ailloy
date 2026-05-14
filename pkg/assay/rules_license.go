package assay

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/nimble-giant/ailloy/pkg/mold"
)

func init() {
	Register(&licenseMissingRule{})
}

// licenseMissingRule reports a suggestion when a mold/ingot/ore package at
// the assay root lacks a `license` field on its manifest. Low-severity,
// informational — declaring a license is best practice for distributed
// packages but never blocks linting.
//
// The rule fires for the manifest types ailloy publishes (mold.yaml,
// ingot.yaml, ore.yaml) at the root, plus each package in a multi-ingot
// layout under ingots/. Manifests with a non-empty `license` are silent.
type licenseMissingRule struct{}

func (r *licenseMissingRule) Name() string                       { return "license-missing" }
func (r *licenseMissingRule) DefaultSeverity() mold.DiagSeverity { return mold.SeveritySuggestion }
func (r *licenseMissingRule) Platforms() []Platform              { return nil }

func (r *licenseMissingRule) Check(ctx *RuleContext) []mold.Diagnostic {
	if ctx == nil || ctx.RootDir == "" {
		return nil
	}

	var diags []mold.Diagnostic

	if path := filepath.Join(ctx.RootDir, "mold.yaml"); fileExistsAt(path) {
		if m, err := mold.LoadMold(path); err == nil && m.License == "" {
			diags = append(diags, r.diag("mold.yaml"))
		}
	}
	if path := filepath.Join(ctx.RootDir, "ore.yaml"); fileExistsAt(path) {
		if o, err := mold.LoadOre(path); err == nil && o.License == "" {
			diags = append(diags, r.diag("ore.yaml"))
		}
	}
	if path := filepath.Join(ctx.RootDir, "ingot.yaml"); fileExistsAt(path) {
		if i, err := mold.LoadIngot(path); err == nil && i.License == "" {
			diags = append(diags, r.diag("ingot.yaml"))
		}
	}

	// Multi-ingot layout: ingots/<name>/ingot.yaml.
	ingotsDir := filepath.Join(ctx.RootDir, "ingots")
	if entries, err := os.ReadDir(ingotsDir); err == nil {
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			manifestPath := filepath.Join(ingotsDir, e.Name(), "ingot.yaml")
			if !fileExistsAt(manifestPath) {
				continue
			}
			i, err := mold.LoadIngot(manifestPath)
			if err != nil || i == nil {
				continue
			}
			if i.License == "" {
				diags = append(diags, r.diag(filepath.Join("ingots", e.Name(), "ingot.yaml")))
			}
		}
	}

	return diags
}

func (r *licenseMissingRule) diag(file string) mold.Diagnostic {
	return mold.Diagnostic{
		Severity: r.DefaultSeverity(),
		Message:  fmt.Sprintf("%s has no `license` field; consider declaring an SPDX identifier so consumers know how the package may be used", file),
		Tip:      "see https://spdx.org/licenses/ for the SPDX list; common choices include Apache-2.0, MIT, BSD-3-Clause",
		File:     file,
		Rule:     r.Name(),
	}
}

func fileExistsAt(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
