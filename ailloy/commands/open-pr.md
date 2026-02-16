# open-pr
description: AI-assisted workflow command

## Examples
- `/open-pr` - Create PR ready for review
- `/open-pr draft` - Create PR in draft mode
## Instructions for Claude

Process this command according to the Ailloy workflow template.
Refer to the full template documentation for detailed instructions.


## Workflow

When this command is used, Claude will:

1. **Enter Plan Mode** to outline the PR creation/update process

2. **Check for existing PR** on the current branch using `gh pr view --json number,title,url,state`

3. **If PR already exists**:
   - Use `/update-pr` to update the existing PR description
   - Skip to step 6

4. **If no PR exists**:
   - Generate PR description using `/pr-description` command
   - Create new PR using `gh pr create`

5. **Set PR status**:
   - Default: Ready for review
   - With `draft` flag: Draft mode

6. **Display PR URL** for user to review


## Configuration

This command reads from `.ailloy/ailloy.yaml` for default values.
