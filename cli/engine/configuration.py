"""Configuration parsing and compatibility checks."""

from __future__ import annotations

import json
from pathlib import Path
from typing import Any, Dict, Mapping

SUPPORTED = {
    "backend": {"django", "fastapi", "gin", "nodejs", "none"},
    "frontend": {"nextjs", "none", "react-mui"},
    "database": {"postgresql", "none"},
}
DEFAULT_CONFIG = {"backend": "fastapi", "frontend": "react-mui", "database": "postgresql"}


class ConfigurationError(ValueError):
    """Raised when a project configuration is malformed or unsupported."""


def build_manifest(choices: Mapping[str, str]) -> Dict[str, Any]:
    """Validate supported choices and return the versioned project manifest."""
    normalized = dict(DEFAULT_CONFIG)
    for key, value in choices.items():
        if key not in SUPPORTED:
            raise ConfigurationError("Unknown configuration option: {}".format(key))
        if value not in SUPPORTED[key]:
            options = ", ".join(sorted(SUPPORTED[key]))
            raise ConfigurationError(
                "Unsupported {} '{}'; supported choice: {}".format(key, value, options)
            )
        normalized[key] = value
    backend = normalized["backend"]
    frontend = normalized["frontend"]
    database = normalized["database"]
    if backend == "none" and frontend == "none":
        raise ConfigurationError("A project must include a backend or a frontend; backend and frontend cannot both be 'none'")
    if backend == "none" and database != "none":
        raise ConfigurationError("Frontend-only projects must use database 'none'; PostgreSQL requires a backend")
    if backend != "none" and database == "none":
        raise ConfigurationError(
            "The {} backend requires database 'postgresql'; database 'none' is only supported for frontend-only projects".format(
                backend
            )
        )
    auth = "none" if backend == "none" else "jwt-cookie"
    return {"schema_version": 1, **normalized, "auth": auth, "deployment": "docker-compose"}


def load_manifest(path: Path) -> Dict[str, Any]:
    """Load JSON-compatible YAML and validate its schema and choices."""
    try:
        parsed = json.loads(path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as error:
        raise ConfigurationError("Could not read {} as JSON-compatible YAML: {}".format(path, error))
    if not isinstance(parsed, dict) or parsed.get("schema_version") != 1:
        raise ConfigurationError("project.config.yml must be an object with schema_version: 1")
    return build_manifest({key: parsed.get(key, "") for key in SUPPORTED})


def load_override(path: Path) -> Dict[str, str]:
    """Read the local secret override, returning an empty mapping if absent."""
    if not path.exists():
        return {}
    try:
        values = json.loads(path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as error:
        raise ConfigurationError("Could not read local environment override: {}".format(error))
    if not isinstance(values, dict) or not all(
        isinstance(key, str) and isinstance(value, str) for key, value in values.items()
    ):
        raise ConfigurationError("Local environment override must contain string values")
    return values


def load_environment_template(path: Path) -> Dict[str, Dict[str, str]]:
    """Load sectioned environment defaults from the committed template."""
    try:
        sections = json.loads(path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as error:
        raise ConfigurationError("Could not read environment template: {}".format(error))
    if not isinstance(sections, dict):
        raise ConfigurationError("Environment template must be an object of service sections")
    normalized: Dict[str, Dict[str, str]] = {}
    for service, values in sections.items():
        if not isinstance(service, str) or not isinstance(values, dict):
            raise ConfigurationError("Environment template sections must contain key/value objects")
        if not all(isinstance(key, str) and isinstance(value, (str, int, bool)) for key, value in values.items()):
            raise ConfigurationError("Environment template values must be strings, integers, or booleans")
        normalized[service] = {key: str(value).lower() if isinstance(value, bool) else str(value) for key, value in values.items()}
    if not {"backend", "database"}.issubset(normalized):
        raise ConfigurationError("Environment template must define backend and database sections")
    return normalized
