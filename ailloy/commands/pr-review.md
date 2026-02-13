# pr-review
description: This command helps reviewers conduct comprehensive code reviews efficiently. It analyzes PR chang...

```
bash
/pr-review <pr-number> [--silent-mode|-s] [--focus=<area>]
/pr-review <pr-url> [--silent-mode|-s] [--focus=<area>]
/pr-review [--silent-mode|-s] [--focus=<area>]

```

## Flags

## Instructions for Claude

Process this command according to the Ailloy workflow template.
Refer to the full template documentation for detailed instructions.


## Workflow

Claude must execute this workflow when the command is invoked:

1. **Parse Flags**: Determine silent mode and focus areas
2. **Fetch PR Information**: Get PR details, diff, and context
3. **Analyze Code Changes**: Comprehensive code review analysis
4. **Generate Review**: Create structured review feedback
5. **Output Mode**: Either save markdown file (silent) or enter plan mode (interactive)


## Configuration

This command reads from `.ailloy/ailloy.yaml` for default values.
