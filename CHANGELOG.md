# Changelog

## Unreleased

- Remove the superseded prototype `apps/api`, and from `legacy/` the committed `.env` files (database password; rotate it
  and the Django `SECRET_KEY`), the vendored `staticfiles/` and a sample CSV. All remain in git history.
- Restructure to the master-project-template layout (`services/backend`, `env/`, `launchpad/`, `docker-compose.yml`,
  `project.config.yml`, quality workflow) with the Go (Gin) backend blueprint. The Django/Celery implementation moved to
  `legacy/`.
- Go API: `POST /api/v1/jobs`, `GET /api/v1/jobs/{id}`, versioned SQL migrations applied at start under an advisory lock,
  API-key auth with per-key job ownership, SSRF checks on source and webhook URLs, response envelope and `X-Request-ID`.
  Database tests run when `TEST_DATABASE_URL` is set.
- Image processing, queue, storage, user accounts and the mobile app are not built yet.
