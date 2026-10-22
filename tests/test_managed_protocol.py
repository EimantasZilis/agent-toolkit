"""Verify the managed backend's public JSON contract against real filesystems."""

from __future__ import annotations

import json
import os
import shutil
import subprocess
import tempfile
import unittest
from pathlib import Path

REPO = Path(__file__).resolve().parents[1]
FIXTURES = REPO / "tests/fixtures/managed"


class ManagedProtocolTests(unittest.TestCase):
    """Exercise versioned commands using a disposable toolkit and provider homes."""

    def setUp(self) -> None:
        """Create an empty installation without accessing the user's providers."""
        temporary = tempfile.TemporaryDirectory(prefix="managed protocol ")
        self.addCleanup(temporary.cleanup)
        self.root = Path(temporary.name)
        self.toolkit = self.root / "toolkit"
        for directory in ("scripts", ".agents"):
            shutil.copytree(REPO / directory, self.toolkit / directory)
        shutil.copyfile(REPO / ".env.template", self.toolkit / ".env.template")

        self.environment = dict(os.environ)
        for key in ("MARKDOWN_OUTPUT_PATH", "AGENT_TOOLKIT_FAIL_STATE"):
            self.environment.pop(key, None)
        self.environment.update(
            AGENT_TOOLKIT_HOME=str(self.toolkit),
            AGENT_TOOLKIT_ENV_FILE=str(self.root / "legacy.env"),
            AGENTS_HOME=str(self.root / "agents"),
            CODEX_HOME=str(self.root / "codex"),
            CLAUDE_HOME=str(self.root / "claude"),
        )

    def command(self, *arguments: str, expected: int = 0) -> dict[str, object]:
        """Parse one JSON document and check its process status and envelope."""
        result = subprocess.run(
            [str(self.toolkit / "scripts/managed.sh"), *arguments],
            cwd=self.root,
            env=self.environment,
            capture_output=True,
            text=True,
            timeout=90,
            check=False,
        )
        self.assertEqual(result.returncode, expected, result.stderr)
        document = json.loads(result.stdout)
        self.assertEqual(document["protocol_version"], 1)
        self.assertEqual(document["ok"], expected == 0)
        if expected == 0:
            self.assertEqual(result.stderr, "")
        return document

    def assert_fixture(self, document: dict[str, object], name: str) -> None:
        """Compare portable response examples after normalizing temporary values."""
        portable = dict(document, revision="<revision>")
        encoded = json.dumps(portable).replace(str(self.root), "<root>")
        expected = json.loads((FIXTURES / name).read_text())
        self.assertEqual(json.loads(encoded), expected)

    def assert_no_installation(self) -> None:
        """Confirm read-only commands did not create state or provider homes."""
        for path in (
            self.toolkit / "config",
            *(self.root / name for name in ("agents", "codex", "claude")),
        ):
            self.assertFalse(path.exists(), path)

    def test_fresh_status_and_preview_match_portable_fixtures(self) -> None:
        """GIVEN no installation WHEN read-only commands run THEN fixtures match."""
        status = self.command("status", "--protocol", "1")
        preview = self.command(
            "preview", "--provider", "codex", "--skill", "commit-message"
        )

        self.assert_fixture(status, "status-fresh.json")
        self.assert_fixture(preview, "preview-install.json")
        self.assert_no_installation()

    def test_catalog_exposes_helpers_bundles_and_explicit_policies(self) -> None:
        """GIVEN the catalog WHEN queried THEN clients can resolve helper roles."""
        catalog = self.command("catalog", "--protocol", "1")

        skills = {skill["name"]: skill for skill in catalog["skills"]}
        bundles = {bundle["name"]: bundle["skills"] for bundle in catalog["bundles"]}
        self.assertEqual(skills["ticket-to-plan"]["required_helpers"], ["plan-review"])
        self.assertEqual(
            skills["plan-review"]["explicit_only"], {"codex": True, "claude": True}
        )
        self.assertEqual(bundles["starter"], ["concise-output", "commit-message"])
        self.assert_no_installation()

    def test_apply_reports_actual_inventory_and_rejects_stale_revision(self) -> None:
        """GIVEN a preview WHEN applied twice THEN the second request is stale."""
        arguments = ("--provider", "codex", "--skill", "commit-message")
        preview = self.command("preview", *arguments)
        revision = str(preview["revision"])

        applied = self.command("apply", *arguments, "--revision", revision)
        stale = self.command("apply", *arguments, "--revision", revision, expected=3)

        self.assertEqual(
            applied["applied"], {"codex": ["commit-message"], "claude": []}
        )
        self.assertNotEqual(applied["revision"], revision)
        self.assertEqual(stale["error"]["code"], "stale_revision")
        self.assertFalse((self.toolkit / "config/apply.lock").exists())

    def test_invalid_versions_and_arguments_fail_without_writes(self) -> None:
        """GIVEN unsupported or hostile input WHEN called THEN no shell code runs."""
        cases = (
            ((), 2),
            (("status", "--protocol", "99"), 5),
            (("preview", "--skill", "$(touch PWN)"), 2),
            (
                (
                    "apply",
                    "--provider",
                    "codex",
                    "--skill",
                    "commit-message",
                    "--revision",
                    "0" * 64,
                ),
                3,
            ),
        )

        for arguments, expected in cases:
            with self.subTest(arguments=arguments):
                self.command(*arguments, expected=expected)

        self.assertFalse((self.root / "PWN").exists())
        self.assert_no_installation()

    def test_status_includes_required_helpers_in_reported_overlap(self) -> None:
        """GIVEN a saved parent skill WHEN status runs THEN helper overlap remains."""
        arguments = ("--provider", "codex", "--skill", "ticket-to-plan")
        preview = self.command("preview", *arguments)
        self.command("apply", *arguments, "--revision", str(preview["revision"]))

        status = self.command(
            "status", "--native-inventory", str(FIXTURES / "codex-inventory.json")
        )

        self.assertEqual(
            status["native_overlap"]["overlaps"], ["plan-review", "ticket-to-plan"]
        )

    def test_output_paths_round_trip_as_literal_json_strings(self) -> None:
        """GIVEN quoted paths WHEN saved THEN JSON and shell handling stay literal."""
        output = str(self.root / 'outputs "quotes" \\ $(touch PWN); &\tend')
        arguments = (
            "--provider",
            "codex",
            "--skill",
            "ticket-to-plan",
            "--output-path",
            output,
        )
        preview = self.command("preview", *arguments)

        self.command("apply", *arguments, "--revision", str(preview["revision"]))
        status = self.command("status")

        self.assertEqual(status["desired"]["output_path"], output)
        self.assertFalse((self.root / "PWN").exists())

    def test_supported_native_export_reports_helper_overlap_without_mutation(
        self,
    ) -> None:
        """GIVEN an exported native inventory WHEN previewed THEN overlap is shown."""
        inventory = FIXTURES / "codex-inventory.json"

        preview = self.command(
            "preview",
            "--provider",
            "codex",
            "--skill",
            "ticket-to-plan",
            "--native-inventory",
            str(inventory),
        )

        self.assertEqual(preview["native_overlap"]["status"], "reported")
        self.assertEqual(
            preview["native_overlap"]["overlaps"], ["plan-review", "ticket-to-plan"]
        )
        self.assertEqual(
            preview["selection"],
            [
                {"name": "ticket-to-plan", "role": "requested"},
                {"name": "plan-review", "role": "required"},
            ],
        )
        self.assert_no_installation()
