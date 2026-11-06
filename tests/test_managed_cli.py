"""Check Bash and backend equivalence using isolated installations and real PTYs."""

from __future__ import annotations

import errno
import hashlib
import json
import os
from pathlib import Path
import pty
import select
import shutil
import signal
import subprocess
import tempfile
import time
import unittest

ROOT = Path(__file__).resolve().parents[1]


class ManagedCLITests(unittest.TestCase):
    """Exercise the human frontend without personal provider or settings files."""

    def setUp(self) -> None:
        """Create a disposable toolkit and provider homes with no installed state."""
        temporary = tempfile.TemporaryDirectory(prefix="managed cli ")
        self.addCleanup(temporary.cleanup)
        self.root = Path(temporary.name)
        self.toolkit = self.root / "toolkit"
        for name in ("scripts", ".agents"):
            shutil.copytree(ROOT / name, self.toolkit / name)
        self.environment = dict(os.environ)
        self.environment.pop("MARKDOWN_OUTPUT_PATH", None)
        self.environment.update(
            PATH="/usr/bin:/bin:/usr/sbin:/sbin",
            AGENT_TOOLKIT_HOME=str(self.toolkit),
            AGENTS_HOME=str(self.root / "agents"),
            CODEX_HOME=str(self.root / "codex"),
            CLAUDE_HOME=str(self.root / "claude"),
        )

    def run_script(self, name: str, *arguments: str, expected: int = 0) -> str:
        """Run one real shell entrypoint and check its process status."""
        result = subprocess.run(
            [str(self.toolkit / "scripts" / name), *arguments],
            env=self.environment,
            cwd=self.root,
            capture_output=True,
            text=True,
            timeout=90,
        )
        self.assertEqual(result.returncode, expected, result.stdout + result.stderr)
        return result.stdout

    def backend(self, *arguments: str) -> dict:
        """Read a protocol response from the same engine used by the frontend."""
        return json.loads(self.run_script("managed.sh", *arguments))

    def install_both(self) -> None:
        """Apply one non-default selection to both providers."""
        self.run_script(
            "install.sh",
            "--provider",
            "codex",
            "--provider",
            "claude",
            "--skill",
            "ticket-to-plan",
            "--concise",
            "true",
            "--output-path",
            str(self.root / "output with spaces"),
        )

    def snapshot(self) -> dict[str, str]:
        """Capture provider files, links, and canonical state."""
        result = {}
        paths = [self.root / name for name in ("agents", "codex", "claude")]
        paths.append(self.toolkit / "config")
        for path in paths:
            for item in [path, *path.rglob("*")] if path.is_dir() else [path]:
                if item.is_symlink():
                    result[str(item)] = "link:" + os.readlink(item)
                elif item.is_file():
                    result[str(item)] = hashlib.sha256(item.read_bytes()).hexdigest()
        return result

    def terminal(
        self, arguments: list[str], responses: list[tuple[bytes, bytes]]
    ) -> int:
        """Answer real terminal prompts and always reap the process group."""
        master, slave = pty.openpty()
        process = subprocess.Popen(
            [str(self.toolkit / "scripts/install.sh"), *arguments],
            env=self.environment,
            cwd=self.root,
            stdin=slave,
            stdout=slave,
            stderr=slave,
            start_new_session=True,
        )
        os.close(slave)
        try:
            self.drive_prompts(process, master, responses)
            return process.wait(timeout=5)
        finally:
            if process.poll() is None:
                os.killpg(process.pid, signal.SIGKILL)
                process.wait()
            os.close(master)

    def drive_prompts(
        self,
        process: subprocess.Popen,
        master: int,
        responses: list[tuple[bytes, bytes]],
    ) -> None:
        """Match prompts in order, including terminal EOF at a removal prompt."""
        output = b""
        index = 0
        deadline = time.monotonic() + 60
        while process.poll() is None and time.monotonic() < deadline:
            if select.select([master], [], [], 0.05)[0]:
                try:
                    output += os.read(master, 65536)
                except OSError as error:
                    if error.errno != errno.EIO:
                        raise
            if index < len(responses) and responses[index][0] in output:
                os.write(master, responses[index][1])
                index += 1
                output = b""
        self.assertEqual(index, len(responses), output)
        self.assertIsNotNone(process.poll(), output)

    def test_equivalent_selections_and_denied_removals_are_read_only(self) -> None:
        """GIVEN two installs WHEN CLI selects fewer THEN backend rules are identical."""
        self.install_both()
        arguments = ["--provider", "codex", "--skill", "commit-message"]
        preview = self.backend("preview", *arguments)
        before = self.snapshot()

        output = self.run_script("install.sh", *arguments, "--dry-run")
        self.run_script("install.sh", *arguments, expected=2)

        for operation in preview["operations"]:
            self.assertIn(operation["path"], output)
            self.assertIn(operation["action"], output)
        self.assertEqual(self.snapshot(), before)
        self.run_script("install.sh", *arguments, "--confirm-removals")
        self.assertEqual(self.backend("status")["desired"], preview["desired"])
        self.assertFalse((self.root / "claude/skills/ticket-to-plan").exists())
        self.assertTrue(
            (self.root / "agents/skills/agent-toolkit/plan-review").exists()
        )

    def test_terminal_removal_accept_decline_and_eof(self) -> None:
        """GIVEN removal preview WHEN confirmed, declined, or EOF THEN only yes applies."""
        self.install_both()
        arguments = ["--provider", "codex", "--skill", "commit-message"]
        before = self.snapshot()

        for answer in (b"n\n", b"\x04"):
            with self.subTest(answer=answer):
                status = self.terminal(arguments, [(b"Apply the removals", answer)])
                self.assertEqual(status, 2)
                self.assertEqual(self.snapshot(), before)
        status = self.terminal(arguments, [(b"Apply the removals", b"y\n")])

        self.assertEqual(status, 0)
        self.assertEqual(self.backend("status")["desired"]["providers"], ["codex"])

    def test_terminal_setup_prompts_use_managed_state(self) -> None:
        """GIVEN fresh terminal setup WHEN prompts are answered THEN canonical state saves."""
        responses = [
            (b"LLM providers", b"codex claude\n"),
            (b"Skills (", b"ticket-to-plan\n"),
            (b"Update global guidance", b"True\n"),
            (
                b"Markdown output path",
                str(self.root / "output with spaces").encode() + b"\n",
            ),
            (b"Concise output loading", b"True\n"),
        ]

        status = self.terminal([], responses)

        self.assertEqual(status, 0)
        desired = self.backend("status")["desired"]
        self.assertEqual(desired["providers"], ["claude", "codex"])
        self.assertEqual(desired["requested_skills"], ["ticket-to-plan"])
        self.assertTrue(desired["full_template"])

    def test_uninstall_rolls_back_and_preserves_unrelated_content(self) -> None:
        """GIVEN saved installs WHEN uninstall fails THEN state restores; success preserves users."""
        self.install_both()
        native = self.root / "codex/plugins/unrelated/keep"
        native.parent.mkdir(parents=True)
        native.write_text("native content\n")
        user = self.root / "claude/skills/user-skill/SKILL.md"
        user.parent.mkdir(parents=True)
        user.write_text("user content\n")
        before = self.snapshot()

        for variable, value, status in (
            ("AGENT_TOOLKIT_FAIL_PROVIDER", "claude", 97),
            ("AGENT_TOOLKIT_FAIL_STATE", "after-settings", 1),
        ):
            self.environment[variable] = value
            self.run_script("uninstall.sh", expected=status)
            del self.environment[variable]
            self.assertEqual(self.snapshot(), before)
        self.run_script("uninstall.sh")

        self.assertFalse((self.toolkit / "config/settings.env").exists())
        self.assertFalse((self.toolkit / "config/inventory.json").exists())
        self.assertEqual(native.read_text(), "native content\n")
        self.assertEqual(user.read_text(), "user content\n")

    def test_fresh_defaults_and_explicit_output_modes(self) -> None:
        """GIVEN fresh setup WHEN output is selected THEN defaults and preferences save."""
        output = str(self.root / "output with spaces")

        self.run_script("install.sh", "--output-path", output)

        desired = self.backend("status")["desired"]
        self.assertEqual(desired["providers"], ["codex"])
        self.assertEqual(
            desired["requested_skills"], ["commit-message", "concise-output"]
        )
        self.assertEqual(desired["output_path"], output)
        self.assertFalse(desired["full_template"])
        self.assertFalse(desired["concise_output"])

        self.run_script("install.sh", "--output-path", "", "--confirm-removals")

        self.assertEqual(self.backend("status")["desired"]["output_path"], "")

    def test_saved_provider_deselection_does_not_restore_defaults(self) -> None:
        """GIVEN no saved providers WHEN setup repeats THEN defaults stay disabled."""
        self.install_both()
        self.run_script("install.sh", "--no-providers", "--confirm-removals")
        before = self.snapshot()

        self.run_script("install.sh", "--dry-run")
        desired = self.backend("status")["desired"]

        self.assertEqual(desired["providers"], [])
        self.assertEqual(self.snapshot(), before)

    def test_help_list_and_dry_run_do_not_write(self) -> None:
        """GIVEN no installation WHEN inspected THEN provider files and state stay absent."""
        before = self.snapshot()

        for arguments in (
            ("--help",),
            ("--list",),
            ("--provider", "claude", "--bundle", "starter", "--dry-run"),
        ):
            self.run_script("install.sh", *arguments)

        self.assertEqual(self.snapshot(), before)
        self.assertFalse((self.toolkit / "config").exists())


if __name__ == "__main__":
    unittest.main()
