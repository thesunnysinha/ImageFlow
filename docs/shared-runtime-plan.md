# Shared agent runtime: what goes where

Date: 2026-10-03. Status: **the runtime is built in this template; the other projects have not been migrated.**
Supersedes the earlier "separate `agent-runtime` package" design. That package is not needed (section 2), and its
folder was removed.

## 1. The rule

- **One source of truth: `master-project-template/blueprints/shared/services/agent/`.** New projects get it by
  `python run.py configure`.
- **Logic about HTTP, Postgres, tracing, graph control flow, or generic attacks goes in the runtime.**
  Logic that names a business concept (invoice, dream, document) stays in the project.
- **Each project owns:** its system prompt, its tools, which tools need approval, its domain guardrails, its
  domain models, and its channels (WhatsApp, a chat UI).
- **No new PyPI package, no new repo.** If copying later causes too much drift, publish the same directory as a
  package and depend on a pinned git tag (see ADR 0002 and section 6).

## 2. Is a separate `agent-runtime` needed? No.

| Reason | Detail |
|---|---|
| The template is already where new projects start | It holds the runtime, the project skeleton, and the generator in one repo. |
| A second repo adds cost for little gain | Versioning, tags, a git dependency that Docker builds must reach, cross-repo PRs. |
| The old folder was not the shared package | It was an early snapshot of LedgerLens (no git history, older code, a stray `.env`). Nothing in it is used. |

Revisit only if copied runtime files drift between projects often enough to hurt (section 6).

## 3. What lives where

### master-project-template (the source)

| Piece | Path | State |
|---|---|---|
| `ToolAgent`: LangGraph graph with approval gate, loop guard, duplicate-call suppression, fallback, optional groundedness verifier, in-memory or Postgres checkpointing | `blueprints/shared/services/agent/tool_agent.py` | built, tested |
| `SecurityGuardrail`, `PIIGuardrail`, `LoopGuardrail`, `BaseGuardrail` | `.../guardrails/` | built, tested |
| `AgentServiceError` hierarchy, each error with `http_status` and `error_code` | `.../errors.py` | built, tested |
| structlog + OpenTelemetry + Langfuse, settings-driven | `.../telemetry.py` | built (tracing paths not exercised) |
| Tool-call logging, progress hub | `.../tool_logging.py`, `tool_progress.py` | built; progress hub not wired to an endpoint yet |
| FastAPI blueprint: async chat, `session_id`, `/agent/approval`, guardrails, error handling | `blueprints/backend-fastapi/app/` | built, tested |
| Django blueprint: guardrails on the chat view | `blueprints/backend-django/agents/` | built, minimally verified |
| Original `build_agent_graph` / `invoke_agent` | `.../graph.py` | unchanged |

### revenue-leakage-agent (LedgerLens): keeps the domain, adopts the plumbing

| Stays | Moves to the shared copy |
|---|---|
| Billing tools, `PromptRegistry`, `BillingRepository`, seed data | `guardrails/{base,security,pii,loop_guard}.py` |
| `GroundednessGuardrail` (currency, percent, `LF-`/`ADJ-` IDs); passed to `ToolAgent` as `verifier` | `exceptions.py` (statuses replace the `ERROR_STATUS` dict in `server.py`) |
| `ApprovalPolicyGuardrail`; passed as `approval_policy` | `core/{telemetry,tool_logging,tool_progress}.py` |
| Chat persistence (`chat_models`, `chat_repository`, Alembic), `/billing-data`, `/demo/overview`, `/activity`, `core/auth.py`, `core/migrations.py` | the generic graph in `agents/financial_detective.py` becomes a `ToolAgent` |

Differences to bridge: LedgerLens's error format is `{error_code, message, trace_id, details}` (`X-Trace-ID`),
while the template's is `{success, code, message, data, meta, trace_id}`. LedgerLens keeps its own server and
envelope; only the agent and shared classes are swapped. Its per-request API key maps to `context={"api_key": ...}`.

### ai-life-coach: keeps the coach, adopts the plumbing

