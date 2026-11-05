"""Check publication safety and byte equality through a fake GitHub boundary."""

from __future__ import annotations

import copy
import io
import sys
import unittest
import urllib.error
import urllib.request
from http.client import HTTPMessage
from pathlib import Path
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).resolve().parent.parent / "scripts"))
import publish_tui_release as publication
import test_tui_release as releases


class FakeGitHub(publication.GitHub):
    """Retain external state and an exact mutation log without network access."""

    def __init__(self) -> None:
        """Start with an empty release service and configurable failure points."""
        self.repository = "fixture/toolkit"
        self.tag: str | None = None
        self.annotated = False
        self.state: dict[str, object] | None = None
        self.files: dict[str, bytes] = {}
        self.calls: list[tuple] = []
        self.fail_upload: str | None = None
        self.corrupt: tuple[bool, str] | None = None
        self.delays = 0
        self.fail_update = False

    def api(
        self, method: str, path: str, document: dict[str, object] | None = None
    ) -> object:
        """Supply lightweight or annotated tag objects to the real peel logic."""
        if path.startswith("/git/ref/"):
            if self.tag is None:
                raise urllib.error.HTTPError(path, 404, "missing", {}, None)
            return {
                "object": {
                    "type": "tag" if self.annotated else "commit",
                    "sha": "c" * 40 if self.annotated else self.tag,
                }
            }
        if path.startswith("/git/tags/"):
            return {"object": {"type": "commit", "sha": self.tag}}
        raise AssertionError((method, path, document))

    def create_tag(self, version: str, sha: str) -> None:
        """Record tag creation, refusing any replacement."""
        assert self.tag is None
        self.calls.append(("create_tag", version, sha))
        self.tag = sha

    def release(self, version: str) -> dict[str, object] | None:
        """Return a snapshot so state changes cannot mutate previous evidence."""
        return copy.deepcopy(self.state)

    def create_release(self, version: str, sha: str, body: str) -> dict[str, object]:
        """Create one owned draft, never an unrelated release."""
        assert self.state is None
        self.calls.append(("create_release", version, sha))
        self.state = {"id": 42, "tag_name": version, "draft": True, "body": body}
        return copy.deepcopy(self.state)

    def assets(self, release_id: int) -> list[dict[str, object]]:
        """Expose exact upload inventory with stable asset IDs."""
        assert release_id == 42
        return [
            {"id": index + 1, "name": name, "state": "uploaded"}
            for index, name in enumerate(self.files)
        ]

    def upload(self, release_id: int, name: str, data: bytes) -> None:
        """Reject clobbers and optionally fail partway through uploads."""
        assert release_id == 42 and name not in self.files
        self.calls.append(("upload", name))
        if name == self.fail_upload:
            raise OSError("injected upload failure")
        self.files[name] = data

    def download(self, asset_id: int, public_url: str | None = None) -> bytes:
        """Return retained bytes, corruption, or bounded public availability errors."""
        public = public_url is not None
        name = list(self.files)[asset_id - 1]
        if public and self.delays:
            self.delays -= 1
            raise urllib.error.HTTPError(public_url, 404, "pending", {}, None)
        if self.corrupt == (public, name):
            return b"substituted bytes"
        return self.files[name]

    def update(self, release_id: int, body: str, draft: bool) -> dict[str, object]:
        """Record state changes for the owned release or simulate API failure."""
        assert release_id == 42
        self.calls.append(("update", draft, body))
        if self.fail_update:
            raise OSError("injected update failure")
        self.state.update(body=body, draft=draft)
        return copy.deepcopy(self.state)


