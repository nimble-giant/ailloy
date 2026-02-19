package mold

import (
	"fmt"
	"io/fs"
	"regexp"
	"strings"
)

// semverRegex matches semver strings like "1.0.0", "0.2.0-beta.1", etc.
var semverRegex = regexp.MustCompile(`^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)` +
	`(-((0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(\.(0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?` +
	`(\+([0-9a-zA-Z-]+(\.[0-9a-zA-Z-]+)*))?$`)

// versionConstraintRegex matches version constraints like ">=0.2.0", "^1.0.0", "~1.2.0".
var versionConstraintRegex = regexp.MustCompile(`^[>=<^~!]*\d+\.\d+\.\d+`)

// validFluxTypes is the set of allowed types for flux variable declarations.
var validFluxTypes = map[string]bool{
	"string": true,
	"bool":   true,
	"int":    true,
	"list":   true,
}

// ValidateMold validates a Mold manifest for required fields and correct formats.
func ValidateMold(m *Mold) error {
	var errs []string

	if m.APIVersion == "" {
		errs = append(errs, "apiVersion is required")
	}
	if m.Kind == "" {
		errs = append(errs, "kind is required")
	} else if m.Kind != "mold" {
		errs = append(errs, fmt.Sprintf("kind must be \"mold\", got %q", m.Kind))
	}
	if m.Name == "" {
		errs = append(errs, "name is required")
	}
	if m.Version == "" {
		errs = append(errs, "version is required")
	} else if !semverRegex.MatchString(m.Version) {
		errs = append(errs, fmt.Sprintf("version %q is not valid semver", m.Version))
	}

	if m.Requires.Ailloy != "" && !versionConstraintRegex.MatchString(m.Requires.Ailloy) {
		errs = append(errs, fmt.Sprintf("requires.ailloy %q is not a valid version constraint", m.Requires.Ailloy))
	}

	for i, f := range m.Flux {
		if f.Name == "" {
			errs = append(errs, fmt.Sprintf("flux[%d].name is required", i))
		}
		if f.Type == "" {
			errs = append(errs, fmt.Sprintf("flux[%d].type is required", i))
		} else if !validFluxTypes[f.Type] {
			errs = append(errs, fmt.Sprintf("flux[%d].type %q is not valid (allowed: string, bool, int, list)", i, f.Type))
		}
	}

	for i, d := range m.Dependencies {
		if d.Ingot == "" {
			errs = append(errs, fmt.Sprintf("dependencies[%d].ingot is required", i))
		}
		if d.Version == "" {
			errs = append(errs, fmt.Sprintf("dependencies[%d].version is required", i))
		} else if !versionConstraintRegex.MatchString(d.Version) {
			errs = append(errs, fmt.Sprintf("dependencies[%d].version %q is not a valid version constraint", i, d.Version))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("mold validation failed:\n  - %s", strings.Join(errs, "\n  - "))
	}
	return nil
}

// ValidateMoldFiles checks that all files referenced in the mold manifest exist within the given filesystem.
func ValidateMoldFiles(m *Mold, fsys fs.FS, base string) error {
	var missing []string

	for _, cmd := range m.Commands {
		path := base + "/claude/commands/" + cmd
		if _, err := fs.Stat(fsys, path); err != nil {
			missing = append(missing, path)
		}
	}
	for _, skill := range m.Skills {
		path := base + "/claude/skills/" + skill
		if _, err := fs.Stat(fsys, path); err != nil {
			missing = append(missing, path)
		}
	}
	for _, wf := range m.Workflows {
		path := base + "/github/workflows/" + wf
		if _, err := fs.Stat(fsys, path); err != nil {
			missing = append(missing, path)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("mold references missing files:\n  - %s", strings.Join(missing, "\n  - "))
	}
	return nil
}

// ValidateIngot validates an Ingot manifest for required fields and correct formats.
func ValidateIngot(i *Ingot) error {
	var errs []string

	if i.APIVersion == "" {
		errs = append(errs, "apiVersion is required")
	}
	if i.Kind == "" {
		errs = append(errs, "kind is required")
	} else if i.Kind != "ingot" {
		errs = append(errs, fmt.Sprintf("kind must be \"ingot\", got %q", i.Kind))
	}
	if i.Name == "" {
		errs = append(errs, "name is required")
	}
	if i.Version == "" {
		errs = append(errs, "version is required")
	} else if !semverRegex.MatchString(i.Version) {
		errs = append(errs, fmt.Sprintf("version %q is not valid semver", i.Version))
	}

	if i.Requires.Ailloy != "" && !versionConstraintRegex.MatchString(i.Requires.Ailloy) {
		errs = append(errs, fmt.Sprintf("requires.ailloy %q is not a valid version constraint", i.Requires.Ailloy))
	}

	if len(errs) > 0 {
		return fmt.Errorf("ingot validation failed:\n  - %s", strings.Join(errs, "\n  - "))
	}
	return nil
}

// ValidateIngotFiles checks that all files referenced in the ingot manifest exist within the given filesystem.
func ValidateIngotFiles(i *Ingot, fsys fs.FS, base string) error {
	var missing []string

	for _, f := range i.Files {
		path := base + "/" + f
		if _, err := fs.Stat(fsys, path); err != nil {
			missing = append(missing, path)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("ingot references missing files:\n  - %s", strings.Join(missing, "\n  - "))
	}
	return nil
}