| Stays | Adopts |
|---|---|
| `CoachAgent` graph (`guard_input -> gather_context -> coach -> guard_output -> finalize`) because it has no tools or approval | `BaseGuardrail`, `PIIGuardrail`, `SecurityGuardrail` (with its extra patterns passed in), error hierarchy, telemetry |
| Moderation, crisis routing, Redis rate limits and mute, profile/memory/vector store, `ConversationService`, WhatsApp channel and webhook | Optionally the checkpointer lifecycle helper (not yet extracted) |
| History in the checkpointer only (no chat tables) | |

`ToolAgent` is **not** used here. Forcing the coach through it would distort both. Not used: loop guard, approval.

### docpipe: tools only, plus an opt-in backend

- **Stays:** everything. No runtime code moves into docpipe (strict optional-dependency boundaries).
- **Gets, later:** an optional extra `docpipe-sdk[agents]` that adds `agent_backend="runtime"` to `/agents/query`,
  built on `ToolAgent` with the existing `search_documents` / `parse_document` tools as plain LangChain tools,
  plus security/PII guardrails, loop guard, and tracing. Python 3.11+ only for this extra.
- **Default unchanged:** AutoGen stays the default backend; the current LangGraph backend stays. Deprecating
  AutoGen is a separate decision after the runtime backend matches its results.
- **LangChain stays** in docpipe's ingestion, embeddings and vector layers (about 20 files). Not replaced.

### common-services, fastapi-agent-service, django-agent-service

- `common-services`: unaffected.
- `fastapi-agent-service`, `django-agent-service`: stale 28 Sep outputs of the old template (not git repos, no
  tests). Leave alone; regenerate from the template when a real project needs them. Decide later whether to remove.

## 4. Order of work

| # | Step | Gate before starting |
|---|---|---|
| 0 | Put the template under git (`git init`, authored as Sunny Kumar Sinha), commit the runtime work, fix or retire its 2 stale configure tests, correct ADR 0001's stack note | your yes to `git init`; pushing to GitHub needs another yes |
| 1 | LedgerLens: swap shared classes and the graph for the shared copy; all 72 tests plus the approval regression test pass unchanged; PR | LedgerLens `main` is clean (it has 21 uncommitted files) and the double approval write (`append_approval_turn` called twice in `server.py`) is fixed first |
| 2 | ai-life-coach: adopt guardrail base, PII, Security, errors, telemetry; keep the coach graph; all 107 tests pass; docs and CHANGELOG; new decision record superseding 0017 | step 1 merged |
| 3 | docpipe: `[agents]` extra and the `runtime` backend, default behaviour untouched; PR | steps 1 and 2 done |
| 4a | Template deploy skeleton (`configure --deploy`, ADR 0004): done, awaiting a first real deploy | none |
| 4 | Template: Django async agent (optional), progress SSE endpoint, Postgres checkpointing test with a real database | any time |

Each step is one branch and one PR per repo. Nothing is pushed, published or deployed without asking.

## 5. How behaviour stays identical

- Run each project's full suite on `main` first and record the result.
- Move code verbatim; generalise only parameters (service name, patterns, write tools).
- Keep thin re-export modules at old import paths so imports and test patches keep working.
- Keep an adapter with each project's old `execute(...)` signature.
- Compare status codes, error bodies and headers before and after.

## 6. Keeping copies in step

The template copies the runtime into each generated project, so a later fix does not reach projects already
generated. Until that hurts:
- Treat `shared/services/agent/` in a project as vendored: change it in the template first, then re-copy.
- Add `python run.py sync-shared --target <project>` to diff and copy it (not built yet).
- If projects start to diverge, publish the directory as a package and pin a tag in each project.

## 7. Known gaps and risks

- LedgerLens `main` has 21 uncommitted files and the double approval write.
- Template: not under git; 2 of its 3 original tests fail (they assert files the generator does not produce).
- Postgres checkpointing, Langfuse and OpenTelemetry paths are written but not exercised in tests.
- Django cannot use `ToolAgent` (sync views) until an async path exists.
- The template and LedgerLens use different response envelopes.
- The Python floor differs: template CLI 3.9, LedgerLens 3.11, ai-life-coach 3.12, docpipe 3.10.
