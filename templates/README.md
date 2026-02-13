# Ailloy Templates

Ailloy templates are embedded in the CLI binary and automatically copied to projects during initialization. Templates support commands, agents, and hooks for Claude Code workflows.

## Template Location

Templates are embedded in the Go binary from `pkg/templates/` and deployed to:
- **Project templates**: `.claude/commands/` (created by `ailloy init`)
- **Source templates**: `pkg/templates/claude/commands/` (embedded in binary)

## Available Templates

### Command Templates

- **`create-issue.md`**: Generate well-formatted GitHub issues with proper structure and metadata
- **`start-issue.md`**: Fetch GitHub issue details and begin implementation workflow
- **`open-pr.md`**: Create pull requests with structured descriptions
- **`pr-description.md`**: Generate comprehensive PR descriptions
- **`pr-review.md`**: Review pull requests with comprehensive feedback
- **`pr-comments.md`**: Add structured comments to pull requests
- **`update-pr.md`**: Update existing pull requests
- **`preflight.md`**: Pre-flight checks and setup

### Future: Agents & Hooks

Templates for specialized AI agents and workflow hooks are planned for future releases.

## Template Structure

Templates are Markdown files containing detailed instructions for Claude Code. Each template includes:

- **Purpose**: Clear explanation of what the template accomplishes
- **Command syntax**: How to invoke the template in Claude Code
- **Workflow steps**: Detailed instructions for Claude to follow
- **Output format**: Expected structure of results
- **Integration**: GitHub CLI commands and API usage

## Usage with CLI

```bash
# List available templates
ailloy template list

# View a specific template
ailloy template show create-issue
```

## Using Templates with Claude Code

1. **View template**: Use `ailloy template show <template-name>` to see the full instructions
2. **Copy to Claude**: Copy the template content into your Claude Code conversation  
3. **Follow workflow**: Claude will execute the template's instructions

## Template Development

Templates are embedded from `pkg/templates/claude/commands/` during the build process. To add new templates:

1. Create a new `.md` file in `pkg/templates/claude/commands/`
2. Include clear instructions for Claude Code
3. Define workflow steps and expected outputs
4. Rebuild the Ailloy binary to embed the new template
5. Test with `ailloy template show <new-template-name>`

Templates are automatically discovered from the embedded filesystem.
