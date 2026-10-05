"""Optional features: a catalog read from ``blueprints/features/<id>/feature.json`` and copied into a project.

A feature folder holds ``feature.json`` and ``files/`` (copied into ``services/backend``). ``feature.json``:
``id``, ``label``, ``description``, ``requires`` (other feature ids), ``requirements`` (pip lines),
``env`` (``name``, ``default``, ``description``), ``services`` (platform services such as ``redis``) and
``secrets`` (names whose values the operator supplies). Optional: ``postgres_extensions`` and ``workers``
(background processes run from the backend image: ``name``, ``command``, ``replicas``, ``cpu``, ``memory``).
"""

from __future__ import annotations

import json
import shutil
from pathlib import Path
from typing import Any, Dict, Iterable, List, Mapping

from cli.engine.configuration import ConfigurationError

REQUIRED_KEYS = ("id", "label", "description", "requires", "requirements", "env", "services", "secrets")
LOCAL_REDIS_URL = "redis://redis:6379/0"


def _features_root(template_root: Path) -> Path:
    return template_root / "blueprints" / "features"


def load_catalog(template_root: Path) -> Dict[str, Dict[str, Any]]:
    """Read and validate every ``feature.json``; keys are feature ids."""
    catalog: Dict[str, Dict[str, Any]] = {}
    root = _features_root(template_root)
    for path in sorted(root.glob("*/feature.json")):
        try:
            feature = json.loads(path.read_text(encoding="utf-8"))
        except (OSError, json.JSONDecodeError) as error:
            raise ConfigurationError("Could not read {}: {}".format(path, error))
        missing = [key for key in REQUIRED_KEYS if key not in feature]
        if missing or feature["id"] != path.parent.name:
            raise ConfigurationError("{} must define {} and match its folder name".format(path, ", ".join(REQUIRED_KEYS)))
        catalog[feature["id"]] = feature
    return catalog


def parse_feature_list(value: str | None) -> List[str]:
    return [item.strip() for item in (value or "").split(",") if item.strip()]


def resolve_features(catalog: Mapping[str, Mapping[str, Any]], selected: Iterable[str]) -> List[str]:
    """Return the selected features plus what they require, dependencies first, without duplicates."""
    ordered: List[str] = []

    def visit(feature_id: str, chain: tuple) -> None:
        if feature_id not in catalog:
            raise ConfigurationError(
                "Unknown feature '{}'; available: {}".format(feature_id, ", ".join(sorted(catalog)) or "none")
            )
        if feature_id in chain:
            raise ConfigurationError("Feature dependency cycle: {}".format(" -> ".join(chain + (feature_id,))))
        for dependency in catalog[feature_id]["requires"]:
            visit(dependency, chain + (feature_id,))
        if feature_id not in ordered:
            ordered.append(feature_id)

    for feature_id in selected:
        visit(feature_id, ())
    return ordered


def validate_features(catalog: Mapping[str, Mapping[str, Any]], ordered: List[str], backend: str, deploy_mode: str | None) -> None:
    if ordered and backend == "none":
        raise ConfigurationError("Features require a backend and are not available for frontend-only projects")
    if ordered and backend != "fastapi":
        raise ConfigurationError("Features are only available for the fastapi backend")
    if deploy_mode == "vm_tool":
        blocked = [
            fid
            for fid in ordered
            if catalog[fid]["services"] or catalog[fid]["secrets"] or catalog[fid].get("postgres_extensions") or catalog[fid].get("workers")
        ]
        if blocked:
            raise ConfigurationError(
                "Feature(s) {} need platform services, secrets or database extensions that the vm_tool deploy files do not "
                "provide; use --deploy-mode launchpad, or configure without deployment files".format(", ".join(blocked))
            )


def apply_features(destination: Path, template_root: Path, ordered: List[str]) -> None:
    """Copy each feature's files into the backend and add its pip requirements."""
    written: Dict[Path, str] = {}
    lines: List[str] = []
    for feature_id in ordered:
        source = _features_root(template_root) / feature_id / "files"
        for path in sorted(p for p in source.rglob("*") if p.is_file() and "__pycache__" not in p.parts):
            relative = path.relative_to(source)
            if relative in written:
                raise ConfigurationError("Features '{}' and '{}' both provide {}".format(written[relative], feature_id, relative))
            written[relative] = feature_id
            target = destination / relative
            target.parent.mkdir(parents=True, exist_ok=True)
            shutil.copy2(str(path), str(target))
    catalog = load_catalog(template_root)
    for feature_id in ordered:
        lines.extend(catalog[feature_id]["requirements"])
    if lines:
        requirements = destination / "requirements.txt"
        existing = requirements.read_text(encoding="utf-8").splitlines()
        present = {line.split(">")[0].split("=")[0].strip().lower() for line in existing}
        additions = [line for line in lines if line.split(">")[0].split("=")[0].strip().lower() not in present]
        requirements.write_text("\n".join(existing + ["# Added by features: " + ", ".join(ordered)] + additions) + "\n", encoding="utf-8")