class PublicationTests(unittest.TestCase):
    """Exercise publication and retries using complete inert release archives."""

    def setUp(self) -> None:
        """Bind a validated six-file fixture to four successful native reports."""
        self.fixture = releases.ReleaseTests()
        self.fixture.setUp()
        self.addCleanup(self.fixture.doCleanups)
        self.directory = self.fixture.directory
        self.api = FakeGitHub()
        self.record = {
            "version": releases.VERSION,
            "source_sha": "a" * 40,
            "workflow_sha": "b" * 40,
            "run_id": "123",
            "run_attempt": "1",
            "payload_id": "456",
            "candidate_record_id": "457",
            "repository": self.api.repository,
            "status": "approved",
            "verification_attempt": "1",
            "jobs": {name: "success" for name in ("preflight", "build", "runtime")},
            "hashes": publication.candidate.payload_hashes(
                self.directory, releases.VERSION
            ),
            "report_artifact_ids": {
                target: str(index + 10)
                for index, target in enumerate(publication.validator.TARGETS)
            },
        }
        self.record["reports"] = [
            dict(
                {field: self.record[field] for field in publication.candidate.IDENTITY},
                target=target,
                verification_attempt="1",
                checks={
                    name: {"status": "passed", "reason": "fixture native evidence"}
                    for name in publication.candidate.CHECKS
                },
            )
            for target in publication.validator.TARGETS
        ]

    def publish(self) -> dict[str, object]:
        """Run the real publication orchestration with the fake external service."""
        return publication.Publisher(self.api, self.record, self.directory).run()

    def seed_release(self, draft: bool = True) -> None:
        """Create an existing complete release owned by this same candidate."""
        self.api.tag = self.record["source_sha"]
        self.api.create_release(
            self.record["version"],
            self.record["source_sha"],
            publication.release_body(self.record, not draft),
        )
        self.api.state["draft"] = draft
        self.api.files = {
            name: (self.directory / name).read_bytes() for name in self.record["hashes"]
        }
        self.api.calls.clear()

    def test_new_release_verifies_draft_and_public_bytes(self) -> None:
        """GIVEN checked bytes WHEN published THEN both stages match retained hashes."""
        result = self.publish()

        self.assertEqual(result["status"], "verified", result)
        self.assertEqual(result["draft"], "passed")
        self.assertEqual(result["public"], "passed")
        self.assertEqual(
            len([call for call in self.api.calls if call[0] == "upload"]), 6
        )
        self.assertFalse(self.api.state["draft"])
        self.assertIn(publication.VERIFIED, self.api.state["body"])
        self.assertEqual(self.api.tag, self.record["source_sha"])

    def test_existing_tag_matches_and_annotated_tag_peeling(self) -> None:
        """GIVEN matching lightweight/annotated tags WHEN publishing THEN never move them."""
        for annotated in (False, True):
            with self.subTest(annotated=annotated):
                self.api = FakeGitHub()
                self.api.tag = self.record["source_sha"]
                self.api.annotated = annotated

                result = self.publish()

                self.assertEqual(result["status"], "verified", result)
                self.assertFalse(
                    any(call[0] == "create_tag" for call in self.api.calls)
                )

    def test_conflicting_tag_or_release_causes_zero_writes(self) -> None:
        """GIVEN conflicting ownership or commit WHEN checked THEN no external mutation."""
        self.api.tag = "f" * 40

        result = self.publish()

        self.assertEqual(result["status"], "unverified")
        self.assertEqual(self.api.calls, [])
        self.seed_release()
        self.api.state["body"] = "unrelated release"

        result = self.publish()

        self.assertEqual(result["status"], "unverified")
        self.assertEqual(self.api.calls, [])

    def test_exact_public_retry_only_reverifies(self) -> None:
        """GIVEN a complete verified public release WHEN retried THEN perform no writes."""
        self.seed_release(draft=False)

        result = self.publish()

        self.assertEqual(result["status"], "verified", result)
        self.assertEqual(self.api.calls, [])

    def test_complete_owned_draft_can_finish_publication(self) -> None:
        """GIVEN an owned complete draft WHEN resumed THEN verify and expose existing bytes."""
        self.seed_release()

        result = self.publish()

        self.assertEqual(result["status"], "verified", result)
        self.assertTrue(all(call[0] == "update" for call in self.api.calls))

    def test_partial_prior_upload_is_not_resumed(self) -> None:
        """GIVEN a partial draft WHEN retried THEN stop without uploads or edits."""
        self.seed_release()
        self.api.files.pop("release.json")

        result = self.publish()

        self.assertEqual(result["status"], "unverified")
        self.assertIn("partial", result["error"])
        self.assertEqual(self.api.calls, [])

    def test_duplicate_assets_and_disappearing_release_fail_safely(self) -> None:
        """GIVEN conflicting inventory or a vanished release WHEN resumed THEN never edit it."""
        self.seed_release()
        items = self.api.assets(42)
        items[-1] = items[0]
        with patch.object(self.api, "assets", return_value=items):
            result = self.publish()

        self.assertEqual(result["status"], "unverified")
        self.assertEqual(self.api.calls, [])
        with self.assertRaisesRegex(ValueError, "disappeared"):
            publication.require_owned(None, self.record)

    def test_authenticated_download_error_keeps_draft_unverified(self) -> None:
        """GIVEN an API download failure WHEN checking draft bytes THEN never expose the draft."""
        with patch.object(
            self.api, "download", side_effect=OSError("download unavailable")
        ):
            result = self.publish()

        self.assertEqual(result["status"], "unverified")
        self.assertTrue(self.api.state["draft"])
        self.assertIn("download failed", result["error"])
        self.assertIn(publication.WARNING, self.api.state["body"])

    def test_local_substitutions_prevent_all_service_calls(self) -> None:
        """GIVEN changed manifest/checksums/archive WHEN checked THEN fail before writes."""
        for name in (
            "release.json",
            "SHA256SUMS",
            next(name for name in self.record["hashes"] if name.endswith(".zip")),
        ):
            with self.subTest(name=name):
                path = self.directory / name
                retained = path.read_bytes()
                path.write_bytes(b"substituted")

                result = self.publish()

                self.assertEqual(result["status"], "unverified")
                self.assertIn(name, result["error"])
                self.assertEqual(self.api.calls, [])
                path.write_bytes(retained)

    def test_download_substitution_fails_with_asset_identity(self) -> None:
        """GIVEN altered draft/public downloads WHEN verified THEN retain unverified status."""
        for public in (False, True):
            for name in ("release.json", "SHA256SUMS"):
                with self.subTest(public=public, name=name):
                    self.api = FakeGitHub()
                    self.api.corrupt = (public, name)

                    result = self.publish()

                    self.assertEqual(result["status"], "unverified")
                    self.assertIn(name, result["error"])
                    self.assertIn(publication.WARNING, self.api.state["body"])

    def test_upload_and_update_failures_never_produce_handoff(self) -> None:
        """GIVEN API failure WHEN upload/publication runs THEN keep release identity and warning."""
        self.api.fail_upload = "SHA256SUMS"

        result = self.publish()

        self.assertEqual(result["status"], "unverified")
        self.assertEqual(result["release_id"], 42)
        self.assertIn("draft_created", result["steps"])
        self.api = FakeGitHub()
        self.api.fail_update = True

        result = self.publish()

        self.assertEqual(result["status"], "unverified")
        self.assertIn("warning_error", result)

    def test_public_availability_retries_are_bounded(self) -> None:
        """GIVEN delayed public bytes WHEN fetched THEN retry availability only, within bounds."""
        for delays, expected in ((2, "verified"), (3, "unverified")):
            with self.subTest(delays=delays), patch.object(publication.time, "sleep"):
                self.api = FakeGitHub()
                self.api.delays = delays

                result = self.publish()

                self.assertEqual(result["status"], expected, result)

    def test_wrong_repository_or_failed_candidate_never_writes(self) -> None:
        """GIVEN wrong repository/failed aggregate WHEN publishing THEN no service mutation."""
        for field, value in (
            ("repository", "other/repo"),
            ("status", "failed"),
            ("source_sha", "bad"),
        ):
            with self.subTest(field=field):
                record = dict(self.record, **{field: value})

                result = publication.Publisher(self.api, record, self.directory).run()

                self.assertEqual(result["status"], "unverified")
                self.assertEqual(self.api.calls, [])


