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
| Secret names | `API_KEYS` (values are never stored in the manifest) |

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
- Image fetching and processing are not implemented yet: jobs are accepted and stored as `queued`.
