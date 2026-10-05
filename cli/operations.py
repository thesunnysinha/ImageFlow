"""Operational commands retained in configured projects."""

from __future__ import annotations

import json
import re
import subprocess
import sys
from pathlib import Path

from cli.engine.configuration import ConfigurationError, load_environment_template, load_manifest, load_override
from cli.engine.render import render_files
from cli.engine.transaction import TransactionError, recover_interrupted_configure


def _env_values(text: str) -> dict:
    return dict(line.split("=", 1) for line in text.splitlines() if "=" in line)


def doctor(root: Path) -> int:
    try:
        manifest = load_manifest(root / "project.config.yml")
        # load_manifest keeps only the core choices; the file also records deployment and features.
        recorded = json.loads((root / "project.config.yml").read_text(encoding="utf-8"))
        override = load_override(root / "env" / "env.override.local.yml")
        required_secrets = []
        if manifest["backend"] != "none":
            required_secrets.append("DJANGO_SECRET_KEY")
        if manifest["database"] != "none":
            required_secrets.append("POSTGRES_PASSWORD")
        if any(not override.get(key) for key in required_secrets):
            raise ConfigurationError("Local secret override is missing required generated values")
        template = (
            load_environment_template(root / "env" / "env.template.yml")
            if manifest["backend"] != "none"
            else {}
        )
        failures = []
        if recorded.get("auth") != manifest["auth"]:
            failures.append("manifest auth does not match the selected backend")
        # Compose and nginx are rendered from the master template, which this project does not
        # carry, and configure post-processes compose for features; check only that they exist.
        for relative in ("docker-compose.yml", "docker/nginx/nginx.conf"):
            if not (root / relative).is_file():
                failures.append("missing {}".format(relative))
        if manifest["backend"] == "none":
            forbidden_paths = (
                "services/backend",
                "env/backend",
                "env/database",
                "env/env.override.local.yml",
            )
            for relative in forbidden_paths:
                if (root / relative).exists():
                    failures.append("frontend-only project contains forbidden {}".format(relative))
            compose_path = root / "docker-compose.yml"
            if compose_path.is_file():
                compose = compose_path.read_text(encoding="utf-8")
                for service in ("backend", "database", "postgres", "redis"):
                    if re.search(r"^  {}:\s*$".format(service), compose, re.MULTILINE):
                        failures.append("frontend-only compose enables forbidden {} service".format(service))
            launchpad_path = root / "launchpad" / "application.json"
            if launchpad_path.is_file():
                launchpad = json.loads(launchpad_path.read_text(encoding="utf-8"))
                expected_flags = (
                    ("backend.enabled", launchpad.get("backend", {}).get("enabled"), False),
                    ("frontend.enabled", launchpad.get("frontend", {}).get("enabled"), True),
                    ("services.postgres.enabled", launchpad.get("services", {}).get("postgres", {}).get("enabled"), False),
                    ("services.redis.enabled", launchpad.get("services", {}).get("redis", {}).get("enabled"), False),
                )
                for label, actual, expected in expected_flags:
                    if actual is not expected:
                        failures.append("frontend-only Launchpad manifest has {}={!r}; expected {!r}".format(label, actual, expected))
        for relative, expected in render_files(recorded, override, template, include_docker_templates=False).items():
            output = root / relative
            if not output.is_file():
                failures.append("missing {}".format(relative))
                continue
            actual = output.read_text(encoding="utf-8")
            if relative.endswith(".env"):
                # Features add entries that the project's env template does not list, so extras are fine;
                # every generated value must still be present and unchanged.
                expected_values, actual_values = _env_values(expected), _env_values(actual)
                drifted = any(actual_values.get(key) != value for key, value in expected_values.items())
            else:
                drifted = actual != expected
            if drifted:
                failures.append("out of sync: {}".format(relative))
        backend_files = {
            "django": ("manage.py", "config/settings.py", "api/urls.py", "agents/services.py", "shared/services/agent/graph.py"),
            "nodejs": ("package.json", "src/app.ts", "src/server.ts"),
            "gin": ("go.mod", "cmd/server/main.go", "internal/httpapi/app.go"),
            "fastapi": ("app/main.py", "app/agents/service.py", "app/agents/routes.py", "shared/services/agent/graph.py"),
        }
        frontend_files = {
            "react-mui": ("package.json", "src/App.tsx", "src/state/api.ts"),
            "nextjs": ("package.json", "next.config.mjs", "app/page.tsx", "app/layout.tsx"),
        }
        required = {".": ("run.py",), "cli": ("runtime.py", "operations.py")}
        if manifest["backend"] != "none":
            required["services/backend"] = backend_files[manifest["backend"]]
        if manifest["frontend"] != "none":
            required["services/frontend"] = frontend_files[manifest["frontend"]]
        for relative, filenames in required.items():
            directory = root / relative
            if not directory.is_dir():
                failures.append("missing {}".format(relative))
                continue
            for filename in filenames:
                if not (directory / filename).is_file():
                    failures.append("missing {}/{}".format(relative, filename))
        if failures:
            print("doctor: FAILED", file=sys.stderr)
            for failure in failures:
                print("  - {}".format(failure), file=sys.stderr)
            return 1
    except (ConfigurationError, OSError, json.JSONDecodeError) as error:
        print("doctor: FAILED: {}".format(error), file=sys.stderr)
        return 1
    print("doctor: OK — manifest, generated configuration, and selected services are consistent")
    return 0


def recover(root: Path) -> int:
    snapshot = recover_interrupted_configure(root.resolve())
    print("Recovered partial outputs. Source snapshot retained at {}".format(snapshot))
    print("Rerun configure from the master template when ready.")
    return 0


def setup(root: Path) -> int:
    if doctor(root.resolve()) != 0:
        return 1
    manifest = load_manifest(root / "project.config.yml")
    built = subprocess.run(["docker", "compose", "build"], cwd=root, check=False)
    if built.returncode:
        return built.returncode
    if manifest["backend"] == "django":
        return subprocess.call(["docker", "compose", "run", "--rm", "backend", "python", "manage.py", "migrate"], cwd=root)
    if manifest["backend"] == "none":
        print("Frontend image built; this project has no backend or database setup.")
        return 0
    print("FastAPI selected; this blueprint defines no database migrations.")
    return 0


def dev(root: Path) -> int:
    if doctor(root.resolve()) != 0:
        return 1
    return subprocess.call(["docker", "compose", "up", "--build"], cwd=root)
