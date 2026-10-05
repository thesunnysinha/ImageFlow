"""Deterministic project file rendering."""

from __future__ import annotations

import json
import re
from pathlib import Path
from typing import Dict, Mapping



def _dotenv(values: Mapping[str, str]) -> str:
    """Render simple environment values in stable key order."""
    return "".join("{}={}\n".format(key, values[key]) for key in sorted(values))


NO_FRONTEND_PAGE = """    location / {
        default_type text/plain;
        return 200 "API only: this project has no frontend. The API is under /api/v1.\\n";
    }
"""


def _without_frontend_compose(compose: str) -> str:
    """Drop the frontend service and the proxy's dependency on it."""
    stripped = re.sub(r"  frontend:\n(?:    .*\n)+", "", compose)
    stripped = stripped.replace("      - frontend\n", "")
    if "frontend" in stripped:
        raise ValueError("docker-compose template still mentions the frontend after removing it")
    return stripped


def _without_frontend_nginx(nginx: str) -> str:
    """Answer ``/`` with a plain note instead of proxying to the missing frontend."""
    replaced = re.sub(r"    location / \{\n(?:        .*\n)+    \}\n", lambda _match: NO_FRONTEND_PAGE, nginx)
    if "http://frontend" in replaced:
        raise ValueError("nginx template still mentions the frontend after removing it")
    return replaced


def _without_backend_compose(compose: str) -> str:
    """Drop the backend and database services from a frontend-only Compose file."""
    for service in ("database", "backend"):
        compose = re.sub(r"^  {}:\n(?:^    .*\n)+".format(service), "", compose, flags=re.MULTILINE)
    compose = compose.replace("      - backend\n", "")
    compose = re.sub(r"(^  frontend:\n(?:^    (?!environment:).*(?:\n|$))*)^    environment:\n(?:^      .*\n)+", r"\1", compose, flags=re.MULTILINE)
    compose = re.sub(r"volumes:\n  postgres-data:\n?", "", compose)
    if re.search(r"(?:^|[./_-])(backend|database|postgres)(?:$|[./_:-])", compose, re.MULTILINE | re.IGNORECASE):
        raise ValueError("docker-compose template still mentions backend or database after removing them")
    return compose


def _without_backend_nginx(nginx: str) -> str:
    """Remove routes that proxy to a backend which a frontend-only project does not have."""
    replaced = re.sub(
        r"\n    location (?:/api/|= /docs|= /openapi\.json|/admin/|/static/) \{\n(?:        .*\n)+    \}\n",
        "",
        nginx,
    )
    if "http://backend" in replaced:
        raise ValueError("nginx template still mentions the backend after removing it")
    return replaced


def render_files(
    manifest: Mapping[str, object],
    secrets: Mapping[str, str],
    environment_template: Mapping[str, Mapping[str, str]],
    include_docker_templates: bool = True,
) -> Dict[str, str]:
    """Return all generated file contents for the supported golden path.

    Compose and nginx come from the master template's ``docker/templates``, which a configured
    project does not carry. ``include_docker_templates=False`` omits them so a project can render
    the rest of its own files (``doctor`` does this).
    """
    if manifest.get("backend") == "none":
        files = {"project.config.yml": json.dumps(manifest, indent=2, sort_keys=True) + "\n"}
        if include_docker_templates:
            docker_templates = Path(__file__).resolve().parents[2] / "docker" / "templates"
            files["docker-compose.yml"] = _without_backend_compose(
                (docker_templates / "docker-compose.yml").read_text(encoding="utf-8")
            )
            files["docker/nginx/nginx.conf"] = _without_backend_nginx(
                (docker_templates / "nginx.conf").read_text(encoding="utf-8")
            )
        return files

    database_password = secrets["POSTGRES_PASSWORD"]
    django_secret = secrets["DJANGO_SECRET_KEY"]
    backend_env = {
        **environment_template["backend"],
        "DJANGO_SECRET_KEY": django_secret,
        "POSTGRES_PASSWORD": database_password,
    }
    database_env = {
        **environment_template["database"],
        "POSTGRES_PASSWORD": database_password,
    }
    files = {
        "project.config.yml": json.dumps(manifest, indent=2, sort_keys=True) + "\n",
        "env/backend/.env": _dotenv(backend_env),
        "env/database/.env": _dotenv(database_env),
    }
    if include_docker_templates:
        docker_templates = Path(__file__).resolve().parents[2] / "docker" / "templates"
        compose = (docker_templates / "docker-compose.yml").read_text(encoding="utf-8")
        nginx = (docker_templates / "nginx.conf").read_text(encoding="utf-8")
        if manifest.get("frontend") == "none":
            compose = _without_frontend_compose(compose)
            nginx = _without_frontend_nginx(nginx)
        files["docker-compose.yml"] = compose
        files["docker/nginx/nginx.conf"] = nginx
    return files


def blueprint_sources(template_root: Path, backend: str = "fastapi", frontend: str = "react-mui") -> Dict[str, Path]:
    """Map the selected backend and frontend blueprints into service locations (no frontend, no entry)."""
    sources = {}
    if backend != "none":
        sources["services/backend"] = template_root / "blueprints" / ("backend-" + backend)
    if frontend != "none":
        sources["services/frontend"] = template_root / "blueprints" / ("frontend-" + frontend)
    return sources
