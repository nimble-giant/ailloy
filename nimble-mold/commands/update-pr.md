# update-pr

Update the description of an existing pull request for the current branch.

## Usage

```
/update-pr
```

## Workflow

When this command is used, Claude will:

1. **Enter Plan Mode** to outline the PR update process

2. **Verify PR exists** for the current branch using `{{pr.check}}`

3. **Generate updated description** using `/pr-description` command
   - Analyzes current branch changes against main
   - Creates comprehensive PR description
   - Includes issue references if `/gh-issue` was used

4. **Update PR description** using `{{pr.update_body}}`

5. **Confirm update** and display PR URL

## Integration

This command integrates with:

- `/pr-description` - For generating updated PR descriptions
- `/gh-issue` - Includes issue references in the description
- `/open-pr` - Called automatically when PR already exists

## Error Handling

- If no PR exists for current branch, suggests using `/open-pr` instead
- If unable to update PR, provides {{scm.provider}} CLI troubleshooting steps

## Notes

- Assumes PR already exists for the current branch
- Regenerates entire description from scratch for accuracy
- Puts Claude into Plan Mode for transparency
- Uses same description format as `/pr-description` command
- Does not include Claude Code attribution in updated PR descriptions
