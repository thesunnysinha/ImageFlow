# ADR 0001: First golden-path stack and configure safety

- **Status:** Accepted; the first-stack decision is amended by [0003](0003-fastapi-default-backend.md) (FastAPI is now the default backend)
- **Date:** 2026-09-28

## Context

The master template plan describes several frameworks, databases, and optional systems, but the workspace began without an application repository. Implementing every combination at once would leave the configure survey ahead of the code it promises. The first slice needs to be useful while preserving recovery from generation errors.

## Decision

- Implement Django + React MUI + PostgreSQL as the only supported first stack.
- Use Django REST Framework serializers at the HTTP boundary, Pydantic Settings for backend configuration, and a checked-in OpenAPI contract.
- Use the source plan's uniform success/error response envelope and UUIDv4 custom user identifiers.
- Store browser JWTs in HTTP-only cookies and require CSRF protection for state-changing cookie-authenticated requests.
- Keep unselected source blueprints. Defer pruning until generation and recovery are proven.
- Use JSON-compatible YAML for the CLI manifest/template in Phase 0 so the configurator can run using the Python standard library.

## Consequences

The first configure flow is narrower than the eventual survey, but every exposed choice maps to an available blueprint. Generated projects are easy to inspect and recover. The current implementation does not yet demonstrate framework parity, alternative databases, or the later optional capabilities; those remain phase work.

## References

- `PLAN.md` (source architecture plan)
- `outputs/IMPLEMENTATION_PLAN.md` (phased execution plan)
