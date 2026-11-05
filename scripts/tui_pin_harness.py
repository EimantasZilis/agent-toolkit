#!/usr/bin/env python3
"""Instrument launcher downloads and Bash routes in isolated acceptance checks."""

from __future__ import annotations

import importlib.util
import json
import os
import sys
import urllib.request
from pathlib import Path
from unittest.mock import patch


def record(kind: str, value: str) -> None:
    """Append an observable boundary event outside the checked checkout."""
    with Path(os.environ["PIN_CHECK_LOG"]).open("a") as output:
        output.write(json.dumps({"kind": kind, "value": value}) + "\n")


def launch(arguments: list[str]) -> int:
    """Load the checkout's real launcher with only its network boundary observed."""
    path = Path(arguments[0]).resolve()
    if path != Path(os.environ["PIN_CHECK_ROOT"]).resolve() / "scripts/launch_tui.py":
        os.execv(sys.executable, [sys.executable, *arguments])
    spec = importlib.util.spec_from_file_location("checked_launcher", path)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    if os.environ.get("PIN_CHECK_REPOSITORY"):
        module.RELEASE_URL = (
            "https://github.com/"
            + os.environ["PIN_CHECK_REPOSITORY"]
            + "/releases/download"
        )
    sys.argv = arguments
    original = urllib.request.OpenerDirector.open

    def observed_open(opener: object, url: str, timeout: int) -> object:
        """Disable downloads on repeat launches without a production offline flag."""
        record("download", url)
        if os.environ.get("PIN_CHECK_OFFLINE") == "1":
            raise OSError("acceptance harness forbids a cached-launch download")
        fixture = os.environ.get("PIN_CHECK_FIXTURE")
        if fixture:
            return (Path(fixture) / url.rsplit("/", 1)[-1]).open("rb")
        return original(opener, url, timeout=timeout)

    with patch.object(urllib.request.OpenerDirector, "open", observed_open):
        return module.main()


def main() -> int:
    """Intercept only the installer route; execute other Bash commands unchanged."""
    kind, *arguments = sys.argv[1:]
    if kind == "python":
        return launch(arguments)
    installer = Path(arguments[0]).resolve()
    if installer == Path(os.environ["PIN_CHECK_ROOT"]).resolve() / "scripts/install.sh":
        record("installer", json.dumps(arguments[1:]))
        if os.environ.get("PIN_CHECK_ROUTE") == "1":
            return 0
    os.execv("/bin/bash", ["bash", *arguments])
    return 1


if __name__ == "__main__":
    sys.exit(main())
