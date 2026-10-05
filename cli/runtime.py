"""Runtime-only dispatcher copied into configured projects."""

from __future__ import annotations

import argparse
from pathlib import Path
from typing import Optional, Sequence

from cli.operations import dev, doctor, recover, setup


def main(argv: Optional[Sequence[str]] = None) -> int:
    parser = argparse.ArgumentParser(description="Operate the configured project.")
    subparsers = parser.add_subparsers(dest="command", required=True)
    commands = {}
    for name, help_text in (("doctor", "validate generated outputs"), ("recover", "recover an interrupted configure"), ("setup", "build images and prepare the selected services"), ("dev", "run the local stack")):
        command = subparsers.add_parser(name, help=help_text)
        command.add_argument("--root", type=Path, default=Path.cwd())
        commands[name] = command
    args = parser.parse_args(argv)
    try:
        return {"doctor": doctor, "recover": recover, "setup": setup, "dev": dev}[args.command](args.root)
    except Exception as error:
        parser.exit(2, "{}: {}\n".format(args.command, error))


if __name__ == "__main__":
    raise SystemExit(main())
