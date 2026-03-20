# Ailloy Lint Demo
# Records the `ailloy lint` command running against the official
# Anthropic Claude plugins marketplace repository.

Output demo/lint-demo.mp4

Require ailloy

Set Shell "bash"
Set FontSize 14
Set Width 1400
Set Height 800
Set Theme "Catppuccin Mocha"
Set WindowBar "Colorful"
Set WindowBarSize 36
Set Padding 16
Set BorderRadius 12
Set TypingSpeed 45ms
Set Framerate 24

# ── Setup (hidden) ──────────────────────────────────────────────────────────
Hide
Type "cd /tmp/lint-demo"
Enter
Sleep 100ms
Type "clear"
Enter
Sleep 200ms
Show

# ── Establish context: Anthropic Claude plugins marketplace ──────────────────
Type "git remote get-url origin"
Enter
Sleep 1s

Type "ls plugins/ | sort"
Enter
Sleep 2s

# ── Show version ─────────────────────────────────────────────────────────────
Type "ailloy --version"
Enter
Sleep 1s

# ── Run lint and wait for it to finish ───────────────────────────────────────
Type "ailloy lint ."
Enter
Sleep 6s

# ── Jump back to start of lint output (hidden — instant cut) ─────────────────
Hide
ScrollUp@1ms 1100
Show
Sleep 1s

# ── Show a sample of findings (first ~6 screens, readable scroll pace) ────────

# Banner + line-count rule header
ScrollDown@80ms 40
Sleep 1s

# More line-count warnings
ScrollDown@80ms 40
Sleep 1s

# More line-count warnings
ScrollDown@80ms 40
Sleep 1s

# structure / agents-md-presence / import-validation headers appear
ScrollDown@80ms 40
Sleep 1s

# duplicate-topics section
ScrollDown@80ms 40
Sleep 1s

# command-frontmatter / plugin-manifest sections
ScrollDown@80ms 40
Sleep 1s

# ── Fast-jump to the final summary ───────────────────────────────────────────
ScrollDown@1ms 2000
Sleep 2s

# ── Pause on "129 file(s) scanned, 99 error(s), 144 warning(s), 75 suggestion(s)" ──
Sleep 5s
