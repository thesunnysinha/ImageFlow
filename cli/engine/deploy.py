"""Optional deployment files for a generated project, in one of two modes.

``vm_tool``: the project carries its own Kubernetes manifests and a workflow that runs
``vm_tool deploy-k8s``. ``launchpad``: the project carries only a Launchpad manifest, and
Launchpad (common-services) provisions and deploys it.
"""

from __future__ import annotations

import json
import re
from pathlib import Path
from typing import Dict, Optional

from cli.engine.configuration import ConfigurationError

SLUG_PATTERN = re.compile(r"^[a-z][a-z0-9-]{1,29}[a-z0-9]$")
DOMAIN_PATTERN = re.compile(r"^(?=.{4,253}$)([a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z]{2,}$")
REPOSITORY_PATTERN = re.compile(r"^[A-Za-z0-9][A-Za-z0-9-]*/[A-Za-z0-9._-]+$")
PREFIX_PATTERN = re.compile(r"^[a-z][a-z0-9-]{0,20}-$")
SUPPORTED_BACKENDS = {"fastapi", "django", "gin", "nodejs", "none"}
MODES = ("vm_tool", "launchpad")
DEFAULT_NAMESPACE_PREFIX = "apps-"  # common-services platform/config.json: vps.namespacePrefix


def slugify(value: str) -> str:
    """Lower-case, dash-separated name usable as a Kubernetes namespace and image name."""
    return re.sub(r"-+", "-", re.sub(r"[^a-z0-9]+", "-", value.lower())).strip("-")


def validate_deploy_options(
    backend: str,
    mode: str,
    slug: str,
    domain: Optional[str],
    repository: Optional[str],
    namespace_prefix: str = DEFAULT_NAMESPACE_PREFIX,
) -> None:
    if mode not in MODES:
        raise ConfigurationError("--deploy-mode must be one of: {}".format(", ".join(MODES)))
    if backend not in SUPPORTED_BACKENDS or (backend != "fastapi" and mode != "launchpad"):
        raise ConfigurationError(
            "Deployment files are available for the fastapi backend in both modes and for the django, gin and nodejs "
            "backends and frontend-only projects in launchpad mode only (the vm_tool manifests are written for "
            "fastapi); got {} with {}.".format(backend, mode)
        )
    if not SLUG_PATTERN.match(slug):
        raise ConfigurationError(
            "Project name '{}' must be 3-31 characters of lowercase letters, digits and dashes (starting with a "
            "letter); pass --name.".format(slug)
        )
    if not domain or not DOMAIN_PATTERN.match(domain):
        raise ConfigurationError("Deployment files need --domain with a valid host name such as app.example.com")
    if repository and not REPOSITORY_PATTERN.match(repository):
        raise ConfigurationError("--repository must look like owner/name")
    if mode == "launchpad" and not PREFIX_PATTERN.match(namespace_prefix):
        raise ConfigurationError("--namespace-prefix must be lowercase letters, digits and dashes ending in '-'")


def render_deploy_files(
    template_root: Path,
    mode: str,
    slug: str,
    name: str,
    domain: str,
    repository: Optional[str],
    namespace_prefix: str = DEFAULT_NAMESPACE_PREFIX,
    backend: str = "fastapi",
    frontend: str = "react-mui",
) -> Dict[str, str]:
    """Return the deployment files for ``mode`` with the project's names filled in.

    A folder ``blueprints/deploy/backends/<backend>`` overlays the common and mode files (for example a CI workflow
    for a different language). With ``frontend == "none"`` the CI workflow loses its frontend job.
    """
    tokens = {
        "[[SLUG]]": slug,
        "[[NAME]]": name,
        "[[DOMAIN]]": domain,
        "[[REPOSITORY]]": repository or "OWNER/{}".format(slug),
        "[[DB_NAME]]": slug.replace("-", "_"),
        "[[NAMESPACE]]": (namespace_prefix + slug) if mode == "launchpad" else slug,
    }
    rendered: Dict[str, str] = {}
    for source in (
        template_root / "blueprints" / "deploy" / "modes" / "common",
        template_root / "blueprints" / "deploy" / "modes" / mode,
        template_root / "blueprints" / "deploy" / "backends" / backend,
    ):
        if not source.is_dir():
            continue
        for path in sorted(p for p in source.rglob("*") if p.is_file()):
            content = path.read_text(encoding="utf-8")
            for token, value in tokens.items():
                content = content.replace(token, value)
            leftover = re.findall(r"\[\[[A-Z_]+\]\]", content)
            if leftover:
                raise ConfigurationError("Unfilled placeholder {} in {}".format(leftover[0], path.relative_to(source)))
            rendered[str(path.relative_to(source))] = content
    if frontend == "none":
        rendered = {name: _without_frontend_job(content) if name.startswith(".github/workflows/") else content
                    for name, content in rendered.items()}
    if backend == "none":
        rendered = {name: _without_backend_job(content) if name.startswith(".github/workflows/") else content
                    for name, content in rendered.items()}
    return rendered


