# ADR 0005: Two deployment modes, vm_tool and Launchpad

- **Status:** Accepted
- **Date:** 2026-10-03
- **Amends:** [0004](0004-optional-deploy-skeleton.md)

## Context

ADR 0004 generated Kubernetes manifests and a `vm_tool` workflow, copied from ai-life-coach. Reading common-services
shows there are two ways an app connects to the platform. Legacy apps keep their own manifests and reach the shared
Postgres and Redis through `ExternalName` aliases. Apps managed by Launchpad are described by one manifest
(`applications/<id>.json`); Terraform and the *Application platform* workflow create the `apps-<id>` namespace, a
dedicated database role delivered as Secret `<id>-database` (`DATABASE_URL`, `POSTGRES_*`), an optional dedicated
Redis (`REDIS_URL`) or MinIO bucket, the Deployment, ingress, DNS and Vercel project, and build the image from the
manifest's Dockerfile path. A project made through a Launchpad wizard should not carry deployment files of its own.

## Decision

- `configure --deploy-mode` takes `vm_tool` (previous behaviour; `--deploy` stays as its shorthand) or `launchpad`.
- `launchpad` generates only `launchpad/application.json` with `managed: true` for the backend and database,
  `namespace: apps-<name>` (prefix configurable), `secretNames: ["OPENAI_API_KEY"]`, the Dockerfile and health path of
  the FastAPI blueprint, and `AGENT_STORE=postgres` so the app uses the injected `DATABASE_URL`; plus a Launchpad
  specific `DEPLOYMENT.md` and the shared quality workflow. Redis, storage and the Vercel frontend are off by default.
- The generator never touches common-services, GitHub settings, DNS or the cluster. Registering the app remains a
  pull request to common-services (or Launchpad's *New application*), which the guide describes.
- Both modes are validated in tests against common-services' `application.schema.json` when a checkout is available.

## Consequences

New projects can be provisioned entirely by Launchpad, which is the basis for a create-from-template wizard. Known
gaps: common-services' registry script does not enforce the `apps-` namespace prefix (the Terraform runtime stack
rejects other prefixes at plan time), so the prefix default matters; the managed flow was validated only against
the schema and registry script, not by provisioning a real app; the `vm_tool` mode remains for apps that keep their
own manifests.
