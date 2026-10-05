# ADR 0011: A Node.js backend with the same HTTP contract

- **Status:** Accepted
- **Date:** 2026-10-04

## Context

The template offered FastAPI and Django backends, both Python and both built on the shared LangGraph agent runtime. A
Node.js option was requested. The shared runtime is Python, so a Node backend cannot reuse it.

## Decision

- `blueprints/backend-nodejs` is a Fastify and TypeScript service that keeps the contract the frontends and Launchpad rely
  on: `GET /api/v1/health` (no dependencies, answers any Host), `GET /api/v1/ready` (database), `POST
  /api/v1/agent/chat` with the same request and response fields, and the same envelope
  `{success, code, message, data, meta, trace_id}` with `X-Request-ID`.
- The agent is one model call with persisted history (`chat_messages` in Postgres, created at start). It has **no tools,
  approval flow or guardrail layer**: those live in the Python runtime and are not ported. The deployment notes say so.
- Database settings come from `DATABASE_URL` or the `POSTGRES_*` variables, which Launchpad and local Compose both supply.
- Like Django, Node.js is accepted in `launchpad` mode only, and optional features stay FastAPI only.
- `blueprints/deploy/backends/<backend>/` overlays the deploy files, so a Node project gets a Node CI workflow instead of
  the Python one. The shared Python runtime is not copied into a Node project.
- The generator now ignores `node_modules`, `.next`, `dist` and `__pycache__` when copying blueprints.
- The Django blueprint gained tests and a `ruff.toml` excluding migrations: the generated CI ran `pytest` and `ruff`, and
  `pytest` fails when it finds no tests and `ruff` flagged the generated migration.

## Consequences

- Verified: typecheck and 7 tests pass; the image builds, boots against Postgres, creates its table, answers the probe
  with a bare-IP Host, returns the right errors, runs as UID 10001 and reports healthy; the Postgres store was exercised
  against a real database. Django's 4 new tests and the generated lint pass on Python 3.13.
- Not verified: a real Launchpad deployment; the Next.js starter against this backend; any model call (no key was used).
- A project that needs tools, approvals or guardrails on Node must build them; the Python backends have them.
