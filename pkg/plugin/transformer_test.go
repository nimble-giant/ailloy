package plugin

import (
	"strings"
	"testing"
)

func TestNewTransformer(t *testing.T) {
	tr := NewTransformer()
	if tr == nil {
		t.Fatal("expected non-nil transformer")
	}
	if !tr.PreserveVariables {
		t.Error("expected PreserveVariables to be true by default")
	}
	if !tr.SimplifyFormat {
		t.Error("expected SimplifyFormat to be true by default")
	}
}

func TestTransformer_Transform_BasicBlank(t *testing.T) {
	tr := NewTransformer()
	tmpl := BlankInfo{
		Name:        "test-command",
		Description: "A test command",
		Content: []byte(`# Test Command

## Purpose
This is a test command for testing.

## Instructions
1. First step
2. Second step
3. Third step
`),
	}

	output, err := tr.Transform(tmpl)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content := string(output)

	// Should contain command header
	if !strings.Contains(content, "# test-command") {
		t.Error("expected output to contain command header")
	}

	// Should contain description
	if !strings.Contains(content, "description:") {
		t.Error("expected output to contain description field")
	}

	// Should contain instructions section
	if !strings.Contains(content, "## Instructions for Claude") {
		t.Error("expected output to contain instructions section")
	}

	// Should contain configuration section
	if !strings.Contains(content, "## Configuration") {
		t.Error("expected output to contain configuration section")
	}
}

func TestTransformer_ParseBlank(t *testing.T) {
	tr := NewTransformer()
	content := `# Command
## Purpose
This is the purpose.

## Invocation Syntax
` + "`/command`" + `

## Flags
- --flag1: Description 1
- --flag2: Description 2

## Examples
` + "```bash\n/command --flag1\n```" + `

## Instructions
1. Do this
2. Do that
`

	sections := tr.parseBlank(content)

	if sections["purpose"] == "" {
		t.Error("expected purpose section to be parsed")
	}
	if sections["invocation"] == "" {
		t.Error("expected invocation section to be parsed")
	}
	if sections["flags"] == "" {
		t.Error("expected flags section to be parsed")
	}
	if sections["examples"] == "" {
		t.Error("expected examples section to be parsed")
	}
	if sections["instructions"] == "" {
		t.Error("expected instructions section to be parsed")
	}
}

func TestTransformer_NormalizeSection(t *testing.T) {
	tr := NewTransformer()

	tests := []struct {
		header   string
		expected string
	}{
		{"Invocation Syntax", "invocation"},
		{"invocation", "invocation"},
		{"Command Flags", "flags"},
		{"Flag Options", "flags"},
		{"Usage Examples", "examples"},
		{"example", "examples"},
		{"Step-by-step Instructions", "instructions"},
		{"Execution Workflow", "workflow"},
		{"workflow steps", "workflow"},
		{"GitHub CLI Reference", "github-cli"},
		{"CLI Commands", "github-cli"},
		{"Purpose", "purpose"},
		{"Description", "purpose"},
		{"Random Section", "random-section"},
	}

	for _, tt := range tests {
		t.Run(tt.header, func(t *testing.T) {
			result := tr.normalizeSection(tt.header)
			if result != tt.expected {
				t.Errorf("normalizeSection(%q) = %q, want %q", tt.header, result, tt.expected)
			}
		})
	}
}

