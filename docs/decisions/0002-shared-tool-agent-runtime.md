# ADR 0002: Shared tool-agent runtime in the template

- **Status:** Accepted
- **Date:** 2026-10-03

## Context

The revenue-leakage agent and the life-coach backend carry near-identical copies of telemetry, guardrails, the
exception hierarchy and the tool-agent graph; fixes drifted between copies. The template's shared agent was a
44-line loop with no guards, persistence, approval or tracing, and each backend repeated the same adapter.
ADR 0001 describes Django as the only supported stack; FastAPI has since become the default, recorded in
[0003](0003-fastapi-default-backend.md).

## Decision

- Put the reusable runtime in `blueprints/shared/services/agent/` and let `configure` copy it into generated
  projects, as it already does for the older graph. Generated projects depend on no extra package.
- Keep it framework-neutral (no FastAPI or Django imports). The FastAPI blueprint uses the async `ToolAgent`;
  the Django blueprint, whose views are synchronous, uses only the guardrails and errors for now.
- Tools that need a person's approval are declared in the app's `tool_registry.APPROVAL_TOOLS`.
- Keep the template's response envelope; the runtime errors carry an `http_status` and `error_code` and each
  backend formats them.
- Depend on `langgraph` and `langchain-core`; the full `langchain` package is not required.

## Consequences

New projects get approval, loop protection, guardrails and tracing by writing a prompt and tools. Projects already
generated do not receive later fixes unless the shared files are re-copied. If that drift becomes a cost, publish
the directory as an installable package and depend on a pinned tag instead.

Supersedes nothing.
