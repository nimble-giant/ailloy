# Start Issue

Fetch a {{scm_provider}} issue and begin implementation.

## Usage

```
/start-issue <issue-number>
/start-issue <issue-url>
```

## Examples

- `/start-issue 1234` - Start work on issue #1234
- `/start-issue {{scm_base_url}}/your-org/your-repo/issues/1234` - Start work using full URL

## Workflow

When this command is used, Claude will:

1. **Fetch the issue details** using `{{issue_view_cmd}}`

2. **Create a todo list** with tasks derived from the issue requirements

3. **Begin implementation** following the issue requirements and acceptance criteria

4. **Use issue context** to inform commit messages and implementation approach

## Scope Boundaries

**IMPORTANT: Stay focused on the specified issue ONLY.**

- Do NOT search for, fetch, or work on related issues, sub-issues, parent issues, or adjacent issues
- Do NOT follow `#XXXX` references in the issue body to other issues
- Do NOT use `{{issue_list_cmd}}` or `{{issue_search_cmd}}` to discover other issues
- If the issue description mentions other issues for context, note them but do NOT expand scope to include their requirements
- All work should be scoped strictly to what the specified issue describes

## Manual {{scm_provider}} Issue Commands

### Fetching Issue Details

```bash
{{issue_view_cmd}}
```

## Notes

- Authentication handled through user's {{scm_provider}} CLI login
- Use `#<issue-number>` format in commit messages
- Focus on rapid implementation of the single issue
