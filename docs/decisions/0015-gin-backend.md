# ADR 0015: A Go (Gin) backend with the same HTTP contract

- **Status:** Accepted
- **Date:** 2026-10-05

## Context

Backends were FastAPI, Django and Node.js. A Go option was requested for services that want a small static binary,
cheap concurrency and a typed API, for example image or batch processing APIs. Like Node.js, Go cannot reuse the
Python shared agent runtime.

## Decision

- `blueprints/backend-gin` is a Gin service (Go 1.25, pgx) that keeps the contract the frontends and Launchpad rely on:
  `GET /api/v1/health` (no dependencies, answers any Host), `GET /api/v1/ready` (database), `POST /api/v1/agent/chat`
  with the same request and response fields, the envelope `{success, code, message, data, meta, trace_id}` and
  `X-Request-ID`. Unknown routes, wrong methods, validation errors and panics all use the envelope.
- Layout is Go-idiomatic and dependency-injected: `cmd/server` (wiring, graceful shutdown, `-healthcheck` flag),
  `internal/{config,envelope,httpapi,model,store}`. `store.Store` and `model.Model` are interfaces so handlers are tested
  without Postgres or a model provider; the OpenAI call uses `net/http`, with `OPENAI_BASE_URL` for tests.
- The agent is one model call with persisted history (`chat_messages`, created at start and retried until Postgres
  answers). It has **no tools, approval flow or guardrails**, same as Node.js.
- The image is a static binary on distroless, UID 10001; the container HEALTHCHECK runs the binary itself because the
  base has no shell.
- Like Django and Node.js, Gin is accepted in `launchpad` mode only and takes no optional features.
  `blueprints/deploy/backends/gin/` overlays a Go CI workflow (gofmt, vet, race tests, build).
- Routing is Gin rather than `net/http` because Gin is the most widely used Go framework; handlers keep business logic
  out of `*gin.Context` so a different router could replace it.

## Consequences

- Verified: `go vet` and `go test` pass in the blueprint; generation tests cover files, manifest, workflow, notes and
  the Launchpad-only rule.
- Not verified: the image build and boot against Postgres, a real Launchpad deployment, and any model call.
