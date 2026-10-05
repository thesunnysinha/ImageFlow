# ImageFlow

Batch image-processing API. Clients submit image URLs and get a job they can poll; images are processed asynchronously.

The project follows the layout of [master-project-template](https://github.com/thesunnysinha/master-project-template)
(Go + Gin backend, PostgreSQL, Launchpad deployment, no web frontend: a mobile app will consume the API).

## Layout

| Path | What |
|---|---|
| `services/backend` | Go (Gin) API: `cmd/server`, `internal/{config,envelope,httpapi,jobs,database,safeurl}`, `migrations/*.sql`, `openapi.yaml` |
| `env/` | `env.template.yml` (safe defaults); generated secrets stay in ignored `env/env.override.local.yml` |
| `docker-compose.yml`, `docker/nginx` | local stack: Postgres, backend, proxy on <http://localhost:8080> |
| `launchpad/application.json`, `DEPLOYMENT.md` | Launchpad manifest and one-time setup |
| `.github/workflows/quality.yml` | changelog check, gofmt, vet, race tests against Postgres, build |
| `legacy/` | the original Django/Celery implementation, kept for reference until the new service reaches parity |

## API (`/api/v1`)

All responses use the envelope `{success, code, message, data, meta, trace_id}` and carry `X-Request-ID`.

| Method | Path | Auth | |
|---|---|---|---|
| GET | `/health`, `/ready` | none | liveness (no dependencies) and readiness (database) |
| POST | `/jobs` | `Authorization: Bearer <key>` | body `{"source_urls": [...], "webhook_url": "..."}`, returns 202 + `Location` |
| GET | `/jobs/{id}` | same | job and its items; only visible to the key that created it |

Source and webhook URLs are rejected when they resolve to private, loopback or link-local addresses.

## Develop

```bash
python run.py doctor        # needs env/env.override.local.yml (python run.py configure generates it in a new project)
python run.py dev           # docker compose up --build; API at http://localhost:8080/api/v1
cd services/backend && go test -race ./...
TEST_DATABASE_URL=postgres://... go test ./internal/jobs   # database tests skip without it
```

## Status

Phase 1 is in place: job creation and status, migrations, API-key auth, URL safety checks. Not built yet: the worker
(real fetching and compression, retries, webhooks), object storage with presigned uploads, user accounts, and the
Expo mobile app.
