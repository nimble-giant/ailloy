# Start Issue

Fetch a GitHub issue and begin implementation.

## Usage

```
/start-issue <issue-number>
/start-issue <issue-url>
```

## Examples

- `/start-issue 1234` - Start work on issue #1234
- `/start-issue https://github.com/your-org/your-repo/issues/1234` - Start work using full URL

## Workflow

When this command is used, Claude will:

1. **Fetch the issue details** using `gh issue view <issue-number> --json title,body,author,state,labels,assignees,milestone,number,url,comments`

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

## Manual GitHub Issue Commands

### Fetching Issue Details

**IMPORTANT:** Always use `--json` to avoid GitHub Projects (classic) deprecation errors.

```bash
gh issue view <URL or issue-number> --json title,body,author,state,labels,assignees,milestone,number,url,comments
```

Examples:

- `gh issue view 1234 --json title,body,author,state,labels,assignees,milestone,number,url,comments`
- `gh issue view https://github.com/your-org/your-repo/issues/1234 --json title,body,author,state,labels,assignees,milestone,number,url,comments`

## Notes

- Authentication handled through user's GitHub CLI login
- Use `#<issue-number>` format in commit messages
- Focus on rapid implementation of the single issue
