# ADR 0006: Optional features, copied in at configure time

- **Status:** Accepted
- **Date:** 2026-10-03

## Context

ai-life-coach proved several modules that any chat-style agent needs: moderation, rate limits, a WhatsApp channel.
Copying them by hand into each project is the drift problem the shared runtime solved for the agent code. A
Launchpad "new application" wizard also needs a machine-readable list of what can be switched on.

## Decision

- A feature is a folder `blueprints/features/<id>/` with `feature.json` (label, description, `requires`, pip
  `requirements`, `env`, `services`, `secrets`) and `files/` copied into `services/backend`, tests included.
  `configure --features a,b` resolves `requires`, refuses unknown ids, the Django backend and file collisions, and
  `run.py features --json` prints the catalog.
- Library code goes to `shared/services/<area>/` (not edited per project); the project-owned part is
  `app/features/<id>.py`, which holds wording and wiring and is meant to be edited.
- The FastAPI blueprint gains an always-present hook package, `app/features/`: modules may define `ORDER`,
  `ROUTERS`, `startup`/`shutdown`, `before_chat` (return a fixed reply, or raise an `AgentServiceError` to reject) and
  `after_chat`. The HTTP route and channels call the same `run_chat` pipeline, so a feature applies to every channel.
- Configuration is read from environment variables by each feature, so the application settings class is untouched.
- Platform needs come from the catalog: `redis` adds a Redis container to local Compose and, in `launchpad` mode,
  turns the Redis service on and lists the feature's secret names in the manifest. `vm_tool` mode refuses features
  that need services or secrets, because its manifests do not provide them.
- Not extracted yet: the per-user vector memory store and the Alembic migrations helper, which need a database
  migration story the FastAPI blueprint does not have. Voice notes (docpipe transcription) are left out of the
  WhatsApp channel; unsupported message types are logged and ignored.

## Consequences

New projects get tested, consistent safety and channel code with one flag, and the wizard can render the catalog.
Costs: each feature is a copy, so later fixes reach only newly generated projects (as with the runtime); features
beyond these four need the same discipline (tests, `feature.json`). Cross-feature behaviour is limited to ordering:
for example, moderation does not yet record a violation with the rate limiter.
