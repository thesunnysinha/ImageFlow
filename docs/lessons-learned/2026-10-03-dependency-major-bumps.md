# Latest majors change APIs

- **Date:** 2026-10-03 (from ai-life-coach)

**What happened.** Installing the latest packages pulled new major versions of the UI library and TypeScript, which
removed props and changed typings the code relied on.

**What to do differently.** Commit lockfiles and use `npm ci` in CI. The generated backend's `requirements.txt` uses
ranges, so pin or lock it (for example with `uv lock`) before a production release.
