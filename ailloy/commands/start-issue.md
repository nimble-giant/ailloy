# start-issue
description: AI-assisted workflow command

## Flags
- `--sub-issues`: Enable searching for related sub-issues (default: false)
- `-s`: Shorthand for --sub-issues

## Examples
- `/start-issue 1234` - Start work on issue #1234 (no sub-issue search)
- `/start-issue --sub-issues 1234` - Start work on issue #1234 and search for sub-issues
- `/start-issue -s 1234` - Same as above using shorthand
- `/start-issue https://github.com/testifysec/judge/issues/1234` - Start work using full URL (no sub-issue search)
## Instructions for Claude

Process this command according to the Ailloy workflow template.
Refer to the full template documentation for detailed instructions.


## Workflow

When this command is used, Claude will:

1. **Parse flags** from the command to determine if sub-issue search is enabled

2. **Fetch the issue details** using `gh issue view <issue-number>`

3. **Handle sub-issues based on flag**:
   - **If `--sub-issues` flag is present**: Automatically discover related sub-issues using comprehensive search strategies:
     - Direct references in the issue description and comments (format: #XXXX)
     - Explicit links: `gh issue list --search "mentions:#<issue-number>"` and `gh issue list --search "linked:<issue-number>"`
     - Contextual searches using key terms from the issue title and description
     - Temporal analysis for issues created by the same author around the same time
     - Logical dependency analysis for related implementation steps
   - **If flag is NOT present (default)**: Skip sub-issue discovery and proceed directly to implementation

4. **Create a todo list** with tasks derived from:
   - The main issue requirements
   - Any discovered sub-issues (only if `--sub-issues` flag was used)

5. **Confirm task order** with the user if sub-issues are found, asking which one to start with

6. **Begin implementation** following the issue requirements and acceptance criteria

7. **Use issue context** to inform commit messages and implementation approach


## GitHub CLI Commands

```bash
gh issue view <URL or issue-number>

gh issue list --search "jobs table"
gh issue list --search "All Jobs"
gh issue list --search "pagination"
gh issue list --search "product detail"
gh issue list --search "extract jobs"
gh issue list --search "job type filtering"

```


## Configuration

This command reads from `.ailloy/ailloy.yaml` for default values.
