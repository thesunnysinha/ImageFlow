# ADR 0008: A scaffold workflow in the template, with the token kept there

- **Status:** Accepted
- **Date:** 2026-10-03

## Context

A Launchpad "new application" wizard should create a repository, or link an existing one, and populate it from the
template. Launchpad's GitHub App deliberately has no Administration or Workflows write permission, and a personal
account's repositories cannot be created by an App installation token at all. Widening an internet-facing Next.js
console with those powers would be a poor trade.

## Decision

- The template repository owns the capability: `scaffold.yml` is dispatched with the choices, and `scripts/scaffold.py`
  does the work with a token stored only as this repository's Actions secret `SCAFFOLD_TOKEN`.
- Launchpad only dispatches the workflow (a permission it already uses elsewhere) and then continues with its existing
  registration flow.
- `create` never touches an existing repository. `populate` goes through a pull request, copies only files that do not
  exist, and never changes an existing file, matching the generator's own refusal to overwrite.
- Inputs reach the script through environment variables, never inside `run:` text, and the workflow has read-only
  repository permissions. The commit identity comes from repository variables, not from code.
- The generated project is published according to its own `.gitignore`, so real `.env` files, the secrets override and
  configure backups are never committed.

## Consequences

Repo creation and file population work without giving the console extra GitHub permissions. The cost is one
long-lived token in this repository's secrets, which the owner must create and rotate. The logic is tested against
local git repositories; the GitHub calls themselves (`gh repo create`, `gh pr create`), the token's exact permissions
and the dispatch from Launchpad still have to be proven on a first real run.
