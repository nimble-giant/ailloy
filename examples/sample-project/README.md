# Sample Ailloy Project Configuration

This directory shows how Ailloy can be integrated into an existing project.

## Project Structure

```
sample-project/
├── .ailloy/
│   ├── config/
│   │   └── project.yaml
│   └── templates/
│       └── custom-workflow.md
├── src/
│   └── (your application code)
└── README.md
```

## Configuration Files

- `project.yaml` - Project-specific Ailloy configuration
- `custom-workflow.md` - Example custom template for this project

## Usage Examples

```bash
# Initialize Ailloy in this project
ailloy init

# Run a template
ailloy template run create-issue

# List available templates
ailloy template list
```

## Integration Points

This example shows how Ailloy can integrate with:
- GitHub Actions workflows
- Issue tracking systems  
- Development workflows
- Team collaboration processes