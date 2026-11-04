"""Require same-run retained evidence before publication credentials can be used."""

from __future__ import annotations

import copy
import os
import sys
import unittest
from pathlib import Path
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).resolve().parent.parent / "scripts"))
import tui_publication_gate as gate
import test_tui_candidate as candidates
import test_tui_release_publication as publication_tests


class ArtifactAPI(publication_tests.FakeGitHub):
    """Expose retained artifact metadata without implementing any release writes."""

    def __init__(self, record: dict[str, object]) -> None:
        """Bind every expected artifact ID to its original run and exact name."""
        super().__init__()
        self.artifacts = {
            artifact_id: {
                "id": int(artifact_id),
                "name": name,
                "expired": False,
                "workflow_run": {"id": int(record["run_id"])},
            }
            for artifact_id, name in gate.artifact_names(record, "458").items()
        }

    def optional(self, path: str) -> dict[str, object] | None:
        """Read metadata by explicit ID without searching names or selecting latest."""
        return self.artifacts.get(path.rsplit("/", 1)[-1])


class GateTests(unittest.TestCase):
    """Exercise real ancestry and payload checks with fake artifact metadata."""

    def setUp(self) -> None:
        """Create an approved release fixture and a real trusted Git source."""
        release = publication_tests.PublicationTests()
        release.setUp()
        self.addCleanup(release.doCleanups)
        source = candidates.PreflightTests()
        source.setUp()
        self.addCleanup(source.doCleanups)
        self.record = release.record
        self.directory = release.directory
        self.root = release.fixture.root
        (self.root / "tooling").symlink_to(source.root, target_is_directory=True)
        self.record.update(source_sha=source.sha, workflow_sha=source.sha)
        for report in self.record["reports"]:
            report.update(source_sha=source.sha, workflow_sha=source.sha)
        self.api = ArtifactAPI(self.record)
        self.reports = self.root / "reports"
        self.reports.mkdir()
        for report in self.record["reports"]:
            gate.candidate.write_json(
                self.reports / f"report-{report['target']}.json", report
            )
        self.environment()
        previous = Path.cwd()
        self.addCleanup(os.chdir, previous)
        os.chdir(self.root)

    def environment(self) -> None:
        """Supply immutable job outputs and an independent current publication attempt."""
        values = {
            field.upper(): str(self.record[field]) for field in gate.candidate.IDENTITY
        }
        values.update(
            GITHUB_REPOSITORY=self.record["repository"],
            RECORD_ID="457",
            VERIFICATION_ATTEMPT="1",
            GITHUB_REF="refs/heads/main",
            DEFAULT_BRANCH="main",
            GITHUB_RUN_ATTEMPT="2",
            RESULT_ID="458",
        )
        environment = patch.dict(os.environ, values)
        environment.start()
        self.addCleanup(environment.stop)

    def check(self, record: dict[str, object] | None = None) -> None:
        """Run the complete gate and physical report comparison."""
        selected = self.record if record is None else record
        gate.check_gate(self.api, selected, self.directory, "458")
        gate.check_reports(selected, self.reports)

    def test_retained_candidate_can_be_reused_on_publication_retry(self) -> None:
        """GIVEN same-run evidence WHEN publication attempt changes THEN retain original identity."""
        self.check()

        self.assertEqual(self.record["run_attempt"], "1")
        self.assertEqual(os.environ["GITHUB_RUN_ATTEMPT"], "2")
        self.assertEqual(self.api.calls, [])

    def test_wrong_run_sha_version_or_failed_aggregate_prevents_writes(self) -> None:
        """GIVEN substituted identity or failed approval WHEN gated THEN perform zero release writes."""
        for field in (
            *gate.candidate.IDENTITY,
            "repository",
            "candidate_record_id",
            "verification_attempt",
            "status",
        ):
            with self.subTest(field=field):
                record = dict(self.record, **{field: "substituted"})

                with self.assertRaises(ValueError):
                    self.check(record)
                self.assertEqual(self.api.calls, [])

    def test_expired_missing_or_wrong_run_artifacts_fail(self) -> None:
        """GIVEN unavailable/substituted retained evidence WHEN gated THEN reject every artifact kind."""
        for artifact_id in tuple(self.api.artifacts):
            original = copy.deepcopy(self.api.artifacts[artifact_id])
            variants = (
                None,
                dict(original, expired=True),
                dict(original, name="other"),
                dict(original, workflow_run={"id": 999}),
            )
            for item in variants:
                with self.subTest(artifact_id=artifact_id, item=item):
                    self.api.artifacts[artifact_id] = item

                    with self.assertRaises(ValueError):
                        self.check()
                    self.assertEqual(self.api.calls, [])
            self.api.artifacts[artifact_id] = original

    def test_missing_native_evidence_blocks_publication(self) -> None:
        """GIVEN an incomplete aggregate WHEN gated THEN reject it before release writes."""
        record = copy.deepcopy(self.record)
        record["reports"].pop()

        with self.assertRaisesRegex(ValueError, "target reports"):
            self.check(record)
        self.assertEqual(self.api.calls, [])

    def test_physical_reports_must_equal_aggregate_snapshots(self) -> None:
        """GIVEN modified raw reports WHEN compared THEN reject copied aggregate approval."""
        path = next(self.reports.iterdir())
        report = gate.candidate.load_json(path)
        report["source_sha"] = "f" * 40
        gate.candidate.write_json(path, report)

        with self.assertRaisesRegex(ValueError, "differs from aggregate"):
            self.check()
        self.assertEqual(self.api.calls, [])

    def test_untrusted_workflow_ref_blocks_publication(self) -> None:
        """GIVEN a feature-branch workflow WHEN gated THEN reject even valid candidate bytes."""
        with patch.dict(os.environ, {"GITHUB_REF": "refs/heads/untrusted"}):
            with self.assertRaisesRegex(ValueError, "default-branch"):
                self.check()
        self.assertEqual(self.api.calls, [])

    def test_failed_gate_summary_is_unverified(self) -> None:
        """GIVEN a gate failure before publishing WHEN summarized THEN retain an unusable handoff."""
        summary = self.root / "summary.md"
        gate.candidate.write_json(
            self.root / "publication-gate.json",
            {"status": "failed", "error": "expired artifact"},
        )
        with patch.dict(os.environ, {"GITHUB_STEP_SUMMARY": str(summary)}):
            result = gate.summarize()

        self.assertEqual(result, 1)
        self.assertEqual(
            gate.candidate.load_json(self.root / "publication-report.json")["status"],
            "unverified",
        )
        self.assertIn("UNVERIFIED", summary.read_text())
        self.assertEqual(self.api.calls, [])


if __name__ == "__main__":
    unittest.main()
