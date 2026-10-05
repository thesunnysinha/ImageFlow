# Deploying imageflow with Launchpad

Launchpad (in the common-services repository) provisions and deploys this app from one manifest,
`launchpad/application.json`. There is no `k8s/` folder and no deploy workflow here on purpose: Launchpad builds the
image from `services/backend/Dockerfile`, creates the namespace, database role, Deployment, Service, ingress, TLS and
DNS, and keeps the result in Git. The backend will be served at **https://imageflow.example.com**.

## What the manifest asks for

| Item | Value |
|---|---|
| Application id / namespace | `imageflow` / `apps-imageflow` |
| Backend | `services/backend/Dockerfile`, port 8000, health `/api/v1/health` |
| Database | `imageflow`, restricted login `imageflow_app`, delivered as Secret `imageflow-database` |
| Redis / storage / frontend | off (set `services.redis.enabled` or `services.storage.enabled`, or enable the Vercel frontend, when you need them) |
| Plain environment | `ENVIRONMENT` |
| Secret names | `API_KEYS`, `WEBHOOK_SECRET` (values are never stored in the manifest) |

The backend reads `DATABASE_URL` and `POSTGRES_*` from the database Secret and applies its SQL migrations at start.

## Steps

1. **Let Launchpad see the repository.** Install the Launchpad GitHub App on `thesunnysinha/ImageFlow` (selected repositories).
2. **Register the app.** Either choose *New application* in Launchpad and pick this repository, or copy
   `launchpad/application.json` to `applications/imageflow.json` in common-services and open a pull request. Review the
   validation checks and the provisioning plan before merging; destructive changes are refused.
3. **Provide the secret value.** Add `API_KEYS` (comma-separated keys clients send as `Authorization: Bearer <key>`) for `imageflow` in Launchpad's backend secrets (it is read from
   the `APPLICATION_SECRETS_JSON` map; see common-services `docs/platform-console.md`). Never commit it.
4. **Merge and watch.** Merging starts planning and provisioning; follow *Activity* in Launchpad.

## Check

```bash
ssh Hostinger "kubectl -n apps-imageflow get pods"
curl https://imageflow.example.com/api/v1/health
```

## Notes

- Tune CPU and memory in the manifest after the first run; Launchpad applies them as the container limits.
- The *Quality checks* workflow in this repo still runs lint, tests and the changelog check on every pull request.
- Prefer the `vm_tool` deploy mode (`configure --deploy-mode vm_tool`) only for apps that keep their own manifests.

## Go (Gin) specifics

- The image is a static Go binary on a distroless base, run as UID 10001 on port 8000. `GET /api/v1/health` never
  touches the database; `GET /api/v1/ready` does. The container HEALTHCHECK runs `/server -healthcheck`.
- Versioned SQL migrations (`services/backend/migrations/*.sql`) are applied at start under an advisory lock, so several
  replicas can start together; they are retried until Postgres answers.
- Authentication is API-key based for now (`API_KEYS`); jobs are visible only to the key that created them.
- The worker runs inside the API process (set `RUN_WORKER=false` on API-only replicas). It claims images with
  `FOR UPDATE SKIP LOCKED`, so several replicas can share one database. Failed fetches are retried with backoff (3
  attempts); a worker that dies mid-image is recovered after a 5-minute lease.
- Webhooks are signed: header `X-ImageFlow-Signature` is `sha256=` + HMAC-SHA256 of `<X-ImageFlow-Timestamp>.<body>` with
  `WEBHOOK_SECRET`. They are retried up to 5 times and never follow redirects.
- **Storage** is local disk by default (`STORAGE_DIR`, default `/data/images`): fine for one replica with a persistent
  volume. For several replicas, ephemeral disks or direct downloads, switch to any S3-compatible service (AWS S3,
  Cloudflare R2, MinIO, Backblaze B2): set `STORAGE_BACKEND=s3`, `S3_ENDPOINT` (host[:port], no scheme), `S3_BUCKET`,
  `S3_REGION`, `S3_USE_SSL` and optionally `S3_PREFIX`, and add `S3_ACCESS_KEY` and `S3_SECRET_KEY` to
  `secretNames` in `launchpad/application.json`. The bucket must already exist; the service refuses to start otherwise.
  With S3, `GET /api/v1/jobs/{id}/items/{position}/output-url` returns a 5-minute presigned download URL.
