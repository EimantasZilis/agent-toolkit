"""Run the TUI launcher with local network responses in subprocess tests."""

import os
import sys
from pathlib import Path
from typing import BinaryIO
from unittest.mock import patch

# The test copies this runner into the isolated checkout's scripts directory.
sys.path.insert(0, str(Path(__file__).resolve().parent.parent))
from scripts import launch_tui  # noqa: E402


def response(url: str, timeout: int) -> BinaryIO:
    """Return local release bytes or simulate an unavailable release."""
    with Path("downloads").open("a") as output:
        output.write("request\n")

    if os.environ.get("FIXTURE_DOWNLOAD_FAILURE"):
        raise OSError("release unavailable")

    return Path("release.zip").open("rb")


def main() -> int:
    """Run the launcher with subprocess-local network and platform mocks."""
    with patch("urllib.request.OpenerDirector.open", side_effect=response):
        with patch("platform.system", return_value=os.environ["FIXTURE_SYSTEM"]):
            return launch_tui.main()


if __name__ == "__main__":
    sys.exit(main())
