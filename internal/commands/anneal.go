package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/goccy/go-yaml"
	"github.com/nimble-giant/ailloy/pkg/blanks"
	"github.com/nimble-giant/ailloy/pkg/mold"
	"github.com/nimble-giant/ailloy/pkg/styles"
	"github.com/spf13/cobra"
)

var annealCmd = &cobra.Command{
	Use:     "anneal [mold-dir]",
	Aliases: []string{"configure"},
	Short:   "Anneal blank flux variables",
	Long: `Anneal team-specific flux values for blanks (alias: configure).

This command reads a mold's flux.schema.yaml to discover which variables
need configuration, then runs an interactive wizard with type-driven prompts
and optional discovery commands. The result is written as a YAML file that
can be passed to cast or forge via the -f flag.

Example:
  ailloy anneal ./nimble-mold -o ore.yaml
  ailloy cast ./nimble-mold -f ore.yaml`,
	Args: cobra.MaximumNArgs(1),
	RunE: runAnneal,
}

var (
	annealSetVars []string
	annealOutput  string
)

func init() {
	rootCmd.AddCommand(annealCmd)

	annealCmd.Flags().StringArrayVarP(&annealSetVars, "set", "s", nil, "set flux variable (format: key=value)")
	annealCmd.Flags().StringVarP(&annealOutput, "output", "o", "", "write flux YAML to file (default: mold's flux.yaml)")
}

func runAnneal(_ *cobra.Command, args []string) error {
	// Scripted mode: --set flags (backward compatible, no mold required)
	if len(annealSetVars) > 0 {
		flux := make(map[string]any)
		if err := mold.ApplySetOverrides(flux, annealSetVars); err != nil {
			return err
		}
		if annealOutput != "" {
			return writeFluxToFile(flux, annealOutput)
		}
		return writeFluxToStdout(flux)
	}

	// Resolve mold directory
	moldDir := "."
	if len(args) >= 1 {
		moldDir = args[0]
	}

	reader, err := blanks.NewMoldReaderFromPath(moldDir)
	if err != nil {
		return fmt.Errorf("reading mold directory: %w", err)
	}

	// Resolve schema and flux defaults
	schema, fluxDefaults, err := resolveAnnealSchema(reader)
	if err != nil {
		return err
	}

	if len(schema) == 0 {
		return fmt.Errorf("no flux variables found in %s (add flux.schema.yaml or flux.yaml)", moldDir)
	}

	// Interactive mode: run dynamic wizard
	wiz := newDynamicWizard(schema, fluxDefaults)
	result, confirmed, err := wiz.run()
	if err != nil {
		return err
	}

	if !confirmed {
		// User chose "Cancel" — print to stdout for inspection
		if result != nil {
			return writeFluxToStdout(result)
		}
		return nil
	}

	// User chose "Save" — write to file
	dest := annealOutput
	if dest == "" {
		dest = filepath.Join(moldDir, "flux.yaml")
	}
	if err := writeFluxToFile(result, dest); err != nil {
		return err
	}

	fmt.Println()
	fmt.Println(styles.SuccessBanner("Blank annealing saved to " + dest))
	return nil
}

// resolveAnnealSchema resolves the schema and flux defaults from a mold.
// Precedence: flux.schema.yaml > mold.yaml flux declarations > inferred from flux.yaml keys.
func resolveAnnealSchema(reader *blanks.MoldReader) ([]mold.FluxVar, map[string]any, error) {
	// Load flux defaults
	fluxDefaults, err := reader.LoadFluxDefaults()
	if err != nil {
		fluxDefaults = make(map[string]any)
	}

	// Try flux.schema.yaml first
	schema, err := reader.LoadFluxSchema()
	if err != nil {
		return nil, nil, fmt.Errorf("loading flux schema: %w", err)
	}
	if schema != nil {
		return schema, fluxDefaults, nil
	}

	// Fall back to mold.yaml flux declarations
	manifest, err := reader.LoadManifest()
	if err == nil && manifest != nil && len(manifest.Flux) > 0 {
		return manifest.Flux, fluxDefaults, nil
	}

	// Fall back: infer schema from flux.yaml keys
	if len(fluxDefaults) > 0 {
		schema = inferSchemaFromFlux(fluxDefaults)
		return schema, fluxDefaults, nil
	}

	return nil, fluxDefaults, nil
}

// inferSchemaFromFlux walks a nested flux map and creates FluxVar entries
// with types inferred from Go values. This provides a basic wizard for molds
// that have flux.yaml but no schema.
func inferSchemaFromFlux(flux map[string]any) []mold.FluxVar {
	var schema []mold.FluxVar
	inferFluxVars(flux, "", &schema)

	// Sort by name for consistent ordering
	sort.Slice(schema, func(i, j int) bool {
		return schema[i].Name < schema[j].Name
	})

	return schema
}

// inferFluxVars recursively walks a nested map and appends FluxVar entries.
func inferFluxVars(m map[string]any, prefix string, schema *[]mold.FluxVar) {
	for key, val := range m {
		name := key
		if prefix != "" {
			name = prefix + "." + key
		}

		switch v := val.(type) {
		case map[string]any:
			inferFluxVars(v, name, schema)
		case bool:
			*schema = append(*schema, mold.FluxVar{
				Name: name,
				Type: "bool",
			})
		case int, int64, float64:
			*schema = append(*schema, mold.FluxVar{
				Name: name,
				Type: "int",
			})
		default:
			*schema = append(*schema, mold.FluxVar{
				Name: name,
				Type: "string",
			})
		}
	}
}

// writeFluxToStdout marshals flux to YAML and prints to stdout.
func writeFluxToStdout(flux map[string]any) error {
	data, err := yaml.Marshal(flux)
	if err != nil {
		return fmt.Errorf("failed to marshal flux: %w", err)
	}
	fmt.Print(string(data))
	return nil
}

// writeFluxToFile marshals flux to YAML and writes to the given path.
func writeFluxToFile(flux map[string]any, path string) error {
	data, err := yaml.Marshal(flux)
	if err != nil {
		return fmt.Errorf("failed to marshal flux: %w", err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write flux file: %w", err)
	}
	return nil
}
