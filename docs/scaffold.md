# Scaffold workflow

`.github/workflows/scaffold.yml` creates a project from this template, or adds the template's files to a repository that
already exists. It is what a Launchpad "new application" wizard dispatches, and you can run it yourself from the Actions
tab. The logic lives in `scripts/scaffold.py`, which you can also run locally.

## Inputs

| Input | Meaning |
|---|---|
| `mode` | `create`: make a new repository (private unless `visibility` says otherwise). `populate`: open a pull request in an existing repository. |
| `repository` | `owner/name` of the target. `create` refuses to run if it already exists. |
| `name` | Project name: lowercase letters, digits and dashes. Becomes the namespace and image name. |
| `domain` | Public host name; required with a deploy mode. |
| `backend` | `fastapi` (default), `django`, `gin`, `nodejs` or `none` (frontend only). Django, Gin and Node.js deploy through `launchpad` mode only and take no features. |
| `frontend` | `react-mui` (Vite, the default), `nextjs` or `none`. Next.js reaches the API through a rewrite to `API_ORIGIN`, set in the Launchpad manifest. `none` generates an API-only project: no `services/frontend`, no frontend CI job, `frontend.enabled: false` in the manifest. |
| `database` | `postgresql` for backend blueprints; `none` for frontend-only projects. Select `backend=none`, `frontend=nextjs` or `react-mui`, and `database=none` together. No database is provisioned for that contract. |
| `features` | Comma-separated feature ids (`python run.py features --json` lists them). Dependencies are added for you. |
| `deploy_mode` | `none`, `launchpad` (a managed Launchpad manifest) or `vm_tool` (own manifests and workflow). |

Frontend-only generation rejects backend features and supports `deploy_mode=none` or `launchpad`. Launchpad manifests set `backend.enabled=false`, `services.postgres.enabled=false`, `services.redis.enabled=false`, and `frontend.enabled=true`, with no backend secrets or API origin. The public domain is assigned to the frontend. Existing backend blueprints require PostgreSQL and reject `database=none`.

```bash
python scripts/scaffold.py --mode create --repository acme/my-site --name my-site --backend none --frontend nextjs --database none --deploy-mode launchpad --domain my-site.example.com --dry-run
```

## What it does

1. Validates every input, then runs `python run.py configure` with them, so all generator rules apply.
2. Publishes only what the generated project does not ignore: the real `.env` files, the secrets override and the
   configure backups are never committed.
3. `create`: commits (as `SCAFFOLD_GIT_NAME` / `SCAFFOLD_GIT_EMAIL`) and runs `gh repo create ... --push`.
4. `populate`: clones the repository, creates the branch `scaffold/<name>-<run id>`, copies **only files that do not
   exist yet** (an existing file is never changed), pushes, and opens a pull request that lists what was added and what
   was skipped. If nothing is new it does nothing. An empty repository gets the files as its first commit.

## One-time setup in this repository

Nothing is configured for you, and no secret is stored in Git.

1. **Secret `SCAFFOLD_TOKEN`**: a token that can create repositories, push (including `.github/workflows/*`) and open
   pull requests in the target repositories. Two options:
   - A fine-grained personal access token limited to your repositories with *Administration: write*, *Contents: write*,
     *Pull requests: write* and *Workflows: write*. GitHub's rules for what fine-grained tokens may create in a personal
     account have changed over time, so check that `create` works with it before relying on it.
   - A classic token with the `repo` and `workflow` scopes, which is known to work, but is broader.
2. **Variables `SCAFFOLD_GIT_NAME` and `SCAFFOLD_GIT_EMAIL`**: the author of the generated commits. The workflow stops
   with a clear message if the secret or the variables are missing.

## Dispatching it from Launchpad

Launchpad already dispatches workflows with its GitHub App. For this it needs *Actions: write* on this repository
and a `POST /repos/<owner>/master-project-template/actions/workflows/scaffold.yml/dispatches` call with the inputs
above. The workflow and CLI database values are `postgresql|none`: translate Launchpad's `request.database=postgres` to `postgresql`, and pass `none` through unchanged. For `backend=none`, send the public frontend host (`request.frontendDomain`) as the workflow `domain` input. With a backend, send the backend host. The CLI requires that selected host for `deploy_mode=launchpad`; local generation (`deploy_mode=none`) needs no domain.

Launchpad never sees `SCAFFOLD_TOKEN`; it stays in this repository's Actions secrets. After the repository
exists, Launchpad's existing flow registers it (see `launchpad/application.json` in the generated project).

## Not verified

The pull-request mode is tested against local git repositories (new files only, existing files untouched, empty
repositories, repeat runs). `gh repo create`, `gh pr create`, the token permissions and the Launchpad dispatch have not
been exercised against GitHub.
