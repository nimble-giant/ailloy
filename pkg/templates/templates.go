package templates

import (
	"embed"
	"fmt"
	"io/fs"
)

//go:embed all:claude
var embeddedTemplates embed.FS

// GetTemplate returns the content of a template file
func GetTemplate(name string) ([]byte, error) {
	path := fmt.Sprintf("claude/commands/%s", name)
	return embeddedTemplates.ReadFile(path)
}

// ListTemplates returns a list of available template files
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
