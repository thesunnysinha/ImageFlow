# Process-local chat history breaks with restarts and replicas

- **Date:** 2026-10-03 (from ai-life-coach)

**What happened.** The first agent kept short-term chat history in a module-level dict (and used a fresh session id
per call, so memory did not work at all). A restart, or a second replica, meant the agent forgot the conversation.

**Root cause.** State lived in the process instead of a shared store.

**What to do differently.** Anything a user would notice losing belongs in Postgres or Redis, never in a Python
dict. The shared `ToolAgent` keeps conversations in memory by default (fine for development); set
`AGENT_STORE=postgres` before running more than one replica or caring about restarts.
