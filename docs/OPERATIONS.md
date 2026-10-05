# Operating ImageFlow

How the pieces fit, every setting, what to check when something looks wrong, and what is still missing before a public
launch. `DEPLOYMENT.md` covers the Launchpad setup; this page covers running it.

## What runs where

| Piece | Where | State |
|---|---|---|
| API and worker | `services/backend`, one Go process (image: static binary, UID 10001, port 8000) | none: everything is in Postgres and storage |
| Database | PostgreSQL 16 | jobs, items, webhook state, `schema_migrations` |
| Image storage | local disk volume or any S3-compatible bucket | compressed results, deleted after `RETENTION_HOURS` |
| Website | `services/frontend`, Next.js (image: standalone server, UID 10001, port 5173) | none; compression runs in the visitor's browser |
| Mobile app | `services/mobile`, Expo | none on the server; the API key lives in the device keystore |
| Proxy (local stack only) | nginx on port 8080: `/api/` to the backend, everything else to the website | none |

## Settings

Backend (environment variables; the service refuses to start on invalid values):

| Variable | Default | Meaning |
|---|---|---|
| `DATABASE_URL` | built from `POSTGRES_*` | full connection string; wins over the next five |
| `POSTGRES_HOST` / `PORT` / `DB` / `USER` / `PASSWORD` | `database` / `5432` / `app` / `app` / none | used when `DATABASE_URL` is empty |
| `API_KEYS` | **required** | comma-separated bearer keys; each key is a separate owner of its jobs |
| `WEBHOOK_SECRET` | **required while the worker runs** | signs webhook callbacks |
| `RUN_WORKER` | `true` | `false` makes an API-only replica (also skips the janitor) |
| `WORKER_CONCURRENCY` | `4` | images processed at once per process (1 to 64) |
| `MAX_ITEMS_PER_JOB` | `1000` | URLs accepted in one job |
| `RATE_LIMIT_PER_MINUTE` | `120` | requests per minute per API key (burst of a quarter of that, at least 5); `0` turns it off. In memory per process, so with several replicas the effective limit is the sum. Health checks are never limited. |
| `RETENTION_HOURS` | `168` | finished jobs and their images are deleted after this long; `0` keeps everything |
| `STORAGE_BACKEND` | `local` | `local` or `s3` |
| `STORAGE_DIR` | `/data/images` | local storage directory (mount a volume here) |
| `S3_ENDPOINT` `S3_BUCKET` `S3_ACCESS_KEY` `S3_SECRET_KEY` | required with `s3` | endpoint is `host[:port]` with no scheme; the bucket must exist |
| `S3_REGION` / `S3_USE_SSL` / `S3_PREFIX` | `us-east-1` / `true` / empty | |
| `HOST` / `PORT` | `0.0.0.0` / `8000` | listen address |

Limits fixed in code (change them in `internal/imageproc` and `internal/worker`): 20 MB per downloaded image, 40
megapixels, longest side 2048 px, JPEG quality 80, 3 attempts per image with exponential backoff, a 5-minute lease for a
crashed worker, 5 webhook attempts.

Website (build time, because `NEXT_PUBLIC_*` is inlined into the pages): `NEXT_PUBLIC_SITE_URL`,
`NEXT_PUBLIC_CONTACT_EMAIL`, `NEXT_PUBLIC_ADSENSE_CLIENT`, `NEXT_PUBLIC_ADSENSE_SLOT_TOP|CONTENT|BOTTOM`; at runtime
`API_ORIGIN` (Vercel forwards `/api/v1` there). The container takes the same values as Docker build arguments.

## Run it locally

```bash
python3 scripts/smoke_env.py      # throwaway secrets and env files (git-ignored)
docker compose up --build         # website and API on http://localhost:8080
scripts/smoke.sh                  # the same stack, then checks it end to end and tears it down
```

## Day-to-day

- **Is it up?** `GET /api/v1/health` (the process) and `GET /api/v1/ready` (can it reach the database). Containers carry a
  healthcheck (`/server -healthcheck`), so Compose and Kubernetes restart a dead process.
