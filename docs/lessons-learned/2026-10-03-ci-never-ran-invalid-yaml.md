# CI never ran because of one unquoted YAML value

- **Date:** 2026-10-03 (from ai-life-coach)

**What happened.** Every GitHub Actions run failed in zero seconds with no jobs. A large PR was merged whose only
visible check was a secret scanner, in the belief that the lint and tests that passed locally were also enforced.

**Root cause.** `DATABASE_URL: sqlite:///:memory:` ends in a colon, so YAML parsed it as a nested mapping and the
whole workflow was invalid. GitHub reports that as a failed run with no jobs, which is easy to read as "CI is just
failing".

**What to do differently.** Quote values that contain `:`; parse workflow files locally (`yaml.safe_load`) before
pushing; and confirm a PR shows the expected jobs before treating "no failures" as green. The generated *Quality
checks* workflow parses all manifests and workflows for this reason.
