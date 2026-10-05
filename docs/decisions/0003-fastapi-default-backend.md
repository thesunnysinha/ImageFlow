# ADR 0003: FastAPI is the default backend; Django stays selectable

- **Status:** Accepted (recorded after the fact; the change itself landed on 2026-09-28)
- **Date:** 2026-10-03
- **Amends:** [0001](0001-initial-architecture.md)

## Context

ADR 0001 chose Django + React MUI + PostgreSQL as the only supported first stack. The template then gained a FastAPI
backend blueprint with the same agent contract, and `configure` and the README started treating FastAPI as the
default. No record explained the change, so 0001 read as if Django were still the only supported backend.

## Decision

- The supported golden path is **FastAPI + React MUI + PostgreSQL**. `configure` selects FastAPI unless
  `--backend django` is passed.
- **Django remains a supported, selectable backend.** It keeps what FastAPI does not have: the user model,
  cookie-JWT authentication, health/readiness routes, and the checked-in OpenAPI contract with CSRF protection.
- Both backends expose `POST /api/v1/agent/chat` and load the shared agent code from `blueprints/shared/`.
- The FastAPI starter defines no database migrations; Django applies them through `python run.py setup`.
- Everything else in 0001 still stands: serializers or schemas at the HTTP boundary, Pydantic Settings, the uniform
  response envelope, keeping unselected blueprints in the template, and JSON-compatible YAML for the manifest.

## Consequences

New projects start on FastAPI. Projects that need accounts and cookie authentication out of the box choose
Django. The two blueprints must keep the same agent contract, which the shared code and the tests under
`tests/shared_agent` help enforce. Django does not yet use the async tool agent (see 0002).
