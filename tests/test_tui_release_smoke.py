"""Exercise the native harness with real PTYs and disposable executables."""

from __future__ import annotations

import json
import sys
import tempfile
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent.parent / "scripts"))
import build_tui
import smoke_tui_release as smoke

VERSION = "v0.0.0-smoke"


class SmokeTests(unittest.TestCase):
    """Build complete fixture releases and check observable runtime failures."""

    def setUp(self) -> None:
        """Create an isolated source root and four-target release directory."""
        temporary = tempfile.TemporaryDirectory(prefix="tui smoke tests ")
        self.addCleanup(temporary.cleanup)
        self.root = Path(temporary.name)
        self.checkout = self.root / "checkout"
        self.checkout.mkdir()
        self.directory = self.root / "release"
        self.directory.mkdir()

    def release(self, body: str, version: str = VERSION) -> None:
        """Package a fixture with exact version output and custom PTY behavior."""
        binary = self.root / "fixture"
        binary.write_text(
            "#!/bin/bash\n"
            "if [[ ${1:-} == --version ]]; then\n"
            f" printf 'agent-toolkit {version}; managed protocol 1\\n'; exit 0\n"
            "fi\n" + body
        )
        self.package(binary)

    def package(self, binary: Path) -> None:
        """Write checksums and all archive records for the fixture."""
        records = {}
        for target in build_tui.TARGETS:
            archive = self.directory / f"agent-toolkit-{VERSION}-{target}.zip"
            build_tui.archive_binary(
                binary,
                archive,
                {
                    "version": VERSION,
                    "protocol_version": 1,
                    "target": target,
                },
            )
            records[target] = {
                "archive": archive.name,
                "sha256": build_tui.digest(archive),
                "binary_sha256": build_tui.digest(binary),
            }
        (self.directory / "release.json").write_text(
            json.dumps(
                {
                    "version": VERSION,
                    "protocol_version": 1,
                    "targets": records,
                }
            )
        )
        (self.directory / "SHA256SUMS").write_text(
            "".join(
                f"{record['sha256']}  {record['archive']}\n"
                for record in records.values()
            )
        )

    def run_smoke(self, timeout: float = 2) -> dict[str, object]:
        """Run the actual release validator and native PTY harness."""
        return smoke.smoke(
            VERSION, smoke.native_target(), self.directory, self.checkout, timeout
        )

    def screen(self) -> str:
        """Return a loaded-screen fixture that handles quit and Discard."""
        return (
            "stty raw -echo\n"
            "echo 'agent-toolkit · setup and settings LLM providers q quit'\n"
            "read -r -n 1 key; [[ $key == q ]] || exit 4\n"
            "echo Discard\n"
            "read -r -n 1 key; [[ $key == d ]] || exit 5\n"
        )

    def test_successful_quit_discards_without_writes(self) -> None:
        """GIVEN a loaded screen WHEN quit is sent THEN Discard exits cleanly."""
        self.release(self.screen())

        report = self.run_smoke()

        self.assertNotIn("error", report)
        self.assertTrue(
            all(check["status"] == "passed" for check in report["checks"].values())
        )

    def test_failure_modes(self) -> None:
        """GIVEN runtime faults WHEN checked THEN each required failure is reported."""
        cases = (
            ("sleep 30\n", VERSION, "startup timeout"),
            ("echo 'version only'\n", VERSION, "startup/exit failed"),
            (self.screen() + "exit 3\n", VERSION, "startup/exit failed"),
            (
                "stty raw -echo\necho 'setup and settings LLM providers q quit'\nsleep 30\n",
                VERSION,
                "exit timeout",
            ),
            (self.screen(), "v9.9.9", "version output mismatch"),
            (
                self.screen() + 'touch "$HOME/forbidden"\n',
                VERSION,
                "unexpected managed",
            ),
            (
                self.screen() + "mkdir config; touch config/settings.env\n",
                VERSION,
                "unexpected managed",
            ),
        )
        for body, version, expected in cases:
            with self.subTest(expected=expected):
                self.release(body, version)

                report = self.run_smoke(timeout=2)

                self.assertIn(expected, report["error"])

    def test_wrong_host_prevents_execution(self) -> None:
        """GIVEN another target WHEN checked THEN native execution is refused."""
        other = next(
            target for target in build_tui.TARGETS if target != smoke.native_target()
        )

        report = smoke.smoke(VERSION, other, self.directory, self.checkout)

        self.assertIn("host mismatch", report["error"])


if __name__ == "__main__":
    unittest.main()
