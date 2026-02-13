# Start Issue

Work with GitHub issues - fetch details, discover sub-issues, and begin implementation.

## Usage

```
/start-issue [flags] <issue-number>
/start-issue [flags] <issue-url>
```

## Flags

- `--sub-issues`: Enable searching for related sub-issues (default: false)
- `-s`: Shorthand for --sub-issues

## Examples

- `/start-issue 1234` - Start work on issue #1234 (no sub-issue search)
- `/start-issue --sub-issues 1234` - Start work on issue #1234 and search for sub-issues
- `/start-issue -s 1234` - Same as above using shorthand
- `/start-issue https://github.com/your-org/your-repo/issues/1234` - Start work using full URL (no sub-issue search)

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

## Manual GitHub Issue Commands

### Fetching Issue Details

```bash
gh issue view <URL or issue-number>
```

Examples:

- `gh issue view https://github.com/your-org/your-repo/issues/1234`
- `gh issue view 1234` (when in the repository directory)

### Finding Sub-Issues

Use a comprehensive multi-step approach:

1. **Direct References**: Check for "#XXXX" mentions in issue description and comments
2. **Explicit Links**:
   - `gh issue list --search "mentions:#<parent-issue>"`
   - `gh issue list --search "linked:<parent-issue>"`
3. **Contextual Search**: Extract key terms and search variations
   - `gh issue list --search "<keyword>"`
4. **Temporal/Author Analysis**: Look for related issues by same author around same time
5. **Logical Dependencies**: Identify issues that build upon each other

Example comprehensive search for issue #1201:

```bash
gh issue list --search "jobs table"
gh issue list --search "All Jobs"
gh issue list --search "pagination"
gh issue list --search "product detail"
gh issue list --search "extract jobs"
gh issue list --search "job type filtering"
```

## Default Behavior

By default (without flags), the command will:

- Fetch and display the issue details
- Create a todo list based only on the main issue
- Skip sub-issue discovery to start work immediately
- Focus on rapid implementation of the single issue

This streamlined approach ensures faster start times and avoids unnecessary searches when working on standalone issues.

## Notes

- Authentication handled through user's GitHub CLI login
- Use `#<issue-number>` format in commit messages
- Sub-issue discovery is OFF by default - use `--sub-issues` flag to enable
- When sub-issue search is enabled, always verify results using multiple search approaches
- Cross-reference to ensure all sub-issues are discovered when flag is used
