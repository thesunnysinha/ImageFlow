"""Staged filesystem writes with snapshots and rollback on failure."""

from __future__ import annotations

import os
import json
import shutil
import tempfile
from datetime import datetime, timezone
from pathlib import Path
from typing import Mapping, Optional
from uuid import uuid4

# Local build output never belongs in a generated project (a blueprint folder may hold it after someone ran npm there).
BUILD_ARTIFACTS = shutil.ignore_patterns("node_modules", ".next", "dist", "__pycache__", "*.pyc")


class TransactionError(RuntimeError):
    """Raised when project generation cannot be completed safely."""


def apply_outputs(root: Path, files: Mapping[str, str], directories: Mapping[str, Path]) -> Optional[Path]:
    """Stage, snapshot, and promote configured output files and blueprints."""
    root.mkdir(parents=True, exist_ok=True)
    marker = root / ".configure-incomplete"
    if marker.exists():
        raise TransactionError("An interrupted configure run was found; run 'python run.py recover' first")
    for relative, source in directories.items():
        if not source.is_dir():
            raise TransactionError("Required blueprint directory is missing: {}".format(source))
        if (root / relative).exists():
            raise TransactionError("Output already exists; refusing to overwrite {}".format(root / relative))
    for relative in files:
        if (root / relative).exists():
            raise TransactionError("Output already exists; refusing to overwrite {}".format(root / relative))

    snapshot: Optional[Path] = None
    snapshot_created = False
    stage = Path(tempfile.mkdtemp(prefix=".configure-stage-", dir=str(root)))
    created = []
    try:
        for relative, contents in files.items():
            staged_file = stage / relative
            staged_file.parent.mkdir(parents=True, exist_ok=True)
            staged_file.write_text(contents, encoding="utf-8")
            if relative.startswith("env/") and (relative.endswith(".env") or "override" in relative):
                os.chmod(str(staged_file), 0o600)
        for relative, source in directories.items():
            shutil.copytree(source, stage / relative, ignore=BUILD_ARTIFACTS)

        timestamp = datetime.now(timezone.utc).strftime("%Y%m%dT%H%M%S%fZ")
        snapshot = root / ".configure-backups" / "{}-{}".format(timestamp, uuid4().hex[:8])
        snapshot.mkdir(parents=True, exist_ok=False)
        snapshot_created = True
        for relative, source in directories.items():
            shutil.copytree(source, snapshot / "source" / relative, ignore=BUILD_ARTIFACTS)

        staged_marker = stage / ".configure-incomplete"
        try:
            manifest = json.loads(files.get("project.config.yml", "{}"))
            backend = manifest.get("backend")
            frontend = manifest.get("frontend")
        except (TypeError, json.JSONDecodeError):
            backend = None
            frontend = None
        staged_marker.write_text(
            json.dumps({"snapshot": str(snapshot), "backend": backend, "frontend": frontend}, indent=2) + "\n",
            encoding="utf-8",
        )
        os.replace(str(staged_marker), str(marker))

        for relative in list(files) + list(directories):
            staged_item = stage / relative
            target = root / relative
            target.parent.mkdir(parents=True, exist_ok=True)
            os.replace(str(staged_item), str(target))
            created.append(target)
        marker.unlink()
    except Exception as error:
        for target in reversed(created):
            if target.is_dir():
                shutil.rmtree(target, ignore_errors=True)
            else:
                target.unlink(missing_ok=True)
        if snapshot_created and snapshot is not None and snapshot.exists():
            shutil.rmtree(snapshot, ignore_errors=True)
        marker.unlink(missing_ok=True)
        raise TransactionError("Configuration failed and outputs were rolled back: {}".format(error))
    finally:
        shutil.rmtree(stage, ignore_errors=True)
    return snapshot


def recover_interrupted_configure(root: Path) -> Path:
    """Remove partial generated outputs and recover any missing source blueprints."""
    marker = root / ".configure-incomplete"
    if not marker.is_file():
        raise TransactionError("No interrupted configure run was recorded")
    try:
        state = json.loads(marker.read_text(encoding="utf-8"))
        snapshot = Path(state["snapshot"])
    except (OSError, KeyError, TypeError, json.JSONDecodeError) as error:
        raise TransactionError("Configure recovery marker is unreadable: {}".format(error))
    if not snapshot.is_dir():
        raise TransactionError("Recovery snapshot is missing: {}".format(snapshot))

    for relative in (
        "project.config.yml",
        "docker-compose.yml",
        "docker/nginx/nginx.conf",
        "env/backend/.env",
        "env/database/.env",
        "services/backend",
        "services/frontend",
    ):
        target = root / relative
        if target.is_dir():
            shutil.rmtree(target)
        elif target.exists():
            target.unlink()

    backend = state.get("backend")
    if backend not in {"django", "fastapi", "gin", "nodejs", "none"}:
        backend = next(
            (name for name in ("django", "fastapi", "gin", "nodejs") if (snapshot / "source" / "blueprints" / ("backend-" + name)).is_dir()),
            "django",
        )
    frontend = state.get("frontend")
    if frontend not in {"nextjs", "react-mui", "none"}:
        frontend = "react-mui"
    source_restore = {}
    if backend != "none":
        source_restore["blueprints/backend-" + backend] = snapshot / "source" / "services/backend"
    if frontend != "none":
        source_restore["blueprints/frontend-" + frontend] = snapshot / "source" / "services/frontend"
    for relative, saved in source_restore.items():
        target = root / relative
        if not target.exists() and saved.is_dir():
            target.parent.mkdir(parents=True, exist_ok=True)
            shutil.copytree(saved, target)
    for stage in root.glob(".configure-stage-*"):
        shutil.rmtree(stage, ignore_errors=True)
    marker.unlink()
    return snapshot
