package plugin

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
)

// Pre-compiled regex patterns for performance
var (
	numberedStepPattern = regexp.MustCompile(`^\d+\.`)
	stepPrefixPattern   = regexp.MustCompile(`^\d+\.\s*`)
)

// Transformer converts Ailloy templates to Claude Code command format
type Transformer struct {
	// Configuration for transformation
	PreserveVariables bool
	SimplifyFormat    bool
}

// NewTransformer creates a new template transformer
func NewTransformer() *Transformer {
	return &Transformer{
		PreserveVariables: true,
		SimplifyFormat:    true,
	}
}

// Transform converts an Ailloy template to Claude Code command format
func (t *Transformer) Transform(tmpl TemplateInfo) ([]byte, error) {
	content := string(tmpl.Content)

	// Parse the template sections
	sections := t.parseTemplate(content)

	// Build Claude command
	var output bytes.Buffer

	// Write command header
	output.WriteString(fmt.Sprintf("# %s\n", tmpl.Name))
	output.WriteString(fmt.Sprintf("description: %s\n\n", t.extractShortDescription(sections)))

	// Add invocation syntax if present
	if syntax := sections["invocation"]; syntax != "" {
		output.WriteString(t.transformInvocationSyntax(syntax))
		output.WriteString("\n")
	}

	// Add flags section if present
	if flags := sections["flags"]; flags != "" {
		output.WriteString(t.transformFlags(flags))
		output.WriteString("\n")
	}

	// Add examples if present
	if examples := sections["examples"]; examples != "" {
		output.WriteString(t.transformExamples(examples))
		output.WriteString("\n")
	}

	// Add main instructions for Claude
	output.WriteString("## Instructions for Claude\n\n")
	if instructions := sections["instructions"]; instructions != "" {
		output.WriteString(t.transformInstructions(instructions))
	} else {
		// Fallback to extracting from main content
		output.WriteString(t.extractInstructions(content))
	}

	// Add execution workflow if present
	if workflow := sections["workflow"]; workflow != "" {
		output.WriteString("\n\n## Workflow\n\n")
		output.WriteString(t.simplifyWorkflow(workflow))
	}

	// Add GitHub CLI commands reference
	if ghCommands := sections["github-cli"]; ghCommands != "" {
		output.WriteString("\n\n## GitHub CLI Commands\n\n")
		output.WriteString(t.extractGitHubCommands(ghCommands))
	}

	// Add configuration note
	output.WriteString("\n\n## Configuration\n\n")
	output.WriteString("This command reads from `.ailloy/ailloy.yaml` for default values.\n")

	return output.Bytes(), nil
}

// parseTemplate parses the template into sections
func (t *Transformer) parseTemplate(content string) map[string]string {
	sections := make(map[string]string)
	lines := strings.Split(content, "\n")

	currentSection := ""
	sectionContent := strings.Builder{}

	for _, line := range lines {
		// Check for section headers
		if strings.HasPrefix(line, "## ") {
			// Save previous section
			if currentSection != "" {
				sections[currentSection] = strings.TrimSpace(sectionContent.String())
			}

			// Start new section
			header := strings.TrimPrefix(line, "## ")
			currentSection = t.normalizeSection(header)
			sectionContent.Reset()
		} else if strings.HasPrefix(line, "### ") && currentSection != "" {
			// Sub-section within current section
			sectionContent.WriteString(line + "\n")
		} else if currentSection != "" {
			sectionContent.WriteString(line + "\n")
		}
	}

	// Save last section
	if currentSection != "" {
		sections[currentSection] = strings.TrimSpace(sectionContent.String())
	}

	return sections
}

// normalizeSection converts section headers to normalized keys
func (t *Transformer) normalizeSection(header string) string {
	header = strings.ToLower(header)
	header = strings.ReplaceAll(header, " ", "-")
	header = strings.ReplaceAll(header, "_", "-")

	// Map common sections
	switch {
	case strings.Contains(header, "invocation"):
		return "invocation"
	case strings.Contains(header, "flag"):
		return "flags"
	case strings.Contains(header, "example"):
		return "examples"
	case strings.Contains(header, "instruction"):
		return "instructions"
	case strings.Contains(header, "workflow") || strings.Contains(header, "execution"):
		return "workflow"
	case strings.Contains(header, "github") || strings.Contains(header, "cli"):
		return "github-cli"
	case strings.Contains(header, "purpose") || strings.Contains(header, "description"):
		return "purpose"
	default:
		return header
	}
}

// extractShortDescription extracts a concise description from the template
func (t *Transformer) extractShortDescription(sections map[string]string) string {
	// Try purpose section first
	if purpose := sections["purpose"]; purpose != "" {
		lines := strings.Split(purpose, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" && !strings.HasPrefix(line, "#") {
				if len(line) > 100 {
					return line[:97] + "..."
				}
				return line
			}
		}
	}

	return "AI-assisted workflow command"
}

