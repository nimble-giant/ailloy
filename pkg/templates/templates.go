package templates

import (
	"embed"
	"fmt"
	"io/fs"
	"strings"

	"github.com/nimble-giant/ailloy/pkg/mold"
)

//go:embed all:claude all:github mold.yaml flux*
var embeddedTemplates embed.FS

// FS returns the embedded filesystem for direct access.
func FS() fs.FS {
	return embeddedTemplates
}

// LoadManifest loads and parses the embedded mold.yaml manifest.
func LoadManifest() (*mold.Mold, error) {
	return mold.LoadMoldFromFS(embeddedTemplates, "mold.yaml")
}

// LoadFluxDefaults loads the embedded flux.yaml default values.
func LoadFluxDefaults() (map[string]string, error) {
	return mold.LoadFluxFile(embeddedTemplates, "flux.yaml")
}

// LoadFluxSchema loads the embedded flux.schema.yaml validation schema.
// Returns nil if no schema file exists.
func LoadFluxSchema() ([]mold.FluxVar, error) {
	return mold.LoadFluxSchema(embeddedTemplates, "flux.schema.yaml")
}

// GetTemplate returns the content of a command template file
func GetTemplate(name string) ([]byte, error) {
	path := fmt.Sprintf("claude/commands/%s", name)
	return embeddedTemplates.ReadFile(path)
}

// ListTemplates returns a list of available command template files
func ListTemplates() ([]string, error) {
	entries, err := fs.ReadDir(embeddedTemplates, "claude/commands")
	if err != nil {
		return nil, err
	}

	var templates []string
	for _, entry := range entries {
		if !entry.IsDir() && entry.Name()[len(entry.Name())-3:] == ".md" {
			templates = append(templates, entry.Name())
		}
	}

	return templates, nil
}

// GetSkill returns the content of a skill file
func GetSkill(name string) ([]byte, error) {
	path := fmt.Sprintf("claude/skills/%s", name)
	return embeddedTemplates.ReadFile(path)
}

// ListSkills returns a list of available skill files
func ListSkills() ([]string, error) {
	entries, err := fs.ReadDir(embeddedTemplates, "claude/skills")
	if err != nil {
		return nil, err
	}

	var skills []string
	for _, entry := range entries {
		if !entry.IsDir() && entry.Name()[len(entry.Name())-3:] == ".md" {
			skills = append(skills, entry.Name())
		}
	}

	return skills, nil
}

// GetWorkflowTemplate returns the content of a GitHub workflow template file
func GetWorkflowTemplate(name string) ([]byte, error) {
	path := fmt.Sprintf("github/workflows/%s", name)
	return embeddedTemplates.ReadFile(path)
}

// ListWorkflowTemplates returns a list of available GitHub workflow template files
func ListWorkflowTemplates() ([]string, error) {
	entries, err := fs.ReadDir(embeddedTemplates, "github/workflows")
	if err != nil {
		return nil, err
	}

	var templates []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".yml") {
			templates = append(templates, entry.Name())
		}
	}

	return templates, nil
}
