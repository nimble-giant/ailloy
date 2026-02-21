# Respond to PR Comments

## Purpose

This command helps PR authors respond to review comments efficiently. It fetches PR details and comments using {{scm.provider}} CLI, enters plan mode to strategically address each comment, and executes the response plan with code changes and comment replies.

## Command Name

`pr-comments`

## Invocation Syntax

```
/pr-comments <pr-number>
/pr-comments <pr-url>
/pr-comments
```

Examples:

- `/pr-comments 1234` - Address comments on PR #1234
- `/pr-comments {{scm.base_url}}/owner/repo/pull/1234` - Address comments using full URL
- `/pr-comments` - Address comments on current branch's PR (auto-detect)

## Workflow Overview

Claude must execute this workflow when the command is invoked:

1. **Fetch PR Information**: Get PR details and all comments using {{scm.provider}} CLI
2. **Enter Plan Mode**: Present all comments and build response strategy
3. **Interactive Planning**: For each comment, ask user how to respond
4. **Consolidate Plan**: Present final response plan for approval
5. **Execute Plan**: Make code changes and post replies as approved

## Phase 1: PR Information Gathering

### Step 1.1: Determine PR Number

If no PR number provided:

```bash
# Get current branch
CURRENT_BRANCH=$(git branch --show-current)

# Find PR for current branch
{{pr.find_for_branch}}
```

If URL provided, extract the PR number from the URL.

### Step 1.2: Fetch PR Details

```bash
# Get PR information
{{pr.view}}

# Get PR comments (review comments on code)
{{pr.view_reviews}}

# Get detailed review comments
{{api.pr_review_comments}}

# Get general PR conversation comments
{{api.pr_conversation_comments}}
```

### Step 1.3: Parse and Organize Comments

Group comments by:

- **Review comments**: Line-specific code feedback
- **General comments**: Overall PR discussion
- **Resolved comments**: Already addressed
- **Pending comments**: Require response

Extract for each comment:

- Author and timestamp
- Comment body/content
- File and line context (for review comments)
- Comment ID for replies
- Resolution status

## Phase 2: Plan Mode - Comment Response Strategy

### Step 2.1: Enter Plan Mode

Claude must immediately enter plan mode and present all pending comments organized by type and priority.

### Step 2.2: Present Comments for Planning

Format in plan mode:

```markdown
# PR Comment Response Plan for #<number>: <title>

## Summary
- Total comments requiring response: X
- Review comments (code-specific): Y
- General discussion comments: Z

## Comments to Address

### Review Comment 1
**File**: `path/to/file.js:42`
**Author**: @reviewer
**Comment**: "This function could be simplified by extracting the validation logic."

**Suggested Actions**:
- [ ] Refactor function to extract validation
- [ ] Add unit tests for new validation function
- [ ] Reply acknowledging the change

**Your Response Strategy**: [User input needed]

---

### General Comment 2
**Author**: @reviewer
**Comment**: "Could you add documentation for the new API endpoints?"

**Suggested Actions**:
- [ ] Add API documentation
- [ ] Update README with usage examples
- [ ] Reply with documentation link

**Your Response Strategy**: [User input needed]

---

[Continue for all comments...]

## Response Plan Summary
[To be filled based on user input]
```

### Step 2.3: Interactive Comment Planning

For each comment, ask the user:

1. **"How would you like to address this comment?"**
   - Provide specific suggestions based on comment content
   - Options: "Make code changes", "Reply with explanation", "Ask for clarification", "Mark as won't fix"

2. **"What specific changes should I make?"** (if code changes selected)
   - Get detailed instructions from user
   - Confirm file locations and approach

3. **"What should the reply message say?"** (if replying)
   - Suggest professional response templates
   - Get user's preferred tone and content

## Phase 3: Plan Consolidation and Approval

### Step 3.1: Build Final Plan

After gathering all user responses, present consolidated plan:

