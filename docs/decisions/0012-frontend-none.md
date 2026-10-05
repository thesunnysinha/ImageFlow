# ADR 0012: `--frontend none` for API-only projects

- **Status:** Accepted
- **Date:** 2026-10-04

## Context

`--frontend` accepted `react-mui` and `nextjs`. A project that is only an API still received a frontend service, a Compose
entry, an nginx route to it and a CI job that builds it. The Launchpad wizard's "No frontend" choice had to send
`react-mui` and then leave the frontend disabled in the manifest.

## Decision

- `none` is a third accepted value of `--frontend`, in the CLI, `SUPPORTED`, `scaffold.py` and `scaffold.yml`.
- `blueprint_sources` returns no `services/frontend` entry, so nothing is copied. `doctor` and `recover` skip the frontend.
- `render_files` derives the project's Compose and nginx files from the shared templates: for `none` it removes the
  `frontend` service and the proxy's `depends_on` entry, and replaces `location /` with a plain-text 200 that says the project
  is API only. Both edits fail loudly if the template changes shape and still mentions the frontend. `/api/`, `/docs`,
  `/admin/` and `/static/` still reach the backend. `doctor` re-renders with the manifest, so it compares against the
  right files.
- `render_deploy_files` takes the frontend and removes the `frontend:` job from the generated workflows. A
  `blueprints/deploy/backends/<backend>/` overlay was not used: each backend overlay replaces the whole workflow, so a
  frontend dimension would need a file per backend and frontend pair.
- The Launchpad manifest keeps `frontend.enabled: false` (already the default) and gets no `framework`.

## Consequences

- Verified by tests that generate projects: no frontend directory, no frontend text in Compose, nginx or the workflow for
  fastapi, django and nodejs in the deploy modes each supports, a passing `doctor`, and unchanged output for the other
  frontends.
- Not verified: starting the generated Compose stack, or a real Launchpad deployment of an API-only project.
- The Launchpad wizard (common-services, `services/launchpad/lib/template.ts`) still sends `react-mui` for "No frontend".
  Changing it to send `none` is a separate change in that repository.
- A project's own `python run.py doctor` fails because `render.py` reads `docker/templates`, which is not copied into the
  project. This is true for every frontend and was not changed here; run `doctor` from the template with `--root`.