def env_additions(catalog: Mapping[str, Mapping[str, Any]], ordered: List[str]) -> Dict[str, str]:
    """Environment entries for env/backend/.env: defaults for settings, empty values for secrets."""
    values: Dict[str, str] = {}
    for feature_id in ordered:
        for item in catalog[feature_id]["env"]:
            values[item["name"]] = LOCAL_REDIS_URL if item["name"] == "REDIS_URL" else str(item["default"])
        for name in catalog[feature_id]["secrets"]:
            values.setdefault(name, "")
    return values


def add_redis_to_compose(compose: str) -> str:
    """Add a Redis container to the local Docker Compose file and make the backend wait for it."""
    service = (
        "  redis:\n    image: redis:7-alpine\n    healthcheck:\n      test: [\"CMD\", \"redis-cli\", \"ping\"]\n"
        "      interval: 5s\n      timeout: 3s\n      retries: 20\n"
    )
    dependency = "      database:\n        condition: service_healthy\n"
    if "  backend:\n" not in compose or dependency not in compose:
        raise ConfigurationError("The compose template changed; update add_redis_to_compose")
    compose = compose.replace("  backend:\n", service + "  backend:\n", 1)
    return compose.replace(dependency, dependency + "      redis:\n        condition: service_healthy\n", 1)


def workers_for(catalog: Mapping[str, Mapping[str, Any]], ordered: List[str]) -> List[Dict[str, Any]]:
    """Background workers the selected features declare, in feature order; a name may only be declared once."""
    found: List[Dict[str, Any]] = []
    for feature_id in ordered:
        for worker in catalog[feature_id].get("workers", []):
            if any(existing["name"] == worker["name"] for existing in found):
                raise ConfigurationError("Two features declare a worker named '{}'".format(worker["name"]))
            found.append(dict(worker))
    return found


def add_workers_to_compose(compose: str, workers: List[Dict[str, Any]]) -> str:
    """Run each worker as a container built from the backend, with the backend's env file and dependencies."""
    marker = "  frontend:\n" if "  frontend:\n" in compose else "  proxy:\n"
    if marker not in compose or "  backend:\n" not in compose:
        raise ConfigurationError("The compose template changed; update add_workers_to_compose")
    services = ""
    for worker in workers:
        services += (
            "  {name}:\n    build: ./services/backend\n    env_file: env/backend/.env\n    command: {command}\n"
            "    depends_on:\n      database:\n        condition: service_healthy\n      redis:\n        condition: service_healthy\n"
        ).format(name=worker["name"], command=json.dumps(worker["command"]))
    return compose.replace(marker, services + marker, 1)


def postgres_extensions(catalog: Mapping[str, Mapping[str, Any]], ordered: List[str]) -> List[str]:
    """Database extensions the selected features need, without duplicates."""
    found: List[str] = []
    for feature_id in ordered:
        for name in catalog[feature_id].get("postgres_extensions", []):
            if name not in found:
                found.append(name)
    return found


def use_pgvector_in_compose(compose: str) -> str:
    """Run local Postgres from an image that ships pgvector."""
    if "image: postgres:16-alpine" not in compose:
        raise ConfigurationError("The compose template changed; update use_pgvector_in_compose")
    return compose.replace("image: postgres:16-alpine", "image: pgvector/pgvector:pg16", 1)


def apply_to_launchpad_manifest(manifest_json: str, catalog: Mapping[str, Mapping[str, Any]], ordered: List[str]) -> str:
    """Turn on the platform services the features need and declare their secret names in the Launchpad manifest."""
    entry = json.loads(manifest_json)
    for feature_id in ordered:
        feature = catalog[feature_id]
        if "redis" in feature["services"]:
            entry["services"]["redis"] = {"enabled": True}
        for extension in feature.get("postgres_extensions", []):
            if extension not in entry["services"]["postgres"]["extensions"]:
                entry["services"]["postgres"]["extensions"].append(extension)
        for worker in feature.get("workers", []):
            entry.setdefault("workers", []).append(dict(worker))
        for item in feature["env"]:
            if item["name"] != "REDIS_URL":  # delivered by Launchpad's Redis Secret
                entry["environment"]["backend"][item["name"]] = str(item["default"])
        for name in feature["secrets"]:
            if name not in entry["secretNames"]:
                entry["secretNames"].append(name)
    return json.dumps(entry, indent=2) + "\n"
