package assay

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/nimble-giant/ailloy/pkg/mold"
)

func init() {
	Register(&oreShadowingRule{})
}

// oreShadowingRule warns when a mold ships a packaged ore at ./ores/<name>/
// while also carrying hand-rolled `ore.<name>.*` entries in flux.schema.yaml.
// The mold-local entries silently win (per the mold-wins precedence rule),
// which almost always indicates the author forgot to clean up after migrating
// from the in-tree convention to the packaged-ore convention.
type oreShadowingRule struct{}

func (r *oreShadowingRule) Name() string                       { return "ore-shadowing" }
func (r *oreShadowingRule) DefaultSeverity() mold.DiagSeverity { return mold.SeverityWarning }
func (r *oreShadowingRule) Platforms() []Platform              { return nil }

func (r *oreShadowingRule) Check(ctx *RuleContext) []mold.Diagnostic {
	if ctx == nil || ctx.RootDir == "" {
		return nil
	}

	// Only applicable inside a mold tree.
	if _, err := os.Stat(filepath.Join(ctx.RootDir, "mold.yaml")); err != nil {
		return nil
	}

	oresDir := filepath.Join(ctx.RootDir, "ores")
	entries, err := os.ReadDir(oresDir)
	if err != nil {
		return nil
	}

	type oreInfo struct {
		installDir string
		namespace  string
	}
	var packaged []oreInfo
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		manifestPath := filepath.Join(oresDir, e.Name(), "ore.yaml")
		if _, err := os.Stat(manifestPath); err != nil {
			continue
		}
		ore, err := mold.LoadOre(manifestPath)
		if err != nil || ore == nil {
			continue
		}
		ns := e.Name()
		if e.Name() == ore.Name {
			ns = ore.EffectiveNamespace()
		}
		if ns == "" {
			continue
		}
		packaged = append(packaged, oreInfo{installDir: e.Name(), namespace: ns})
	}
	if len(packaged) == 0 {
		return nil
	}

	schema, err := mold.LoadFluxSchema(os.DirFS(ctx.RootDir), "flux.schema.yaml")
	if err != nil || len(schema) == 0 {
		return nil
	}

	sort.Slice(packaged, func(i, j int) bool {
		return packaged[i].installDir < packaged[j].installDir
	})

	var diags []mold.Diagnostic
	for _, oi := range packaged {
		prefix := "ore." + oi.namespace + "."
		hasMatch := false
		for _, fv := range schema {
			if strings.HasPrefix(fv.Name, prefix) {
				hasMatch = true
				break
			}
		}
		if !hasMatch {
			continue
		}
		diags = append(diags, mold.Diagnostic{
			Severity: r.DefaultSeverity(),
			Message: fmt.Sprintf(
				"flux.schema.yaml contains entries with prefix %q while a packaged ore exists at ./ores/%s/. Either remove the in-tree entries (they're shadowing the packaged ore) or remove the package.",
				prefix, oi.installDir,
			),
			File: "flux.schema.yaml",
			Rule: r.Name(),
		})
	}
	return diags
}
