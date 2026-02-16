# Claude Code Configuration

This project is set up with Ailloy templates for AI-assisted development workflows.

## Available Commands

The following command templates are available in the `.claude/commands/` directory:

- **create-issue**: Generate well-formatted GitHub issues with proper structure
- **start-issue**: Fetch GitHub issue details and begin implementation
- **open-pr**: Create pull requests with structured descriptions
- **pr-description**: Generate comprehensive PR descriptions
- **pr-comments**: Respond to PR review comments efficiently
- **pr-review**: Conduct comprehensive code reviews with silent/interactive modes
- **update-pr**: Update existing pull requests
- **brainstorm**: Analyze an idea for feasibility, scope, and value using structured brainstorming techniques

## Available Skills

The following skills are available in the `.claude/skills/` directory:

- **brainstorm**: Structured brainstorming methodology for evaluating ideas using freewriting, cubing, and journalistic techniques

The following workflow templates are available in `.github/workflows/`:

- **claude-code**: GitHub Actions workflow for the Claude Code agent (responds to @claude mentions in issues and PRs)

## Usage

To use a command template:

1. Open the template file from the `.claude/commands/` directory
2. Copy the template content into your Claude Code conversation
3. Use the command syntax specified in the template

## Git Hooks (lefthook)

This project uses [lefthook](https://github.com/evilmartians/lefthook) for graduated local checks. Install with `make hooks`.

| Hook         | What Runs                                          |
| ------------ | -------------------------------------------------- |
| `pre-commit` | `go vet` + `gofmt` check (staged `.go` files only) |
| `commit-msg` | conform (conventional commits)                     |
| `pre-push`   | `golangci-lint` + `go build` + `go test -race`     |

## Project Setup

This project was initialized with Ailloy to provide structured AI workflows for:
- GitHub issue management
- Pull request workflows
- Development task automation

For more information about Ailloy, visit: https://github.com/nimble-giant/ailloy
