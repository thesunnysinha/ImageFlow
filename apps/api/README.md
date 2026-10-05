# ImageFlow API (v2, Go + Gin)

Phase 1 skeleton: job creation/status, Postgres schema (goose), API-key auth,
SSRF-checked URLs, health/readiness probes.

    docker compose up --build
    curl -s -X POST localhost:8080/v1/jobs -H 'Authorization: Bearer dev-key-change-me' \
      -H 'Content-Type: application/json' -d '{"source_urls":["https://example.com/a.jpg"]}'

Config (env): `DATABASE_URL`, `API_KEYS` (required); `ADDR`, `MAX_ITEMS_PER_JOB`, `APP_ENV`.
Test: `go test -race ./...`
