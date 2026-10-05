# ADR 0010: Django can be deployed through Launchpad

- **Status:** Accepted
- **Date:** 2026-10-04

## Context

Only the FastAPI backend could be generated with deployment files; the Django blueprint ran the development server and
had no production image (ADR 0004, 0005). Launchpad runs the backend as a pod that is probed over plain HTTP by pod IP
and receives the database settings as `POSTGRES_*` environment variables, which the Django settings already read.

## Decision

- The Django image runs as UID 10001, applies migrations on start and serves with gunicorn; static files are collected at
  build time and served by WhiteNoise. One replica is assumed, because migrations run at start.
- `HealthCheckMiddleware` answers `GET /api/v1/health` before host validation and the HTTPS redirect, since the
  readiness probe has neither a known Host nor TLS. Every other path keeps `ALLOWED_HOSTS` and the redirect.
- `DJANGO_ALLOWED_HOSTS` and `CSRF_TRUSTED_ORIGINS` are environment settings. The generated manifest sets the first to
  the backend host and the secret names to `DJANGO_SECRET_KEY` and `OPENAI_API_KEY`; a frontend on another host adds its
  origin to the second (Launchpad's console does this).
- Django is allowed in `launchpad` mode only. The `vm_tool` manifests are written for FastAPI. Optional features stay
  FastAPI only, because their files target the FastAPI application.
- The Next.js starter shows a sign-in form when the API answers 401, because Django's chat endpoint needs a user. FastAPI
  never answers 401, so nothing changes there.

## Consequences

- Verified by building the image and running it against Postgres: probe by pod IP over HTTP returns 200, an unknown host
  gets 400, normal paths redirect to HTTPS, static files are served, the container runs as UID 10001, and the Next.js
  starter signed in and reached the chat endpoint through its rewrite. Not verified: a real Launchpad deployment, and a
  browser session across two hosts with Secure cookies.
- The first user is created by hand after the first deploy (`createsuperuser`); `DEPLOYMENT.md` says how.
- `python run.py doctor` still fails inside generated projects (see ADR 0009).
