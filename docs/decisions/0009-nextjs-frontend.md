# ADR 0009: A selectable Next.js frontend that reaches the API through rewrites

- **Status:** Accepted
- **Date:** 2026-10-04

## Context

`--frontend` accepted only `react-mui` (Vite), and `blueprints/frontend-nextjs/` was an empty folder. Launchpad can
already provision a Next.js project on Vercel (`frontend.framework`), so a second real choice only needed a blueprint
and the generator wiring. Django still has no production image, so the backend choice stays FastAPI for deployed
projects.

## Decision

- `frontend-nextjs` is a small App Router project (health check and a chat form against `/api/v1/agent/chat`) with
  `dev`, `build` and `typecheck` scripts, so the generated CI workflow works unchanged. Its dev server listens on 5173,
  where the local nginx proxy already sends the site.
- The browser always calls the relative path `/api/v1`. Locally nginx routes it to the backend. On Vercel,
  `next.config.mjs` forwards it to `API_ORIGIN`, which the Launchpad manifest sets from the project's domain. The
  request is server to server, so the backend needs no CORS configuration, and nothing is added to the backend.
- The Launchpad manifest now records `frontend.framework` (`vite` for react-mui, `nextjs` for nextjs).
- `scaffold.yml` and `scripts/scaffold.py` take a `frontend` input and pass it to `configure`.

## Consequences

- Both frontends are tested for what they generate; only the Next.js app was also installed, type-checked and built
  by hand. The template has no CI that builds generated frontends.
- `python run.py doctor` fails inside any generated project (it looks for `docker/templates/`, which is not copied).
  That predates this change and is not fixed here.
- Django remains local-only until it has a production image.
