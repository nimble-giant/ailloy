# Ailloy Configuration Guide

Ailloy uses YAML configuration files to manage settings, template variables, and AI provider configurations. This guide covers the configuration system in detail.

## Configuration Files

Ailloy supports two levels of configuration:

- **Global Configuration**: `~/.ailloy/ailloy.yaml` - User-wide defaults
- **Project Configuration**: `.ailloy/ailloy.yaml` - Project-specific settings

### Configuration Precedence

When both configurations exist:

1. Project configuration takes precedence over global
2. Global configuration provides defaults for undefined values
3. Template variables are merged (project overrides global)

## Configuration Structure

```yaml
# Ailloy Configuration File (ailloy.yaml)

# Project metadata
project:
  name: "My Project"                    # Project display name
  description: "Project description"    # Brief project description
  ai_providers: ["claude"]              # Enabled AI providers
  template_directories: []              # Additional template directories

# Template system configuration
templates:
  default_provider: "claude"            # Default AI provider for templates
  auto_update: true                     # Auto-update templates (future)
  repositories: []                      # Template repositories (future)
  variables:                            # Template variables for customization
    default_board: "Engineering"        # Default GitHub project board
    default_priority: "P1"              # Default issue priority
    default_status: "Ready"             # Default issue status
    organization: "mycompany"           # GitHub organization
    project_id: "PVT_kwDOBTfXA84A808H"  # GitHub project ID
    status_field_id: "PVTSSF_..."       # Status field ID for GraphQL
    priority_field_id: "PVTSSF_..."     # Priority field ID for GraphQL
    iteration_field_id: "PVTIF_..."     # Iteration field ID for GraphQL

# Workflow definitions (future)
workflows:
  issue_creation:
    template: "create-issue"            # Template to use
    provider: "claude"                  # AI provider

# User information
user:
  name: "Your Name"                     # User display name
  email: "your.email@example.com"      # User email

# AI provider configurations
providers:
  claude:
    enabled: true                       # Enable Claude provider
    api_key_env: "ANTHROPIC_API_KEY"    # Environment variable for API key
  gpt:
    enabled: false                      # Enable GPT provider
    api_key_env: "OPENAI_API_KEY"       # Environment variable for API key
```

## Template Engine

Ailloy uses Go's [text/template](https://pkg.go.dev/text/template) engine for template processing. Templates are rendered during `ailloy init`, combining your configuration variables and model state to produce the final output files.

### Variable Syntax

Template variables use `{{variable_name}}` syntax. The engine automatically handles the Go template dot prefix, so you can write:

```markdown
Board: {{default_board}}
Organization: {{organization}}
Project: {{project_id}}
```

The dot-prefixed form `{{.variable_name}}` also works and is required inside Go template directives like `{{if}}` and `{{range}}`.

### Conditional Rendering

Templates can conditionally include or exclude entire sections based on your model configuration:

```markdown
{{if .models.status.enabled}}
Status Field ID: {{.models.status.field_id}}
{{end}}
```

Combine conditions with `or` / `and`:

```markdown
{{if or .models.status.enabled .models.priority.enabled}}
Project field management is configured.
{{end}}
```

Use `{{- ... -}}` trim markers to control whitespace around conditionals:

```markdown
{{- if .models.status.enabled}}
This line has no leading blank line.
{{- end}}
```

### Iterating Over Options

Loop over model options using `range`:

```markdown
**Status Options:**
{{range $key, $opt := .models.status.options}}
- {{$opt.label}}{{if $opt.id}}: `{{$opt.id}}`{{end}}
{{end}}
```

### Accessing Nested Model Data

Model data is available under the `.models` key with three sub-keys: `status`, `priority`, and `iteration`. Each provides:

| Path | Type | Description |
| ---- | ---- | ----------- |
| `.models.<model>.enabled` | bool | Whether the model is configured |
| `.models.<model>.field_id` | string | GitHub Projects field ID |
| `.models.<model>.field_mapping` | string | Display name of the field |
| `.models.<model>.options.<key>.label` | string | Human-readable option label |
| `.models.<model>.options.<key>.id` | string | GitHub Projects option ID |

Example accessing nested data:

```markdown
{{.models.status.options.ready.label}} ({{.models.status.options.ready.id}})
```

### Unresolved Variables

When a template references a variable that isn't defined in your configuration, the engine logs a warning and renders it as an empty string. This allows templates to degrade gracefully when optional variables aren't configured.

## Template Variables

Template variables customize templates for your team's needs. Set them via `ailloy customize` or directly in your YAML configuration.

### Common Variables

| Variable | Syntax | Description | Example |
| -------- | ------ | ----------- | ------- |
| `default_board` | `{{default_board}}` | Default GitHub project board name | "Engineering" |
| `default_priority` | `{{default_priority}}` | Default issue priority | "P1" |
| `default_status` | `{{default_status}}` | Default issue status | "Ready" |
| `organization` | `{{organization}}` | GitHub organization name | "mycompany" |
| `project_id` | `{{project_id}}` | GitHub project ID for API calls | "PVT_kwDOBTfXA84A808H" |

### Setting Template Variables

Use the `ailloy customize` command to manage template variables:

