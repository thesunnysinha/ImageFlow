# 0014: Frontend-only generation

Status: accepted

## Context

Some projects need only a React MUI or Next.js frontend. The existing generator assumed every configuration needed a backend and PostgreSQL, including secrets, local Compose, doctor and deployment output.

## Decision

Accept the explicit contract `backend=none`, `frontend=react-mui|nextjs`, `database=none`. Generate a standalone frontend page, frontend local stack and proxy, and the normal runtime commands. Skip backend blueprints, shared agent code, backend/database environments and secrets, and backend quality checks.

For Launchpad, enable the frontend and assign the supplied domain to it. Disable the backend, PostgreSQL and Redis, clear backend environment and secret names, and omit the Next.js API origin. The scaffold script and workflow expose the database choice and pass the same choices through to configure.

Reject an empty stack, database provisioning without a backend, and backend-dependent features without a backend. Existing backend blueprints depend on PostgreSQL; reject `database=none` for them until their architecture supports it. Frontend-only projects support local Compose and Launchpad deployment; the FastAPI-specific `vm_tool` deployment remains unavailable.

## Consequences

Selecting a frontend no longer silently provisions a database or asks for backend credentials when the explicit frontend-only contract is used. Backend defaults remain FastAPI and PostgreSQL. Frontend-only starters provide a local interaction without calling nonexistent auth, agent or health endpoints. Backend-enabled projects retain their existing starters.

Generator, scaffold and runtime checks exercise both frontend choices. Generated frontends are typechecked and built locally, and both standalone dev servers return HTTP 200 at `/` with no backend running. Their authored pages contain no chat requests, AI-ready messaging or API-origin rewrite. Remote workflow dispatch and live Launchpad provisioning require separate integration validation.
