# Untracking a secrets file does not remove it from history

- **Date:** 2026-04-11 (from ai-life-coach)

**What happened.** Real API credentials were committed under `env/*.env`. The files were later untracked and
ignored, but the old commits still contain the values.

**Root cause.** `git rm --cached` only affects future commits.

**What to do differently.** Keep real values out of the first commit: ship `*.env.example` placeholders and ignore
the real files from day one (generated projects do). If a secret was ever committed, rotate it; rotation is what
fixes it. Scrubbing history with `git filter-repo` is optional, rewrites every commit hash and needs a force-push.
