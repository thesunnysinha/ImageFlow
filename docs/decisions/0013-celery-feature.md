# ADR 0013: A Celery feature that declares its workers

- **Status:** Accepted
- **Date:** 2026-10-04

## Context

Projects need background work (retries, scheduled jobs, slow calls). Launchpad (common-services) can now run extra
Deployments from the backend image, declared as `workers` in the application manifest, and requires a broker.

## Decision

- `celery` is an optional FastAPI feature that requires `redis`. It adds `app/celery_app.py` (broker from
  `CELERY_BROKER_URL`, else `REDIS_URL`; results are not stored; tasks are acknowledged late with a prefetch of one),
  `app/tasks.py` with a `ping` task, and tests.
- A feature may declare `workers` in `feature.json`. The same declaration drives local Compose (one container per
  worker, built from the backend with its env file and database and Redis dependencies) and the Launchpad manifest
  (`workers`). Two features may not declare the same worker name.
- Two workers: `worker` (250m CPU, 256Mi, concurrency 2) and `beat` (100m, 128Mi). Both sit inside Launchpad's worker
  limits and leave `worker` room to scale. `beat` is a singleton: Launchpad refuses more than one replica.
- `vm_tool` mode refuses the feature: its manifests have no worker Deployments.
- No health probe: a Celery process has no HTTP endpoint.

## Consequences

- The app's Redis runs with `allkeys-lru` and a 128 MB cap, so queued tasks can be evicted under memory pressure.
  `DEPLOYMENT.md` says so; work that must not be lost belongs in Postgres. A different broker means setting
  `CELERY_BROKER_URL` (which also satisfies Launchpad's broker rule).
- The generated manifest only validates against a common-services that has the `workers` schema.
- Local Compose starts beat with the worker; stopping one container does not stop the other.
