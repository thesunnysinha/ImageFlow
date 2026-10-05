# ImageFlow

Batch image-processing API. Clients submit image URLs and get a job they can poll; images are processed asynchronously.

The project follows the layout of [master-project-template](https://github.com/thesunnysinha/master-project-template)
(Go + Gin backend, PostgreSQL, Launchpad deployment). The website compresses images in the visitor's browser (no server cost per visit) and is monetised with AdSense; the mobile app and API cover batch and developer use.

## Layout

| Path | What |
|---|---|
| `services/backend` | Go (Gin) API: `cmd/server`, `internal/{config,envelope,httpapi,jobs,database,safeurl}`, `migrations/*.sql`, `openapi.yaml` |
| `services/frontend` | Next.js website: in-browser image compressor and SEO tool pages, AdSense slots with consent defaults, privacy policy, sitemap |
| `services/mobile` | Expo (React Native, TypeScript, Expo Router) app: compress photos on the device (no server), plus server settings in the keystore, job list, new job and live job progress for URL batches |
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
| GET | `/jobs` | same | your jobs, newest first (`limit`, `before` cursor; counts only, no items) |
| GET | `/jobs/{id}` | same | job and its items; only visible to the key that created it |
| GET | `/jobs/{id}/items/{position}/output` | same | the compressed image of a completed item |
| GET | `/jobs/{id}/items/{position}/output-url` | same | 5-minute presigned download URL (S3 storage only; 501 with local storage) |

Source and webhook URLs are rejected when they resolve to private, loopback or link-local addresses, and the worker's
HTTP clients re-check the address at connect time (DNS rebinding, redirects).

## Processing

A worker (in the API process) fetches each image (20 MB and 40 megapixel limits), shrinks it to at most 2048 px on the
longest side and re-encodes it (JPEG quality 80; PNG stays PNG; the original is kept when re-encoding would not make it
smaller). Job status: `queued` → `processing` → `completed`, `partial` or `failed`. When a job has a `webhook_url`, a
signed callback is sent once it finishes (see `DEPLOYMENT.md`).

## Develop

```bash
python run.py doctor        # needs env/env.override.local.yml (python run.py configure generates it in a new project)
python run.py dev           # docker compose up --build; API at http://localhost:8080/api/v1
cd services/backend && go test -race ./...
TEST_DATABASE_URL=postgres://... go test ./internal/jobs   # database tests skip without it
```

## Website

```bash
cd services/frontend
npm install
npm run typecheck && npm test        # unit tests for the compression logic
npm run build && npm run e2e         # a real Chromium compresses real images; also builds an ads-enabled variant
npm run dev                          # http://localhost:5173
```

Ads, consent and deployment: see "Website and AdSense" in `DEPLOYMENT.md`.

## Mobile app

```bash
cd services/mobile
npm install
npm run typecheck && npm test
npx expo start            # scan the QR code with Expo Go, or press a / i for an emulator
```

**Compress photos on this device** works straight away (no server, nothing uploaded): pick up to 20 photos, choose a quality, a longest side and optionally a maximum size in KB, then save or share the results. To process lists of image links, open **Server** and enter your API address and key. On an Android emulator, the host machine is
`http://10.0.2.2:8080`. Plain `http` is for local development only; use `https` anywhere else.

## Status

Phases 1 to 3 are in place: job creation and status, migrations, API-key auth, URL safety, the worker (fetch, compress,
store, retry, signed webhooks, output download), the mobile app, and S3-compatible storage. Not built yet: presigned uploads, user accounts (the app uses a shared API key), push notifications, observability, store builds (EAS).
