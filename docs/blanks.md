# Ailloy Blanks

Ailloy blanks are Markdown instruction files that live in mold directories. They define Claude Code slash commands and are rendered with flux variables via `ailloy forge` or installed into projects via `ailloy cast`.

## Blank Location

Blanks are stored in mold directories and loaded at runtime:
- **Mold source**: `nimble-mold/.claude/commands/` (the official mold)
- **Project output**: `.claude/commands/` (created by `ailloy cast`)
- **Reader package**: `pkg/blanks/` (the `MoldReader` abstraction)

## Available Blanks

### Command Blanks

- **`create-issue.md`**: Generate well-formatted GitHub issues with proper structure and metadata
- **`start-issue.md`**: Fetch GitHub issue details and begin implementation workflow
- **`open-pr.md`**: Create pull requests with structured descriptions
- **`pr-description.md`**: Generate comprehensive PR descriptions
- **`pr-review.md`**: Review pull requests with comprehensive feedback
- **`pr-comments.md`**: Add structured comments to pull requests
- **`update-pr.md`**: Update existing pull requests
- **`preflight.md`**: Pre-flight checks and setup

### Future: Agents & Hooks

Blanks for specialized AI agents and workflow hooks are planned for future releases.

## Blank Structure

Blanks are Markdown files containing detailed instructions for Claude Code. Each blank includes:

- **Purpose**: Clear explanation of what the blank accomplishes
- **Command syntax**: How to invoke the blank in Claude Code
- **Workflow steps**: Detailed instructions for Claude to follow
- **Output format**: Expected structure of results
- **Integration**: GitHub CLI commands and API usage

## Usage with CLI

```bash
# List available blanks
ailloy mold list

# View a specific blank
ailloy mold show create-issue

# Preview rendered output (dry run)
ailloy forge ./nimble-mold

# Install into current project
ailloy cast ./nimble-mold
```

## Using Blanks with Claude Code

1. **View blank**: Use `ailloy mold show <blank-name>` to see the full instructions
2. **Copy to Claude**: Copy the blank content into your Claude Code conversation
3. **Follow workflow**: Claude will execute the blank's instructions

## Blank Development

Blanks live in mold directories (e.g., `nimble-mold/.claude/commands/`). To add new blanks:

1. Create a new `.md` file in `nimble-mold/.claude/commands/`
2. Add the filename to `nimble-mold/mold.yaml` under `commands:`
3. Include clear instructions for Claude Code
4. Define workflow steps and expected outputs
5. Test with `ailloy forge ./nimble-mold`

Blanks are automatically discovered by the `MoldReader` from the mold's `mold.yaml` manifest.
