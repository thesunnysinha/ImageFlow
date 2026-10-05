# Architecture

## Template and generated project

The master template keeps reusable generator code under `cli/`, framework/frontend source under `blueprints/`, and reusable Compose/Nginx source under `docker/templates/`. `cli/commands/` has one module per command; `cli/engine/` contains manifest validation, rendering, and transactional promotion. `configure` selects FastAPI or Django, copies the selected backend and React source into `services/`, and renders the manifest, Compose, proxy, and environment files. FastAPI is the default; Django remains supported.

The shared agent graph/provider/state are in `blueprints/shared/services/agent/`. Each backend adapter supplies its own tool registry while preserving the same `POST /api/v1/agent/chat` request/response contract. Company-specific behavior remains outside these reusable blueprints.

`project.config.yml` records supported selections. `env/env.template.yml` contains safe defaults; generated secrets stay in ignored `env/env.override.local.yml`. Generated service `.env` files derive from those inputs. `doctor` detects missing and out-of-sync outputs.

## Runtime request path

The browser sends same-origin requests through Nginx. `/api/` routes to the selected backend and other paths route to Vite, except API documentation endpoints. Django connects to PostgreSQL and applies migrations explicitly through `python run.py setup`. The FastAPI starter agent does not define database migrations.

## API and authentication

The Django API contract is checked in at `blueprints/backend-django/api/openapi.yaml`; FastAPI publishes generated OpenAPI at `/openapi.json` and keeps its starter contract at `blueprints/backend-fastapi/openapi.yaml`. Both backends expose the shared agent contract. Django additionally provides health/readiness and JWT-cookie user/auth routes. Mutating Django endpoints require CSRF protection. Request IDs are returned in headers and the response envelope.

## Configure safety

`configure --dry-run` previews outputs without writes. Real generation stages files on the target filesystem, stores a recoverable snapshot under `.configure-backups/`, writes an interruption marker, then promotes staged outputs. `recover` removes partial generated outputs, restores missing source blueprints from the snapshot, and preserves local secret overrides. Unselected blueprints are retained in the master template.
