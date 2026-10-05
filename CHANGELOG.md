# Changelog

## Unreleased

- Mobile app (`services/mobile`, Expo SDK 57, React Native, TypeScript, Expo Router, TanStack Query): server address and
  API key kept in the device keystore and verified with a real request before saving; paged job list that refreshes while
  jobs run; new job from pasted URLs with validation and optional webhook; job screen that polls until the job finishes
  and shows compressed images (fetched with the API key) with the bytes saved. Typed API client and pure helpers are unit
  tested; CI typechecks, tests and bundles for Android and iOS.
- API: `GET /api/v1/jobs` lists the caller's jobs newest first, with per-job counts and a `before` cursor.
- Worker: fetches images (size and pixel limits, SSRF-safe client that checks the address at connect time), shrinks and
  re-encodes JPEG and PNG, stores results (local disk behind a `Storage` interface), retries transient failures with
  backoff, recovers items from crashed workers after a lease, and sends HMAC-signed, retried webhooks when a job
  finishes. The queue is Postgres (`FOR UPDATE SKIP LOCKED`), so replicas share the work. New endpoint
  `GET /api/v1/jobs/{id}/items/{position}/output`. Migration `0002_worker.sql`. `WEBHOOK_SECRET` is required while the
  worker runs (`RUN_WORKER=false` disables it).
- Remove the superseded prototype `apps/api`, and from `legacy/` the committed `.env` files (database password; rotate it
  and the Django `SECRET_KEY`), the vendored `staticfiles/` and a sample CSV. All remain in git history.
- Restructure to the master-project-template layout (`services/backend`, `env/`, `launchpad/`, `docker-compose.yml`,
  `project.config.yml`, quality workflow) with the Go (Gin) backend blueprint. The Django/Celery implementation moved to
  `legacy/`.
- Go API: `POST /api/v1/jobs`, `GET /api/v1/jobs/{id}`, versioned SQL migrations applied at start under an advisory lock,
  API-key auth with per-key job ownership, SSRF checks on source and webhook URLs, response envelope and `X-Request-ID`.
  Database tests run when `TEST_DATABASE_URL` is set.
- User accounts, push notifications and presigned uploads are not built yet.
