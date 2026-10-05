# Alembic's fileConfig replaces the application's logging

- **Date:** 2026-10-03 (from ai-life-coach)

**What happened.** Running migrations at API startup calls `logging.config.fileConfig` from `alembic/env.py`, which
swaps the root logger's handlers for the ini file's console handler at WARN. After the first migration, the
structured logging pipeline and the INFO level silently disappeared.

**What to do differently.** When you add Alembic, make `env.py` skip `fileConfig` if the app passes the database URL
programmatically (`config.attributes`), and apply it only for command-line use. Check log output after any change to
startup order.
