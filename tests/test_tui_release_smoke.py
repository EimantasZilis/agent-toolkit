"""Exercise the native harness with real PTYs and disposable executables."""

from __future__ import annotations

import json
import os
import sys
import tempfile
import time
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


class VersionCheckTests(unittest.TestCase):
    """Exercise real pipes and descendant cleanup for faulty version commands."""

    def setUp(self) -> None:
        """Create a disposable executable and working directory."""
        temporary = tempfile.TemporaryDirectory(prefix="version checks ")
        self.addCleanup(temporary.cleanup)
        self.root = Path(temporary.name)
        self.binary = self.root / "fixture"
        self.environment = {
            "PATH": os.environ["PATH"],
            "AGENT_TOOLKIT_HOME": str(self.root),
        }

    def program(self, body: str) -> None:
        """Write a native shell fixture without changing the release builder."""
        self.binary.write_text("#!/bin/bash\n" + body)
        self.binary.chmod(0o755)

    def test_each_output_stream_is_bounded(self) -> None:
        """GIVEN endless stdout or stderr WHEN checked THEN fail on size before timeout."""
        for stream in ("1", "2"):
            with self.subTest(stream=stream):
                self.program(f"while :; do printf '%4096s' x >&{stream}; done\n")

                with self.assertRaisesRegex(ValueError, "output exceeds"):
                    smoke.version_check(self.binary, self.environment, VERSION, 2)

    def test_timeout_cleans_descendants_even_after_parent_exit(self) -> None:
        """GIVEN inherited pipes WHEN a parent hangs or exits THEN kill descendants at timeout."""
        for parent in ("wait", "exit 0"):
            with self.subTest(parent=parent):
                marker = self.root / "leaked"
                ready = self.root / "child-ready"
                ready.unlink(missing_ok=True)
                self.program(
                    "(echo ready > child-ready; sleep 1; touch leaked) &\n"
                    + parent
                    + "\n"
                )
                started = time.monotonic()

                with self.assertRaisesRegex(ValueError, "version timeout"):
                    smoke.version_check(self.binary, self.environment, VERSION, 0.3)

                self.assertLess(time.monotonic() - started, 3)
                self.assertEqual(ready.read_text(), "ready\n")
                time.sleep(1.1)
                self.assertFalse(marker.exists(), "descendant survived cleanup")

    def test_closed_pipes_do_not_bypass_process_deadline(self) -> None:
        """GIVEN closed stdout/stderr WHEN the process hangs THEN the same deadline applies."""
        self.program("exec 1>&- 2>&-\nsleep 30\n")

        with self.assertRaises(smoke.subprocess.TimeoutExpired):
            smoke.version_check(self.binary, self.environment, VERSION, 0.3)

    def test_nonzero_version_exit_is_rejected(self) -> None:
        """GIVEN correct output but nonzero exit WHEN checked THEN fail the version check."""
        self.program(f"echo 'agent-toolkit {VERSION}; managed protocol 1'\nexit 7\n")

        with self.assertRaisesRegex(ValueError, "status 7"):
            smoke.version_check(self.binary, self.environment, VERSION, 2)


if __name__ == "__main__":
    unittest.main()