```bash
# Set individual variables
ailloy customize --set default_board="Engineering"
ailloy customize --set default_priority="P1"
ailloy customize --set organization="mycompany"

# Set multiple variables at once
ailloy customize \
  --set default_board="Engineering" \
  --set default_priority="P1" \
  --set organization="mycompany"

# Interactive mode for guided setup
ailloy customize

# View current variables
ailloy customize --list

# Delete a variable
ailloy customize --delete default_board

# Work with global configuration
ailloy customize --global --set default_board="Global Default"
```

## Models

Models represent GitHub Projects v2 fields (status, priority, iteration) and drive conditional template rendering. When a model is enabled, templates can include sections with field IDs, option lists, and GraphQL mutations specific to your project board.

### Model Configuration

Configure models in your `ailloy.yaml`:

```yaml
models:
  status:
    enabled: true
    field_mapping: "Status"           # Display name of the field
    field_id: "PVTSSF_abc123"        # GitHub Projects field ID
    options:
      ready:
        label: "Not Started"
        id: "opt_001"
      in_progress:
        label: "In Progress"
        id: "opt_002"
      in_review:
        label: "In Review"
        id: "opt_003"
      done:
        label: "Done"
        id: "opt_004"
  priority:
    enabled: true
    field_mapping: "Priority"
    field_id: "PVTSSF_def456"
    options:
      p0:
        label: "Critical"
        id: "opt_100"
      p1:
        label: "High"
        id: "opt_101"
      p2:
        label: "Medium"
        id: "opt_102"
      p3:
        label: "Low"
        id: "opt_103"
  iteration:
    enabled: false
```

When models are disabled (the default), conditional template sections that depend on them are excluded from the rendered output.

### Finding GitHub Projects Field IDs

The recommended way to populate model field IDs and option IDs is through the interactive wizard:

```bash
# Launch the interactive wizard with automatic GitHub discovery
ailloy customize
```

The wizard's **GitHub Integration** section uses `gh api graphql` to automatically:

1. Discover all ProjectV2 boards in your organization
2. List fields and their types for the selected board
3. Smart-match model names (Status, Priority, Iteration) to GitHub field names
4. Auto-map option labels (e.g., "In Progress" matches "In progress")

The discovery layer caches responses for the duration of the session to avoid redundant API calls.

**Requirements**: The `gh` CLI must be installed and authenticated (`gh auth login`).

If you prefer to query manually:

```bash
# List your organization's projects
gh project list --owner your-org

# Get project field information (replace PROJECT_NUMBER)
gh api graphql -f query='
{
  organization(login: "your-org") {
    projectV2(number: PROJECT_NUMBER) {
      fields(first: 20) {
        nodes {
          ... on ProjectV2SingleSelectField {
            id
            name
            options {
              id
              name
            }
          }
          ... on ProjectV2IterationField {
            id
            name
          }
        }
      }
    }
  }
}'
```

### How Models Affect Templates

With models **disabled** (default), the `create-issue` template generates a simple issue creation workflow. With models **enabled**, the same template automatically includes:

- Field IDs for your project board
- Status and priority option lists with their IDs
- Ready-to-use GraphQL mutations for setting field values

This means templates adapt to your team's configuration without manual editing.

## Configuration Management

### Initialize Configuration

```bash
# Initialize project configuration (default)
ailloy init

# Initialize global configuration
ailloy init --global
```

### Manual Configuration

You can also manually edit configuration files:

```bash
# Edit project configuration
$EDITOR .ailloy/ailloy.yaml

# Edit global configuration
$EDITOR ~/.ailloy/ailloy.yaml
```

### Configuration Validation

Ailloy automatically validates configuration files when loading. Invalid YAML or missing required fields will result in error messages.

### Migration

If you have an existing project with old configuration structure, simply run:

```bash
ailloy customize --list
```

This will create the new configuration structure and migrate existing settings.

## Best Practices

### Team Configuration

1. **Use Global Defaults**: Set common defaults in global configuration
2. **Project Overrides**: Override specific values in project configuration
3. **Share Project Config**: Commit `.ailloy/ailloy.yaml` to your repository
4. **Document Variables**: Use clear, descriptive variable names

### Security

1. **No Secrets in Config**: Never store API keys or secrets in configuration files
2. **Environment Variables**: Use environment variables for sensitive data
3. **Git Ignore**: Add sensitive config files to `.gitignore` if needed

### Example Team Setup

```bash
# Global defaults for your organization
ailloy customize --global \
  --set organization="mycompany" \
  --set default_priority="P1" \
  --set default_status="Ready"

# Project-specific board
ailloy customize --set default_board="Backend Team"
```

### Interactive Configuration

The interactive mode (`ailloy customize`) launches a guided wizard with five sections:

1. **Project Basics**: Board name and organization name
2. **GitHub Integration**: Enable automatic project discovery via `gh api graphql`, select a board from discovered ProjectV2 boards
3. **Model Configuration**: Enable Status, Priority, and Iteration models; map each to a GitHub Project field with smart matching and auto-map options
4. **Custom Variables**: Add freeform key-value pairs for template rendering
5. **Review & Save**: See a styled summary of all changes before writing to disk

Each section can be skipped by pressing Enter to keep existing values. The wizard uses `charmbracelet/huh` for a polished terminal UI experience.

Non-interactive flag operations (`--set`, `--list`, `--delete`) remain available for scripting and CI use.

This setup ensures consistency across projects while allowing project-specific customization.
