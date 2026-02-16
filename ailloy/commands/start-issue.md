# start-issue
description: AI-assisted workflow command

## Examples
- `/start-issue 1234` - Start work on issue #1234
- `/start-issue https://github.com/org-name/repo/issues/1234` - Start work using full URL

## Instructions for Claude

Process this command according to the Ailloy workflow template.
Refer to the full template documentation for detailed instructions.


## Workflow

When this command is used, Claude will:

1. **Fetch the issue details** using `gh issue view <issue-number>`

2. **Create a todo list** with tasks derived from the issue requirements

3. **Begin implementation** following the issue requirements and acceptance criteria

4. **Use issue context** to inform commit messages and implementation approach

## Scope Boundaries

**IMPORTANT: Stay focused on the specified issue ONLY.**

- Do NOT search for, fetch, or work on related issues, sub-issues, parent issues, or adjacent issues
- Do NOT follow `#XXXX` references in the issue body to other issues
- Do NOT use `gh issue list` or `gh issue search` to discover other issues
- If the issue description mentions other issues for context, note them but do NOT expand scope to include their requirements
- All work should be scoped strictly to what the specified issue describes


## GitHub CLI Commands

```bash
gh issue view <URL or issue-number>
```


## Configuration

This command reads from `.ailloy/ailloy.yaml` for default values.