class TransportTests(unittest.TestCase):
    """Check response limits and credential handling at the HTTP boundary."""

    def test_redirects_strip_credentials_and_reject_untrusted_hosts(self) -> None:
        """GIVEN an authenticated API request WHEN redirected THEN credentials stay private."""
        request = urllib.request.Request(
            "https://api.github.com/repos/fixture/toolkit/releases/assets/1",
            headers={"Authorization": "Bearer fixture-secret"},
        )
        handler = publication.SafeRedirect()

        redirected = handler.redirect_request(
            request,
            None,
            302,
            "found",
            HTTPMessage(),
            "https://release-assets.githubusercontent.com/fixture",
        )

        self.assertFalse(redirected.has_header("Authorization"))
        for url in (
            "http://github.com/asset",
            "https://evil.example/asset",
            "https://user:pass@github.com/asset",
        ):
            with self.assertRaises(ValueError):
                handler.redirect_request(
                    request, None, 302, "found", HTTPMessage(), url
                )

    def test_download_bound_rejects_excess_response_bytes(self) -> None:
        """GIVEN excessive network output WHEN read THEN enforce the download bound."""
        api = publication.GitHub("fixture/toolkit", "secret")
        with (
            patch.object(api.opener, "open", return_value=io.BytesIO(b"12345")),
            patch.object(publication.validator, "MAX_SIZE", 4),
        ):
            with self.assertRaisesRegex(ValueError, "exceeds"):
                api.download(1)


if __name__ == "__main__":
    unittest.main()
