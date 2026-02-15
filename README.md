# Ailloy

[![CI](https://github.com/nimble-giant/ailloy/actions/workflows/ci.yml/badge.svg)](https://github.com/nimble-giant/ailloy/actions/workflows/ci.yml)
[![Security](https://github.com/nimble-giant/ailloy/actions/workflows/security.yml/badge.svg)](https://github.com/nimble-giant/ailloy/actions/workflows/security.yml)
[![Release](https://github.com/nimble-giant/ailloy/actions/workflows/release.yml/badge.svg)](https://github.com/nimble-giant/ailloy/actions/workflows/release.yml)

![Ailloy Mascot](.assets/Ailloy%20the%20Blacksmith%20Innovator.png)

**Ailloy** is a CLI tool for initializing projects with agentic AI patterns to assist with software development lifecycle (SDLC) tasks. Currently focused on Claude Code integration, Ailloy helps developers set up structured AI workflows using templates and configuration files.

Like in metallurgy—where combining two elements yields a stronger alloy—Ailloy represents the fusion of traditional development practices with AI assistance to create more efficient engineering workflows.

![Ailloy with Claude Integration](.assets/Friendly%20Ailloy%20with%20Glowing%20Orb.png)

## What is Ailloy?

Ailloy helps you:

- **Initialize AI-ready projects**: Set up command template structure for AI-assisted workflows
- **Customize templates**: Configure team-specific defaults for consistent workflows
- **Manage AI templates**: Access pre-built templates for common development tasks
- **Standardize AI workflows**: Use consistent patterns for GitHub issues, PRs, and development tasks

Currently focused on **Claude Code** because it offers the level of customization and template support needed for sophisticated AI-assisted development workflows.

## Quick Start

Get started in minutes with the CLI:

![Ailloy Ready to Help](.assets/Ailloy%20Winking%20with%20Orb%20and%20Hammer.png)

### Installation

#### Quick Install (Recommended)

Download the latest pre-built binary for your platform:

```bash
curl -fsSL https://raw.githubusercontent.com/nimble-giant/ailloy/main/install.sh | bash
```

#### Go Install

If you have Go installed:

```bash
go install github.com/nimble-giant/ailloy/cmd/ailloy@latest
```

#### Build from Source

```bash
git clone https://github.com/nimble-giant/ailloy
cd ailloy
make build
```

The binary will be available at `./bin/ailloy`.

### Initialize a Project

```bash
# Set up Ailloy structure in your current repository (default)
ailloy init

# Set up global user configuration
ailloy init --global
```

### Customize Templates

```bash
# Set team-specific defaults
ailloy customize --set default_board="Engineering" --set default_priority="P1" --set organization="mycompany"

# View current configuration
ailloy customize --list

# Interactive configuration mode
ailloy customize

# Configure global defaults (applies to all projects)
ailloy customize --global --set default_board="My Default Board"
```

### Working with Templates

```bash
# List available templates
ailloy template list

# View a specific template
ailloy template show create-issue
```

## Available Commands

### `ailloy init`

Initialize Ailloy configuration:

- By default, sets up Ailloy structure in the current repository
- `-g, --global`: Install user-level configuration instead

### `ailloy customize`

Configure template variables for team-specific defaults:

- `--set key=value`: Set template variables
- `--list`: List current template variables
- `--delete key`: Delete a template variable
- `--global`: Work with global configuration (vs project-specific)
- Interactive mode (no flags): Guided setup for common variables

### `ailloy template`

Manage AI command templates:

- `list`: Show all available templates
- `show <template-name>`: Display template content

## Templates

Ailloy includes pre-built templates for common SDLC tasks, optimized for Claude Code:

### Available Templates

- **`brainstorm`**: Analyze an idea for feasibility, scope, and value using structured brainstorming techniques
- **`create-issue`**: Generate well-formatted GitHub issues with proper structure
- **`start-issue`**: Fetch GitHub issue details and begin implementation
- **`open-pr`**: Create pull requests with structured descriptions
- **`pr-description`**: Generate comprehensive PR descriptions
- **`update-pr`**: Update existing pull requests

### Available Skills

Skills are proactive workflows that Claude Code can use automatically based on context, without requiring explicit slash command invocation:

- **`brainstorm`**: Structured brainstorming methodology for evaluating ideas using freewriting, cubing, and journalistic techniques

### Workflow Templates

Ailloy also includes GitHub Actions workflow templates:

- **`claude-code`**: GitHub Actions workflow for the [Claude Code agent](https://github.com/anthropics/claude-code-action). Responds to `@claude` mentions in issues, PR comments, and PR reviews. Requires an `ANTHROPIC_API_KEY` secret in your repository.
- **`claude-code-review`**: GitHub Actions workflow for automated PR reviews with the [Claude Code agent](https://github.com/anthropics/claude-code-action). Features brevity-focused formatting, collapsible sections for detailed analysis, and intelligent comment management (updates summary comments, creates reply comments). Requires an `ANTHROPIC_API_KEY` secret in your repository.

### Using Templates

Templates are Markdown files containing instructions for Claude Code. You can:

1. View templates: `ailloy template show create-issue`
2. Copy template content into Claude Code conversations
3. Customize templates for your project's specific needs

### Template Structure

Each template includes:

- Clear instructions for Claude Code
- Context requirements
- Expected output format
- Integration with GitHub CLI commands

### Template Customization

Templates support variables using `{{variable_name}}` syntax. Common variables include:

- `{{default_board}}`: Default GitHub project board name
- `{{default_priority}}`: Default issue priority (P0, P1, P2)
- `{{default_status}}`: Default issue status (Ready, In Progress, etc.)
- `{{organization}}`: GitHub organization name
- `{{project_id}}`: GitHub project ID for API calls
- `{{status_field_id}}`: GitHub project status field ID
- `{{priority_field_id}}`: GitHub project priority field ID

Variables are automatically replaced when templates are copied during `ailloy init`.

## Configuration

Ailloy uses YAML configuration files to store settings and template variables:

### Configuration Files

- **Project**: `.ailloy/ailloy.yaml` - Project-specific configuration
- **Global**: `~/.ailloy/ailloy.yaml` - User-wide defaults

### Configuration Structure

```yaml
# Ailloy Configuration
project:
  name: "My Project"
  description: "Project description"
  ai_providers: ["claude"]
  template_directories: []

templates:
  default_provider: "claude"
  auto_update: true
  repositories: []
  variables:
    default_board: "Engineering"
    default_priority: "P1"
    default_status: "Ready"
    organization: "mycompany"
    # GitHub Project API integration (optional)
    project_id: "PVT_kwDOBTfXA84A408H"
    status_field_id: "PVTSSF_..."
    priority_field_id: "PVTSSF_..."
    iteration_field_id: "PVTIF_..."

workflows:
  issue_creation:
    template: "create-issue"
    provider: "claude"

user:
  name: "Your Name"
  email: "your.email@example.com"

providers:
  claude:
    enabled: true
    api_key_env: "ANTHROPIC_API_KEY"
  gpt:
    enabled: false
    api_key_env: "OPENAI_API_KEY"
```

### Variable Precedence

When both global and project configurations exist:

1. Project variables take precedence over global variables
2. Global variables serve as defaults for undefined project variables
3. Variables are merged automatically during template processing

## Project Structure

```text
/cmd/ailloy          # CLI tool entry point
/internal            # Private Go packages
  /commands          # CLI command implementations
/templates           # AI command templates
  /claude            # Claude Code-specific templates
/docs                # Documentation
/examples            # Example configurations
```

## Current Status

**Alpha Stage**: Ailloy is an early-stage tool focused on Claude Code integration. The CLI provides:

- ✅ Project initialization with command template setup
- ✅ Template management and viewing
- ✅ Template customization with team-specific variables
- ✅ YAML configuration system with global and project scopes
- ✅ Claude Code-optimized workflow templates
- 🔄 Additional AI provider support (planned)
- 🔄 Advanced workflow automation (planned)

## Contributing

This is an evolving framework—community input welcome! As AI development practices mature, Ailloy aims to capture and standardize the most effective patterns for human-AI collaboration.

### Prerequisites

**Option 1: Flox (Recommended)**

[Flox](https://flox.dev/) provides a reproducible development environment with all dependencies:

```bash
# Install Flox (one-time)
curl -fsSL https://flox.dev/install | bash

# Activate the development environment
flox activate

# Verify dependencies
make check-deps
```

**Option 2: Manual Installation**

If you prefer to manage dependencies manually:

| Dependency | Purpose | Installation |
|------------|---------|--------------|
| [Go 1.24+](https://go.dev/dl/) | Build the CLI | `brew install go` |
| [golangci-lint](https://golangci-lint.run/) | Linting | `brew install golangci-lint` |
| [lefthook](https://github.com/evilmartians/lefthook) | Git hooks | `brew install lefthook` |
| [Docker](https://www.docker.com/) | Required for local CI | [Docker Desktop](https://www.docker.com/products/docker-desktop/) |
| [act](https://github.com/nektos/act) | Run GitHub Actions locally | `brew install act` |
| [gh](https://cli.github.com/) | GitHub CLI | `brew install gh` |

### Development Setup

```bash
# Clone the repository
git clone https://github.com/nimble-giant/ailloy
cd ailloy

# Activate Flox environment (or install deps manually)
flox activate

# Install git hooks (runs checks automatically on commit/push)
make hooks

# Build the project
make build

# Run tests
make test

# Run linter
make lint

# Run CI locally (requires Docker)
make ci
```

### To Contribute

1. Fork the repository
2. Create a feature branch
3. Add templates or improve CLI functionality
4. Ensure tests pass: `make test`
5. Ensure linting passes: `make lint`
6. Submit a pull request

## Contributors

<a href="https://github.com/nimble-giant/ailloy/graphs/contributors">
  <img src="https://contrib.rocks/image?repo=nimble-giant/ailloy" />
</a>

## License

Apache License 2.0 - see [LICENSE.md](LICENSE.md) for details.