func TestTransformer_ExtractShortDescription(t *testing.T) {
	tr := NewTransformer()

	t.Run("with purpose section", func(t *testing.T) {
		sections := map[string]string{
			"purpose": "Generate well-formatted GitHub issues",
		}
		desc := tr.extractShortDescription(sections)
		if desc != "Generate well-formatted GitHub issues" {
			t.Errorf("unexpected description: %s", desc)
		}
	})

	t.Run("with long purpose", func(t *testing.T) {
		longPurpose := strings.Repeat("a", 150)
		sections := map[string]string{
			"purpose": longPurpose,
		}
		desc := tr.extractShortDescription(sections)
		if len(desc) > 100 {
			t.Errorf("expected description to be truncated, got length %d", len(desc))
		}
		if !strings.HasSuffix(desc, "...") {
			t.Error("expected truncated description to end with '...'")
		}
	})

	t.Run("without purpose", func(t *testing.T) {
		sections := map[string]string{}
		desc := tr.extractShortDescription(sections)
		if desc != "AI-assisted workflow command" {
			t.Errorf("expected fallback description, got %q", desc)
		}
	})

	t.Run("purpose with empty lines", func(t *testing.T) {
		sections := map[string]string{
			"purpose": "\n\nActual description here",
		}
		desc := tr.extractShortDescription(sections)
		if desc != "Actual description here" {
			t.Errorf("expected 'Actual description here', got %q", desc)
		}
	})
}

func TestTransformer_TransformInvocationSyntax(t *testing.T) {
	tr := NewTransformer()

	t.Run("with code block", func(t *testing.T) {
		syntax := "Use `/create-issue` to create issues"
		result := tr.transformInvocationSyntax(syntax)
		if !strings.Contains(result, "```") {
			t.Error("expected output to contain code block")
		}
		if !strings.Contains(result, "/create-issue") {
			t.Error("expected output to contain command name")
		}
	})

	t.Run("with renamed command", func(t *testing.T) {
		syntax := "Use `/create-github-issue` to create issues"
		result := tr.transformInvocationSyntax(syntax)
		if strings.Contains(result, "create-github-issue") {
			t.Error("expected 'create-github-issue' to be renamed to 'create-issue'")
		}
		if !strings.Contains(result, "/create-issue") {
			t.Error("expected output to contain renamed command")
		}
	})

	t.Run("no code blocks", func(t *testing.T) {
		syntax := "No code blocks here"
		result := tr.transformInvocationSyntax(syntax)
		if result != "" {
			t.Errorf("expected empty result for no code blocks, got %q", result)
		}
	})
}

func TestTransformer_TransformFlags(t *testing.T) {
	tr := NewTransformer()

	flags := `Some intro text
- --flag1: Description one
- --flag2: Description two
Not a flag line
- --flag3: Description three
`

	result := tr.transformFlags(flags)
	if !strings.Contains(result, "## Flags") {
		t.Error("expected Flags header")
	}
	if !strings.Contains(result, "- --flag1: Description one") {
		t.Error("expected flag1")
	}
	if !strings.Contains(result, "- --flag2: Description two") {
		t.Error("expected flag2")
	}
	if !strings.Contains(result, "- --flag3: Description three") {
		t.Error("expected flag3")
	}
	if strings.Contains(result, "Not a flag line") {
		t.Error("non-flag lines should be excluded")
	}
}

func TestTransformer_TransformExamples(t *testing.T) {
	tr := NewTransformer()

	t.Run("with code blocks", func(t *testing.T) {
		examples := "Example:\n```bash\n/create-github-issue --title \"Bug\"\n```"
		result := tr.transformExamples(examples)
		if !strings.Contains(result, "## Examples") {
			t.Error("expected Examples header")
		}
		if strings.Contains(result, "create-github-issue") {
			t.Error("expected 'create-github-issue' to be renamed")
		}
		if !strings.Contains(result, "create-issue") {
			t.Error("expected 'create-issue' in output")
		}
	})

	t.Run("without code blocks", func(t *testing.T) {
		examples := "Just some text without code blocks"
		result := tr.transformExamples(examples)
		if !strings.Contains(result, "## Examples") {
			t.Error("expected Examples header")
		}
		if !strings.Contains(result, "Just some text") {
			t.Error("expected fallback to include raw text")
		}
	})
}

func TestTransformer_TransformInstructions(t *testing.T) {
	tr := NewTransformer()

	instructions := `Here are the steps:
1. First step
2. Second step
- Bullet step one
- Bullet step two
Some non-step line
3. Third step
`

	result := tr.transformInstructions(instructions)
	if !strings.Contains(result, "When this command is invoked, you must:") {
		t.Error("expected standard instruction header")
	}
	if !strings.Contains(result, "1. First step") {
		t.Error("expected first numbered step")
	}
	if !strings.Contains(result, "2. Second step") {
		t.Error("expected second numbered step")
	}
}

