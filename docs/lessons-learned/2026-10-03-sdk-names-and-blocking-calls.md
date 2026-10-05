# SDK category names differ from API names; keep blocking calls off the event loop

- **Date:** 2026-10-03 (from ai-life-coach)

**What happened.** OpenAI's Python SDK exposes moderation categories as `self_harm_intent`, while the API and docs say
`self-harm/intent`. Comparing against the wrong spelling would have silently skipped the crisis route. Separately, an
`async def` webhook called blocking LLM and database code and stalled the event loop.

**What to do differently.** Normalise external names in one function and test safety-critical branches with the
real spellings (the `moderation` feature does, with a test). Treat any synchronous call inside an async route as
suspect.
