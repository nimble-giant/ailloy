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

## Template Variables

Template variables allow you to customize templates for your team's specific needs. Variables use the `{{variable_name}}` syntax and are replaced during template processing.

### Common Template Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `default_board` | Default GitHub project board name | "Engineering" |
| `default_priority` | Default issue priority | "P1" |
| `default_status` | Default issue status | "Ready" |
| `organization` | GitHub organization name | "mycompany" |
| `project_id` | GitHub project ID for API calls | "PVT_kwDOBTfXA84A808H" |
| `status_field_id` | GitHub project status field ID | "PVTSSF_..." |
| `priority_field_id` | GitHub project priority field ID | "PVTSSF_..." |
| `iteration_field_id` | GitHub project iteration field ID | "PVTIF_..." |

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

### GitHub Project Integration

For GitHub project integration, you'll need to find your project's field IDs:

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

Then configure the field IDs:

```bash
ailloy customize --set project_id="PVT_kwDOBTfXA84A808H"
ailloy customize --set status_field_id="PVTSSF_lADOBTfXA84A408Hzgtunuz"
ailloy customize --set priority_field_id="PVTSSF_lADOBTfXA84A408Hzgtun9x"
ailloy customize --set iteration_field_id="PVTIF_lADOBTfXA84A408Hzgtun92"
```

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

The interactive mode (`ailloy customize`) guides you through setting up:

1. **Basic variables**: Essential settings like board name, priority, status, and organization
2. **Advanced GitHub Project API**: Optional integration with GitHub Projects v2 API
3. **Custom variables**: Any additional template variables your team needs

The interactive mode provides examples but doesn't force any defaults, ensuring you only configure what's relevant to your team.

This setup ensures consistency across projects while allowing project-specific customization.
