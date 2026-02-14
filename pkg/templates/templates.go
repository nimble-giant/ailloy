package templates

import (
	"embed"
	"fmt"
	"io/fs"
)

//go:embed all:claude
var embeddedTemplates embed.FS

// GetTemplate returns the content of a command template file
func GetTemplate(name string) ([]byte, error) {
	path := fmt.Sprintf("claude/commands/%s", name)
	return embeddedTemplates.ReadFile(path)
}

// GetSkill returns the content of a skill template file
func GetSkill(name string) ([]byte, error) {
	path := fmt.Sprintf("claude/skills/%s", name)
	return embeddedTemplates.ReadFile(path)
}

// ListTemplates returns a list of available command template files
func ListTemplates() ([]string, error) {
	return listMarkdownFiles("claude/commands")
}

// ListSkills returns a list of available skill template files
func ListSkills() ([]string, error) {
	return listMarkdownFiles("claude/skills")
}

func listMarkdownFiles(dir string) ([]string, error) {
	entries, err := fs.ReadDir(embeddedTemplates, dir)
	if err != nil {
		return nil, err
	}

	var files []string
	for _, entry := range entries {
		if !entry.IsDir() && entry.Name()[len(entry.Name())-3:] == ".md" {
			files = append(files, entry.Name())
		}
	}

	return files, nil
}