// transformInvocationSyntax simplifies the invocation syntax
func (t *Transformer) transformInvocationSyntax(syntax string) string {
	var output bytes.Buffer

	// Extract the command pattern
	re := regexp.MustCompile("`([^`]+)`")
	matches := re.FindAllStringSubmatch(syntax, -1)

	if len(matches) > 0 {
		// Use the first code block as the syntax
		command := matches[0][1]
		// Simplify the command name
		command = strings.ReplaceAll(command, "create-github-issue", "create-issue")
		command = strings.ReplaceAll(command, "/create-github-issue", "/create-issue")

		output.WriteString("```\n")
		output.WriteString(command)
		output.WriteString("\n```\n")
	}

	return output.String()
}

// transformFlags extracts and formats flag documentation
func (t *Transformer) transformFlags(flags string) string {
	var output bytes.Buffer
	output.WriteString("## Flags\n")

	lines := strings.Split(flags, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "- ") {
			// Keep flag descriptions
			output.WriteString(line + "\n")
		}
	}

	return output.String()
}

// transformExamples formats the examples section
func (t *Transformer) transformExamples(examples string) string {
	var output bytes.Buffer
	output.WriteString("## Examples\n")

	// Extract code blocks
	codeBlocks := regexp.MustCompile("```[a-z]*\n([^`]+)\n```").FindAllStringSubmatch(examples, -1)

	if len(codeBlocks) > 0 {
		output.WriteString("```bash\n")
		for _, block := range codeBlocks {
			code := strings.TrimSpace(block[1])
			// Simplify command names
			code = strings.ReplaceAll(code, "/create-github-issue", "/create-issue")
			code = strings.ReplaceAll(code, "create-github-issue", "create-issue")
			output.WriteString(code + "\n")
			if len(codeBlocks) > 1 {
				output.WriteString("\n")
			}
		}
		output.WriteString("```\n")
	} else {
		// Fallback: include as-is
		output.WriteString(examples)
	}

	return output.String()
}

// transformInstructions converts detailed instructions to Claude-focused format
func (t *Transformer) transformInstructions(instructions string) string {
	var output bytes.Buffer

	// Extract key steps
	lines := strings.Split(instructions, "\n")
	stepNumber := 1
	inCodeBlock := false

	output.WriteString("When this command is invoked, you must:\n\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Track code blocks
		if strings.HasPrefix(line, "```") {
			inCodeBlock = !inCodeBlock
			continue
		}

		if inCodeBlock {
			continue
		}

		// Look for numbered steps or key actions
		if numberedStepPattern.MatchString(line) {
			// Extract the step content
			stepContent := stepPrefixPattern.ReplaceAllString(line, "")
			output.WriteString(fmt.Sprintf("%d. %s\n", stepNumber, stepContent))
			stepNumber++
		} else if strings.HasPrefix(line, "- ") && !strings.Contains(strings.ToLower(line), "example") {
			// Convert bullet points to numbered steps
			stepContent := strings.TrimPrefix(line, "- ")
			output.WriteString(fmt.Sprintf("%d. %s\n", stepNumber, stepContent))
			stepNumber++
		}
	}

	return output.String()
}

// extractInstructions extracts instructions from the full content
func (t *Transformer) extractInstructions(content string) string {
	var output bytes.Buffer

	// Look for "Claude must:" or similar patterns
	if idx := strings.Index(content, "Claude must:"); idx != -1 {
		remaining := content[idx+len("Claude must:"):]
		lines := strings.Split(remaining, "\n")

		output.WriteString("When this command is invoked, you must:\n\n")

		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			if strings.HasPrefix(line, "#") {
				break // Stop at next section
			}
			if numberedStepPattern.MatchString(line) {
				output.WriteString(line + "\n")
			}
		}
	} else {
		// Fallback to generic instruction
		output.WriteString("Process this command according to the Ailloy workflow template.\n")
		output.WriteString("Refer to the full template documentation for detailed instructions.\n")
	}

	return output.String()
}

// simplifyWorkflow reduces workflow complexity for Claude commands
func (t *Transformer) simplifyWorkflow(workflow string) string {
	// Remove overly detailed sections
	lines := strings.Split(workflow, "\n")
	var output bytes.Buffer

	for _, line := range lines {
		// Skip overly technical details
		if strings.Contains(strings.ToLower(line), "field id") ||
			strings.Contains(strings.ToLower(line), "graphql") {
			continue
		}
		output.WriteString(line + "\n")
	}

	return output.String()
}

// extractGitHubCommands pulls out just the essential GitHub CLI commands
func (t *Transformer) extractGitHubCommands(ghCommands string) string {
	var output bytes.Buffer

	// Extract code blocks with gh commands
	codeBlocks := regexp.MustCompile("```bash\n([^`]+)\n```").FindAllStringSubmatch(ghCommands, -1)

	for i, block := range codeBlocks {
		if i > 3 {
			break // Limit to first few examples
		}

		code := strings.TrimSpace(block[1])
		// Only include gh commands
		lines := strings.Split(code, "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "gh ") || strings.HasPrefix(line, "#") {
				output.WriteString(line + "\n")
			}
		}
		output.WriteString("\n")
	}

	if output.Len() == 0 {
		output.WriteString("Refer to GitHub CLI documentation for detailed command usage.\n")
	}

	return "```bash\n" + output.String() + "```\n"
}