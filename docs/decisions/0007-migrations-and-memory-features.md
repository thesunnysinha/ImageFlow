# ADR 0007: Migrations and memory as features

- **Status:** Accepted
- **Date:** 2026-10-03
- **Extends:** [0006](0006-optional-features.md)

## Context

ai-life-coach keeps per-user memory in pgvector and applies Alembic migrations at startup. The FastAPI blueprint had
neither a database layer nor a migration story, so these two were left out of the first set of features. A real
database is also needed to check the Postgres agent store, which had only been written, not run.

## Decision

- `migrations` is a feature of its own: `alembic.ini`, an `env.py` that takes the URL from `DATABASE_URL` (or
  `POSTGRES_*`) and skips `logging.config.fileConfig` when the application passes the URL, `app/db.py` (declarative
  `Base`, lazy async sessions), and a startup hook with `ORDER = 1` that applies migrations before anything else.
- Migrations run with `alembic upgrade heads`. A feature ships its migration as a separate branch
  (`branch_labels`, no `down_revision`), so it applies independently of the project's own revisions.
- `memory` requires `migrations`. It keeps LangChain's table names (`langchain_pg_collection`,
  `langchain_pg_embedding`) so data from such a store, such as ai-life-coach's, needs no schema change. The vector size is
  `MEMORY_EMBEDDING_DIM` (default 1536) and cannot change once the table exists. The agent gets a `search_memory` tool
  (the feature contributes tools through a new `tools()` hook) that reads the current user from a context variable set
  in `before_chat`; user messages are stored in `after_chat`, and storage or recall failures never break a chat.
- A feature may declare `postgres_extensions`. Local Compose then uses the `pgvector/pgvector:pg16` image, the Launchpad
  manifest lists the extension for provisioning, and the migration only runs `CREATE EXTENSION` if it is missing,
  because a managed role may not be allowed to. `vm_tool` mode refuses such features.
- Tests that need a real database use `TEST_DATABASE_URL` and skip without it. The generated CI starts a pgvector
  Postgres for them. The template tests it with an embedded Postgres from the `pgserver` package, so no Docker is needed.

## Consequences

Projects can add memory with `--features memory`, and ai-life-coach can move its memory store onto the same code.
Costs: the embedding model and vector size are tied together by the project, only user messages are remembered by
default (change `app/features/memory.py` to store replies too), and memory is a copy like every other feature.
Not done: a migration story for projects that add other features' tables later is just standard Alembic, not wrapped.
