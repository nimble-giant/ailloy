# pr-comments
description: This command helps PR authors respond to review comments efficiently. It fetches PR details and c...

```

/pr-comments <pr-number>
/pr-comments <pr-url>
/pr-comments

```

## Instructions for Claude

Process this command according to the Ailloy workflow template.
Refer to the full template documentation for detailed instructions.


## Workflow

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
gh api repos/:owner/:repo/pulls/<pr-number>/comments/<comment-id>/replies \
  --method POST \
  --field body="<reply-message>"

# Reply to general comments
gh api repos/:owner/:repo/issues/<pr-number>/comments \
  --method POST \
  --field body="<reply-message>"
```

### Step 4.6: Request Re-review (Optional)

If significant changes were made:

```bash
# Request review from specific reviewers
gh pr edit <pr-number> --add-reviewer <reviewer-username>

# Or post a general comment requesting re-review
gh pr comment <pr-number> --body "Thanks for the feedback! I've addressed all the review comments. Please take another look when you have a chance."
```


## Configuration

This command reads from `.ailloy/ailloy.yaml` for default values.