def _without_frontend_job(workflow: str) -> str:
    """Remove the top-level ``frontend:`` job (its indented body runs to the next job or the end of the file)."""
    return re.sub(r"\n  frontend:\n(?:(?:    .*)?\n)*", "\n", workflow).rstrip("\n") + "\n"


def _without_backend_job(workflow: str) -> str:
    """Remove the top-level backend job from frontend-only quality workflows."""
    return re.sub(r"\n  backend:\n(?:(?:    .*)?\n)*", "\n", workflow).rstrip("\n") + "\n"


FRONTEND_FRAMEWORKS = {"react-mui": "vite", "nextjs": "nextjs"}


def apply_frontend_to_launchpad_manifest(manifest_json: str, frontend: str, domain: str) -> str:
    """Record the frontend's build framework (none: the frontend stays disabled); Next.js also gets the backend origin its rewrites forward to."""
    entry = json.loads(manifest_json)
    if frontend == "none":
        entry["frontend"]["enabled"] = False
        return json.dumps(entry, indent=2) + "\n"
    entry["frontend"]["framework"] = FRONTEND_FRAMEWORKS[frontend]
    if frontend == "nextjs":
        entry["environment"]["frontend"]["API_ORIGIN"] = "https://" + domain
    return json.dumps(entry, indent=2) + "\n"


def apply_backend_to_launchpad_manifest(manifest_json: str, backend: str, domain: str) -> str:
    """Django, Gin and Node.js read different settings and secrets than FastAPI: swap the backend environment and secret names."""
    if backend == "none":
        entry = json.loads(manifest_json)
        entry["backend"]["enabled"] = False
        entry["backend"]["domain"] = None
        entry["frontend"]["enabled"] = True
        entry["frontend"]["domain"] = domain
        entry["services"]["postgres"]["enabled"] = False
        entry["services"]["redis"]["enabled"] = False
        entry["environment"]["backend"] = {}
        entry["environment"]["frontend"].pop("API_ORIGIN", None)
        entry["secretNames"] = []
        entry["notes"] = "Generated frontend-only project for Launchpad; deploy services/frontend without backend or database provisioning."
        return json.dumps(entry, indent=2) + "\n"
    if backend == "nodejs":
        entry = json.loads(manifest_json)
        entry["environment"]["backend"] = {"NODE_ENV": "production"}
        return json.dumps(entry, indent=2) + "\n"
    if backend == "gin":
        entry = json.loads(manifest_json)
        entry["environment"]["backend"] = {"ENVIRONMENT": "production"}
        return json.dumps(entry, indent=2) + "\n"
    if backend != "django":
        return manifest_json
    entry = json.loads(manifest_json)
    entry["environment"]["backend"] = {"DJANGO_DEBUG": "false", "DJANGO_ALLOWED_HOSTS": domain}
    entry["secretNames"] = ["DJANGO_SECRET_KEY"] + [name for name in entry["secretNames"] if name != "DJANGO_SECRET_KEY"]
    return json.dumps(entry, indent=2) + "\n"
