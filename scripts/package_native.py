#!/usr/bin/env python3
"""Build or validate native releases without changing managed installations."""

from __future__ import annotations

import argparse
import subprocess
import sys
import zipfile
from pathlib import Path

from native import Catalog
from native.release import ReleaseBuilder
from native.release_validation import validate_release
from native.snapshot import build_snapshot
from native.snapshot import validate_snapshot


def parse_arguments() -> argparse.Namespace:
    """Keep archive and combined Git snapshot interfaces explicit."""
    parser = argparse.ArgumentParser(description=__doc__)
    commands = parser.add_subparsers(dest="command", required=True)
    build = commands.add_parser("build")
    build.add_argument("--version", required=True, help="Release version X.Y.Z")
    build.add_argument(
        "--output", required=True, type=Path, help="New release directory"
    )
    validate = commands.add_parser("validate")
    validate.add_argument("release", type=Path)
    snapshot = commands.add_parser("snapshot")
    snapshot.add_argument("--version", required=True)
    snapshot.add_argument("--output", required=True, type=Path)
    snapshot_check = commands.add_parser("validate-snapshot")
    snapshot_check.add_argument("snapshot", type=Path)
    return parser.parse_args()


def main() -> int:
    """Report build or standalone validation failures without provider writes."""
    args = parse_arguments()
    try:
        if args.command == "build":
            root = Path(__file__).resolve().parents[1]
            ReleaseBuilder(Catalog(root), args.version).build(args.output)
            print(f"Built and validated native release: {args.output}")
        elif args.command == "snapshot":
            build_snapshot(
                Path(__file__).resolve().parents[1], args.version, args.output
            )
            print(f"Built and validated Git snapshot: {args.output}")
        elif args.command == "validate-snapshot":
            validate_snapshot(args.snapshot)
            print(f"Validated Git snapshot: {args.snapshot}")
        else:
            validate_release(args.release)
            print(f"Validated native release: {args.release}")
    except (
        OSError,
        ValueError,
        KeyError,
        TypeError,
        zipfile.BadZipFile,
        subprocess.CalledProcessError,
    ) as error:
        print(f"Native packaging failed: {error}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
