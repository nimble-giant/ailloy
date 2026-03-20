# Ailloy Lint Demo
# Records the `ailloy lint` command running against the official
# Anthropic Claude plugins marketplace repository.

Output demo/lint-demo.gif
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
Set Framerate 30

# ── Setup (hidden) ──────────────────────────────────────────────────────────
Hide
Type "cd /tmp/lint-demo"
Enter
Sleep 100ms
Type "clear"
Enter
Sleep 200ms
Show

# ── Show version ────────────────────────────────────────────────────────────
Type "ailloy --version"
Enter
Sleep 1.5s

# ── Briefly show what we're about to lint ───────────────────────────────────
Type "ls plugins/ | sort"
Enter
Sleep 2s

# ── Run the lint command ─────────────────────────────────────────────────────
Type "ailloy lint ."
Enter

# Let the header and first rule appear
Sleep 2s

# Scroll through the output as findings accumulate
ScrollDown 8
Sleep 600ms
ScrollDown 8
Sleep 600ms
ScrollDown 10
Sleep 600ms
ScrollDown 10
Sleep 600ms
ScrollDown 12
Sleep 600ms
ScrollDown 12
Sleep 600ms
ScrollDown 15
Sleep 600ms
ScrollDown 15
Sleep 600ms
ScrollDown 20
Sleep 600ms
ScrollDown 20
Sleep 600ms
ScrollDown 25
Sleep 600ms
ScrollDown 25
Sleep 800ms
ScrollDown 30
Sleep 800ms
ScrollDown 30
Sleep 800ms
ScrollDown 40
Sleep 800ms
ScrollDown 40
Sleep 1s
ScrollDown 50
Sleep 1s
ScrollDown 50
Sleep 1s
ScrollDown 60
Sleep 1.2s
ScrollDown 60
Sleep 1.2s
ScrollDown 80
Sleep 1.5s
ScrollDown 100
Sleep 2s

# Pause on the final summary
Sleep 4s
