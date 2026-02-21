# Sample Ailloy Project Configuration

This directory shows how Ailloy can be integrated into an existing project.

## Project Structure

```
sample-project/
├── .ailloy/
│   ├── config/
│   │   └── project.yaml
│   └── blanks/
│       └── custom-workflow.md
├── src/
│   └── (your application code)
└── README.md
```

## Configuration Files

- `project.yaml` - Project-specific Ailloy configuration
- `custom-workflow.md` - Example custom blank for this project

## Usage Examples

```bash
# Initialize Ailloy in this project
ailloy init

# Run a blank
ailloy mold run create-issue

# List available blanks
ailloy mold list
```

## Integration Points

This example shows how Ailloy can integrate with:
- GitHub Actions workflows
- Issue tracking systems  
- Development workflows
- Team collaboration processes