- **Logs** are JSON on stdout: one `request` line per call with `trace_id` (also returned as `X-Request-ID`, so a user's
  report can be matched to a log line), `retention sweep` lines when something is deleted, and `worker step failed` or
  `item failed` / `webhook failed` warnings with the reason.
- **Scaling:** every replica can run API and worker; the queue is Postgres (`FOR UPDATE SKIP LOCKED`), so extra replicas
  share the work. Use `STORAGE_BACKEND=s3` before running more than one replica, because local disk is per container.
- **Database backup:** `docker compose exec database pg_dump -U app app > backup.sql`; restore with `psql` into an empty
  database (the service re-applies any missing migrations at start). Stored images are transient by design, so back up
  the database, not the image storage.
- **Migrations** apply automatically at start under an advisory lock, in file order, once. Add new ones as
  `services/backend/migrations/NNNN_name.sql`; never edit one that has shipped.

## When something is wrong

| Symptom | Likely cause and what to do |
|---|---|
| `/ready` returns 503 | The database is unreachable or restarting. Check `DATABASE_URL`/`POSTGRES_*` and that Postgres is up; the API keeps retrying on its own. |
| Jobs stay `queued` | The worker is not running (`RUN_WORKER=false` everywhere, or startup stuck on migrations: look for `could not prepare the database`). |
| An item shows `processing` for minutes | A worker died mid-image. It is reclaimed automatically after the 5-minute lease and retried. |
| Callers get `429 RATE_LIMITED` | They exceeded `RATE_LIMIT_PER_MINUTE` for their key; the `Retry-After` header says how long to wait. The mobile app polls a job every 2 seconds, which is well inside the default. |
| Items fail with `source returned status 4xx` | The source URL is wrong or blocks the fetcher; permanent, not retried. 5xx and network errors are retried 3 times. |
| `The URL is not allowed` on creation | The address resolves to a private, loopback or link-local address (SSRF protection). Public addresses only. |
| Webhook never arrives | Check the receiver returns 2xx within 15 s; five attempts, then it stops. Verify the signature as `sha256=HMAC(WEBHOOK_SECRET, "<X-ImageFlow-Timestamp>.<body>")`. |
| Results disappeared | Retention. Raise `RETENTION_HOURS` or fetch results sooner. |
| Startup exits with `bucket ... does not exist` | `STORAGE_BACKEND=s3` points at a bucket that is missing or unreachable; create it or fix the endpoint/credentials. |

## Rotating secrets

- **`WEBHOOK_SECRET`**: change it and redeploy; tell receivers the new value at the same time.
- **Database password**: change it in Postgres and the secret, then redeploy.
- **API keys are identities.** A job belongs to the key that created it (the owner is derived from the key), so replacing
  a key means the new key cannot see the old key's jobs. To rotate without losing access, run both keys in `API_KEYS`
  for as long as results matter (retention is the natural end), then remove the old one. Per-user accounts would remove
  this limitation.

## Before a public launch

Verified in this repository: the images build and run, the stack passes `scripts/smoke.sh`, migrations, the worker,
retention, S3 (against a fake S3 server) and the website (against a real Chromium) are tested.

Needs you:

- A real domain, and `launchpad/application.json` filled in (the website is switched off in it until then).
- AdSense: approval, publisher id, Google's consent message, ad units (see `DEPLOYMENT.md`); a reviewed privacy policy.
- A real S3 bucket tried with one upload and one download, if you use S3 storage.
- The mobile app run on a real device, store listings, real bundle identifiers (they are `com.example.imageflow`), and
  AdMob if you want ads in the app.
- Rotate the old Django secret and database password that sit in this repository's history.

Known gaps (not built):

- **Rate limits are per process.** They stop one key from flooding a replica, but a shared limit across replicas would need
  a shared store; use your ingress or CDN for a global cap. The website does not call the API, so it is not affected.
- **No user accounts**: access is by shared API key.
- **No metrics endpoint or alerting.** Logs and the health endpoints are what exist; add Prometheus or your platform's
  monitoring before relying on it unattended.
- Presigned uploads (a client sending a file instead of a URL) and image formats beyond JPEG and PNG on the server.