```markdown
# Final PR Comment Response Plan

## Code Changes
1. **File: `src/api/users.js`**
   - Extract validation logic to separate function
   - Add error handling for edge cases
   
2. **File: `docs/api.md`**
   - Add documentation for new endpoints
   - Include usage examples

## Comment Replies
1. **Reply to @reviewer on line 42**: "Great suggestion! I've extracted the validation logic into a separate function and added tests."

2. **Reply to @reviewer on general comment**: "Documentation added! See the updated API docs at [link]."

## Execution Order
1. Make all code changes
2. Run tests to ensure functionality
3. Commit changes with descriptive message
4. Post replies to all comments
5. Request re-review if needed

**Do you approve this plan? (yes/no)**
```

### Step 3.2: Wait for User Approval

Do not proceed until user explicitly approves the plan.

## Phase 4: Plan Execution

### Step 4.1: Make Code Changes

For each planned code change:

1. Read the current file content
2. Make the specified modifications
3. Validate syntax and formatting
4. Stage the changes

```bash
# Example workflow
git add path/to/modified/file.js
```

### Step 4.2: Run Tests (if available)

```bash
# Detect and run test suite
if [ -f "package.json" ]; then
  npm test
elif [ -f "Cargo.toml" ]; then
  cargo test
elif [ -f "go.mod" ]; then
  go test ./...
fi
```

### Step 4.3: Commit Changes

Create a descriptive commit message referencing the PR:

```bash
git commit -m "address PR #<number> review comments

- Extract validation logic for better separation of concerns
- Add comprehensive API documentation with examples
- Improve error handling in user management functions

Addresses feedback from @reviewer"
```

### Step 4.4: Push Changes

```bash
git push origin <branch-name>
```

### Step 4.5: Post Comment Replies

For each planned reply:

```bash
# Reply to specific review comments
{{api.reply_to_review_comment}}

# Reply to general comments
{{api.reply_to_conversation_comment}}
```

### Step 4.6: Request Re-review (Optional)

If significant changes were made:

```bash
# Request review from specific reviewers
{{pr.request_review}}

# Or post a general comment requesting re-review
{{pr.comment}}
```

## Comment Reply Templates

### Acknowledging and Implementing Feedback

- "Great suggestion! I've implemented this change and added tests to cover the new functionality."
- "You're absolutely right. I've refactored this section to be more maintainable."
- "Thanks for catching this! Fixed in the latest commit."

### Providing Explanations

- "Good question! The reason I implemented it this way is [explanation]. Let me know if you'd like me to add a comment in the code to clarify."
- "This approach was chosen because [reasoning]. Happy to discuss alternative solutions if you have concerns."

### Asking for Clarification

- "Could you provide more details about your preferred approach? I want to make sure I implement this correctly."
- "I'm not sure I understand the issue here. Could you elaborate on what specific problem you're seeing?"

### Disagreeing Respectfully

- "I understand your concern, but I think the current approach is better because [reasoning]. What do you think?"
- "I considered that approach, but went with this implementation due to [constraints/requirements]. Open to discussion!"

## Error Handling

### {{scm.provider}} CLI Issues

- If `{{scm.cli}}` command fails, check authentication: `{{auth.check}}`
- If PR not found, verify the number/URL and repository access
- If API rate limits hit, wait and retry with exponential backoff

### Git Issues

- If push fails due to conflicts, help user resolve merge conflicts
- If tests fail, halt execution and help user fix issues
- If commit fails, check for unstaged changes and git status

### Comment Posting Issues

- If reply posting fails, save reply text and provide manual instructions
- If permissions insufficient, guide user through repository access setup

## Security Considerations

- Never commit sensitive information revealed in comments
- Validate all file paths to prevent directory traversal
- Sanitize comment replies to prevent injection attacks
- Respect repository access permissions

## Integration Notes

This template integrates with:

- {{scm.provider}} CLI for PR and comment management
- Git for code changes and commits
- Project test suites for validation
- Claude Code plan mode for user interaction

## Usage Tips

1. **Run from PR branch**: Execute this command while on the branch associated with the PR
2. **Check permissions**: Ensure you have write access to the repository
3. **Review before approval**: Carefully review the consolidated plan before approving
4. **Test locally**: Consider running tests locally before pushing changes
5. **Follow up**: Monitor the PR for additional comments after responses

## Recap

Use this command to efficiently address all PR review comments in one session. Claude will fetch all comments, help you plan responses strategically, make necessary code changes, and post professional replies - all while keeping you in control of the process through plan mode approval.
