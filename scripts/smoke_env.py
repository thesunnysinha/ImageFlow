"""Writes throwaway secrets and the generated env files so `docker compose up` works in a fresh checkout or in CI.

Existing files are left alone. Secrets are random and live only in the git-ignored env files.
"""
import json
import secrets
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))  # the project root, for the cli package

from cli.engine.configuration import load_environment_template, load_manifest
from cli.engine.render import render_files

root = Path(__file__).resolve().parents[1]
override = root / "env" / "env.override.local.yml"
if not override.exists():
    override.write_text(
        '{\n  "DJANGO_SECRET_KEY": "%s",\n  "POSTGRES_PASSWORD": "%s"\n}\n' % (secrets.token_hex(24), secrets.token_hex(16))
    )

files = render_files(
    load_manifest(root / "project.config.yml"),
    json.loads(override.read_text()),
    load_environment_template(root / "env" / "env.template.yml"),
    include_docker_templates=False,
)
for relative, content in files.items():
    target = root / relative
    if relative.endswith(".env") and not target.exists():
        target.parent.mkdir(parents=True, exist_ok=True)
        target.write_text(content)
print("env files ready")
