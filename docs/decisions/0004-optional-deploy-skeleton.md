# ADR 0004: Optional deployment skeleton for generated projects

- **Status:** Accepted
- **Date:** 2026-10-03

## Context

Projects on the shared k3s cluster (ai-life-coach, LedgerLens, jingo) each hand-wrote the same deployment files:
manifests, a build-and-deploy workflow around `vm_tool deploy-k8s`, a secrets validator and a Dockerfile. The copies
drift, and a new project needs a day of copying. common-services (Launchpad) keeps a registry of applications in
`applications/<id>.json` for DNS, domains and inventory.

## Decision

- `configure --deploy` generates those files from `blueprints/deploy/` for the FastAPI backend only. The default
  output is unchanged, so existing flows do not break. Django is refused with a clear message because its image
  runs the development server.
- The generated manifests follow ai-life-coach's tested shape (namespace and quota, ExternalName alias to the
  shared Postgres, one Deployment with probes and a non-root user, Service, Ingress with cert-manager) and use the
  same secret names (`backend-secret`, `db-secret`, `ghcr-secret`).
- Deploys stay opt-in: the workflow skips until the repository variable `K8S_DEPLOY` is `true`.
- Launchpad registration is a generated file, not an action. It is `provisioningEnabled: false`, `managed: false`,
  matching how existing apps are registered, because the project's own workflow applies its manifests. Copying it
  into common-services is a manual pull request.
- Database and DNS creation stay documented manual steps in `DEPLOYMENT.md`; the generator never touches the
  cluster, DNS or GitHub settings.
- Redis and the frontend are not part of the skeleton; the guide says how to add them.

## Consequences

A new project gets a working deploy path by passing two flags and following one checklist. Fixes to the deploy
files reach only newly generated projects, as with the shared agent runtime. The image build and the Postgres
checkpoint path were not exercised when this was written (no Docker or Postgres available); the first real deploy
is the test, which is why the resource limits and probes are expected to need tuning.