func TestTransformer_TransformInstructions_SkipsCodeBlocks(t *testing.T) {
	tr := NewTransformer()

	instructions := "Steps:\n1. Step one\n```bash\nsome code\n```\n2. Step two"
	result := tr.transformInstructions(instructions)

	if strings.Contains(result, "some code") {
		t.Error("code block content should be skipped")
	}
}

func TestTransformer_SimplifyWorkflow(t *testing.T) {
	tr := NewTransformer()

	workflow := `1. Create the issue
2. Set the field ID to match the project
3. Run GraphQL mutation
4. Update the board`

	result := tr.simplifyWorkflow(workflow)

	if !strings.Contains(result, "Create the issue") {
		t.Error("expected regular lines to be preserved")
	}
	if strings.Contains(result, "field ID") {
		t.Error("expected field ID references to be removed")
	}
	if strings.Contains(result, "GraphQL") {
		t.Error("expected GraphQL references to be removed")
	}
	if !strings.Contains(result, "Update the board") {
		t.Error("expected regular lines to be preserved")
	}
}

func TestTransformer_ExtractGitHubCommands(t *testing.T) {
	tr := NewTransformer()

	t.Run("with gh commands", func(t *testing.T) {
		ghCommands := "```bash\ngh issue create --title \"Bug\"\ngh pr create\n```"
		result := tr.extractGitHubCommands(ghCommands)
		if !strings.Contains(result, "gh issue create") {
			t.Error("expected gh issue create command")
		}
		if !strings.Contains(result, "gh pr create") {
			t.Error("expected gh pr create command")
		}
	})

	t.Run("without gh commands", func(t *testing.T) {
		ghCommands := "No commands here"
		result := tr.extractGitHubCommands(ghCommands)
		if !strings.Contains(result, "Refer to GitHub CLI documentation") {
			t.Error("expected fallback message")
		}
	})
}

func TestTransformer_ExtractInstructions_WithClaudeMust(t *testing.T) {
	tr := NewTransformer()

	content := `# Header
Some intro text.
Claude must:
1. Do this first
2. Do this second

## Next Section
Other stuff
`

	result := tr.extractInstructions(content)
	if !strings.Contains(result, "When this command is invoked, you must:") {
		t.Error("expected instruction header")
	}
	if !strings.Contains(result, "1. Do this first") {
		t.Error("expected first instruction")
	}
}

func TestTransformer_ExtractInstructions_FallbackGeneric(t *testing.T) {
	tr := NewTransformer()

	content := "# Header\nSome content without Claude must."
	result := tr.extractInstructions(content)
	if !strings.Contains(result, "Process this command according to the Ailloy workflow blank") {
		t.Error("expected fallback instruction")
	}
}

func TestTransformer_Transform_WithAllSections(t *testing.T) {
	tr := NewTransformer()
	tmpl := BlankInfo{
		Name:        "full-command",
		Description: "A full command with all sections",
		Content: []byte(`# Full Command

## Purpose
A comprehensive test command.

## Invocation Syntax
` + "`/full-command`" + `

## Flags
- --output: Output directory
- --force: Force overwrite

## Examples
` + "```bash\n/full-command --output ./out\n```" + `

## Instructions
1. Parse arguments
2. Validate input
3. Execute command

## Execution Workflow
1. Initialize
2. Process
3. Finalize

## GitHub CLI Reference
` + "```bash\ngh issue list\n```" + `
`),
	}

	output, err := tr.Transform(tmpl)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content := string(output)

	sections := []string{
		"# full-command",
		"description:",
		"## Flags",
		"## Examples",
		"## Instructions for Claude",
		"## Workflow",
		"## GitHub CLI Commands",
		"## Configuration",
	}

	for _, section := range sections {
		if !strings.Contains(content, section) {
			t.Errorf("expected output to contain %q", section)
		}
	}
}
