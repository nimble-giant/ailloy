# update-pr
description: AI-assisted workflow command

## Instructions for Claude

Process this command according to the Ailloy workflow blank.
Refer to the full blank documentation for detailed instructions.


## Workflow

When this command is used, Claude will:

1. **Enter Plan Mode** to outline the PR update process

2. **Verify PR exists** for the current branch using `gh pr view --json number,title,url,state`

3. **Generate updated description** using `/pr-description` command
   - Analyzes current branch changes against main
   - Creates comprehensive PR description
   - Includes issue references if `/gh-issue` was used

4. **Update PR description** using `gh pr edit --body-file <description-file>`

5. **Confirm update** and display PR URL


## Configuration

This command reads from `.ailloy/ailloy.yaml` for default values.
