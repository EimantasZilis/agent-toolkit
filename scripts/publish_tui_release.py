#!/usr/bin/env python3
"""Publish retained TUI bytes and verify draft and public downloads."""

from __future__ import annotations

import argparse
import hashlib
import json
import os
import re
import sys
import tempfile
import time
import urllib.error
import urllib.parse
import urllib.request
from http.client import HTTPMessage
from pathlib import Path
from typing import cast

import tui_candidate as candidate
import validate_tui_release as validator

WARNING = "UNVERIFIED: do not activate this release."
VERIFIED = "All six public assets verified against retained candidate bytes."


class SafeRedirect(urllib.request.HTTPRedirectHandler):
    """Allow HTTPS redirects without forwarding credentials to asset storage."""

    def redirect_request(
        self,
        request: urllib.request.Request,
        response: object,
        code: int,
        message: str,
        headers: HTTPMessage,
        url: str,
    ) -> urllib.request.Request | None:
        """Strip authentication on every redirect and refuse non-HTTPS storage."""
        require_url(url)
        redirected = super().redirect_request(
            request, response, code, message, headers, url
        )
        if redirected is not None:
            redirected.remove_header("Authorization")
        return redirected


def require_url(url: str) -> None:
    """Limit requests and redirects to HTTPS GitHub API and asset hosts."""
    parsed = urllib.parse.urlsplit(url)
    host = parsed.hostname or ""
    if (
        parsed.scheme != "https"
        or parsed.username
        or parsed.password
        or parsed.port not in (None, 443)
    ):
        raise ValueError("unsafe release URL")
    if host not in (
        "github.com",
        "api.github.com",
        "uploads.github.com",
    ) and not host.endswith(".githubusercontent.com"):
        raise ValueError("unexpected release download host")


class GitHub:
    """Provide the narrow release-service boundary used by the publisher."""

    def __init__(self, repository: str, token: str) -> None:
        """Keep credentials in memory, separate from arguments and evidence."""
        self.repository = repository
        self.token = token
        self.base = f"https://api.github.com/repos/{repository}"
        self.opener = urllib.request.build_opener(SafeRedirect())

    def request(
        self, method: str, url: str, data: bytes | None = None, binary: bool = False
    ) -> bytes:
        """Bound all response bodies and authenticate only API/upload requests."""
        require_url(url)
        headers = {
            "Accept": "application/octet-stream"
            if binary and method == "GET"
            else "application/vnd.github+json"
        }
        if urllib.parse.urlsplit(url).hostname in (
            "api.github.com",
            "uploads.github.com",
        ):
            headers["Authorization"] = "Bearer " + self.token
        if data is not None:
            headers["Content-Type"] = (
                "application/octet-stream" if binary else "application/json"
            )
        request = urllib.request.Request(url, data=data, headers=headers, method=method)
        with self.opener.open(request, timeout=30) as response:
            result = response.read(validator.MAX_SIZE + 1)
        if len(result) > validator.MAX_SIZE:
            raise ValueError("release response exceeds 64 MiB")
        return result

    def api(
        self, method: str, path: str, document: dict[str, object] | None = None
    ) -> object:
        """Decode JSON without allowing ambiguous duplicate fields."""
        data = None if document is None else json.dumps(document).encode()
        return json.loads(
            self.request(method, self.base + path, data),
            object_pairs_hook=validator.unique_object,
        )

    def optional(self, path: str) -> dict[str, object] | None:
        """Treat only a missing API object as absence; propagate service failures."""
        try:
            return cast(dict[str, object], self.api("GET", path))
        except urllib.error.HTTPError as error:
            if error.code != 404:
                raise
            return None

    def tag_commit(self, version: str) -> str | None:
        """Peel bounded annotated tag chains to their final commit."""
        reference = self.optional("/git/ref/tags/" + version)
        if reference is None:
            return None
        obj = reference["object"]
        for _ in range(10):
            if obj["type"] == "commit":
                return obj["sha"]
            if obj["type"] != "tag" or not re.fullmatch(r"[0-9a-f]{40}", obj["sha"]):
                raise ValueError("tag does not resolve to a commit")
            obj = cast(dict[str, object], self.api("GET", "/git/tags/" + obj["sha"]))[
                "object"
            ]
        raise ValueError("annotated tag nesting exceeds limit")

    def create_tag(self, version: str, sha: str) -> None:
        """Create a lightweight tag without updating any existing reference."""
        self.api("POST", "/git/refs", {"ref": "refs/tags/" + version, "sha": sha})

    def listing(self, path: str) -> list[dict[str, object]]:
        """Collect bounded paginated inventories without silently truncating."""
        result = []
        for page in range(1, 101):
            batch = cast(
                list[dict[str, object]],
                self.api("GET", f"{path}?per_page=100&page={page}"),
            )
            result.extend(batch)
            if len(batch) < 100:
                return result
        raise ValueError("release inventory exceeds pagination limit")

    def release(self, version: str) -> dict[str, object] | None:
        """Find drafts as well as public releases and reject ambiguous tags."""
        matches = [
            item for item in self.listing("/releases") if item["tag_name"] == version
        ]
        if len(matches) > 1:
            raise ValueError("duplicate releases for version")
        return matches[0] if matches else None

    def create_release(self, version: str, sha: str, body: str) -> dict[str, object]:
        """Create an explicitly unverified draft at the candidate commit."""
        return cast(
            dict[str, object],
            self.api(
                "POST",
                "/releases",
                {
                    "tag_name": version,
                    "target_commitish": sha,
                    "name": version,
                    "draft": True,
                    "body": body,
                },
            ),
        )

    def assets(self, release_id: int) -> list[dict[str, object]]:
        """Retrieve the entire inventory before comparing the exact six names."""
        return self.listing(f"/releases/{release_id}/assets")

    def upload(self, release_id: int, name: str, data: bytes) -> None:
        """Upload one allowlisted file without a replacement operation."""
        url = f"https://uploads.github.com/repos/{self.repository}/releases/{release_id}/assets?name={urllib.parse.quote(name, safe='')}"
        self.request("POST", url, data, binary=True)

    def download(self, asset_id: int, public_url: str | None = None) -> bytes:
        """Fetch draft bytes through the API or public bytes without credentials."""
        url = public_url or f"{self.base}/releases/assets/{asset_id}"
        return self.request("GET", url, binary=True)

    def update(self, release_id: int, body: str, draft: bool) -> dict[str, object]:
        """Update only this candidate's release state and verification warning."""
        return cast(
            dict[str, object],
            self.api(
                "PATCH", f"/releases/{release_id}", {"body": body, "draft": draft}
            ),
        )


def validate_candidate(
    record: dict[str, object], directory: Path, repository: str
) -> None:
    """Reject invalid repository, approval, identity, reports, or retained bytes."""
    if not re.fullmatch(
        r"[A-Za-z0-9][A-Za-z0-9_.-]*/[A-Za-z0-9][A-Za-z0-9_.-]*", repository
    ):
        raise ValueError("invalid repository")
    if record.get("repository") != repository or record.get("status") != "approved":
        raise ValueError("candidate repository or approval mismatch")
    candidate.check_payload(record, directory)
    candidate.aggregate(
        record, record["reports"], record["jobs"], record["verification_attempt"]
    )
    ids = record.get("report_artifact_ids")
    if not isinstance(ids, dict) or set(ids) != set(validator.TARGETS):
        raise ValueError("missing native report artifact identities")
    for value in [record.get("candidate_record_id"), *ids.values()]:
        if not re.fullmatch(r"[1-9][0-9]*", str(value)):
            raise ValueError("invalid retained evidence artifact ID")


def release_body(record: dict[str, object], verified: bool = False) -> str:
    """Bind release ownership to repository, source, payload identity, and hashes."""
    identity = {
        field: record[field] for field in (*candidate.IDENTITY, "repository", "hashes")
    }
    marker = (
        "<!-- agent-toolkit-candidate:"
        + json.dumps(identity, sort_keys=True, separators=(",", ":"))
        + " -->"
    )
    return marker + "\n\n" + (VERIFIED if verified else WARNING)


def require_tag(api: GitHub, record: dict[str, object], create: bool = False) -> None:
    """Require exact tag agreement, creating a missing tag only when requested."""
    sha = api.tag_commit(record["version"])
    if sha is None and create:
        api.create_tag(record["version"], record["source_sha"])
        sha = api.tag_commit(record["version"])
    if sha != record["source_sha"]:
        raise ValueError("tag/source SHA conflict")


def require_owned(release: dict[str, object] | None, record: dict[str, object]) -> None:
    """Refuse to adopt or modify a release from another candidate."""
    if not isinstance(release, dict):
        raise ValueError("release disappeared during publication")
    if release.get("tag_name") != record["version"] or release.get("body") not in (
        release_body(record),
        release_body(record, True),
    ):
        raise ValueError("existing release belongs to another candidate")
    if type(release.get("id")) is not int or release["id"] <= 0:
        raise ValueError("invalid release identity")


def inventory(
    api: GitHub, release_id: int, record: dict[str, object]
) -> dict[str, dict[str, object]]:
    """Require an exact, complete, unique six-asset inventory."""
    items = api.assets(release_id)
    names = [item["name"] for item in items]
    if len(names) != 6 or set(names) != set(record["hashes"]):
        raise ValueError(
            "release asset conflict: partial, duplicate, or unexpected inventory"
        )
    if any(
        item.get("state") != "uploaded" or type(item.get("id")) is not int
        for item in items
    ):
        raise ValueError("release asset is incomplete or lacks identity")
    return {item["name"]: item for item in items}


class Publisher:
    """Coordinate conflict checks, draft uploads, and independent verification."""

    def __init__(self, api: GitHub, record: dict[str, object], directory: Path) -> None:
        """Keep mutable external-step evidence scoped to one publication attempt."""
        self.api, self.record, self.directory = api, record, directory
        self.report: dict[str, object] = dict(
            record,
            status="unverified",
            draft="not_reached",
            public="not_reached",
            steps=[],
        )
        self.created: dict[str, object] | None = None

    def prepare(self) -> dict[str, object]:
        """Check state before any writes and never resume incomplete prior uploads."""
        existing = self.api.release(self.record["version"])
        if existing is not None:
            require_owned(existing, self.record)
            require_tag(self.api, self.record)
            inventory(self.api, existing["id"], self.record)
            return existing
        existing_sha = self.api.tag_commit(self.record["version"])
        if existing_sha is not None and existing_sha != self.record["source_sha"]:
            raise ValueError("tag/source SHA conflict")
        require_tag(self.api, self.record, create=True)
        self.report["steps"].append("tag_checked")
        if self.api.release(self.record["version"]) is not None:
            raise ValueError("release appeared during publication preflight")
        release = self.api.create_release(
            self.record["version"], self.record["source_sha"], release_body(self.record)
        )
        self.created = release
        self.identify(release)
        self.report["steps"].append("draft_created")
        self.upload(release)
        return release

    def identify(self, release: dict[str, object]) -> None:
        """Retain external identity even when a later upload or download fails."""
        self.report.update(
            release_id=release["id"],
            tag=self.record["version"],
            release_url=f"https://github.com/{self.record['repository']}/releases/tag/{self.record['version']}",
        )

    def upload(self, release: dict[str, object]) -> None:
        """Recheck owned state before uploading into an initially empty draft."""
        current = self.api.release(self.record["version"])
        require_owned(current, self.record)
        require_tag(self.api, self.record)
        if (
            current["id"] != release["id"]
            or current.get("draft") is not True
            or self.api.assets(release["id"])
        ):
            raise ValueError("draft changed before upload")
        for name, expected in self.record["hashes"].items():
            data = validator.read_file(self.directory / name)
            if hashlib.sha256(data).hexdigest() != expected:
                raise ValueError(f"{name}: retained file changed before upload")
            self.api.upload(release["id"], name, data)
            self.report["steps"].append("uploaded:" + name)

    def current(self, release: dict[str, object]) -> dict[str, object]:
        """Recheck ownership and identity before changing a release's state."""
        current = self.api.release(self.record["version"])
        require_owned(current, self.record)
        if current["id"] != release["id"]:
            raise ValueError("release changed during verification")
        inventory(self.api, current["id"], self.record)
        return current

    def fetch(self, item: dict[str, object], name: str, public: bool) -> bytes:
        """Retry only public availability errors, never checksum mismatches."""
        url = (
            f"https://github.com/{self.record['repository']}/releases/download/{self.record['version']}/{name}"
            if public
            else None
        )
        for attempt in range(3):
            try:
                return self.api.download(item["id"], url)
            except urllib.error.HTTPError as error:
                if not public or error.code not in (404, 502, 503, 504) or attempt == 2:
                    raise
                time.sleep(attempt + 1)
        raise RuntimeError("public download retry exhausted")

    def verify(self, release: dict[str, object], public: bool) -> None:
        """Compare every download with retained hashes and the original manifest."""
        stage = "public" if public else "draft"
        items = inventory(self.api, release["id"], self.record)
        self.report[stage + "_hashes"] = {}
        with tempfile.TemporaryDirectory(prefix="tui-published-") as temporary:
            directory = Path(temporary)
            for name, expected in self.record["hashes"].items():
                try:
                    data = self.fetch(items[name], name, public)
                except OSError as error:
                    raise ValueError(
                        f"{stage} {name}: download failed: {error}"
                    ) from error
                if len(data) > validator.MAX_SIZE:
                    raise ValueError(f"{stage} {name}: download exceeds 64 MiB")
                actual = hashlib.sha256(data).hexdigest()
                self.report[stage + "_hashes"][name] = {
                    "expected": expected,
                    "actual": actual,
                }
                if actual != expected:
                    raise ValueError(f"{stage} {name}: download sha256 mismatch")
                (directory / name).write_bytes(data)
            validator.validate_release(
                self.record["version"], directory, self.directory / "release.json"
            )
        self.report[stage] = "passed"
        self.report["steps"].append(stage + "_verified")

    def run(self) -> dict[str, object]:
        """Return successful evidence only after both download stages pass."""
        try:
            validate_candidate(self.record, self.directory, self.api.repository)
            release = self.prepare()
            self.identify(release)
            self.verify(release, public=False)
            require_tag(self.api, self.record)
            release = self.current(release)
            if release["draft"]:
                release = self.api.update(
                    release["id"], release_body(self.record), draft=False
                )
                self.report["steps"].append("published_unverified")
            self.verify(release, public=True)
            require_tag(self.api, self.record)
            release = self.current(release)
            if release.get("body") != release_body(self.record, True):
                self.api.update(
                    release["id"], release_body(self.record, True), draft=False
                )
            self.report["status"] = "verified"
        except (OSError, ValueError, KeyError, TypeError, RuntimeError) as error:
            self.report["status"] = "unverified"
            self.report["error"] = str(error)
            self.retain_warning()
        return self.report

    def retain_warning(self) -> None:
        """Best-effort warning updates are limited to releases created in this call."""
        if self.created is None:
            return
        try:
            current = self.api.release(self.record["version"])
            require_owned(current, self.record)
            if current["id"] != self.created["id"]:
                raise ValueError("release identity changed; warning not updated")
            self.api.update(current["id"], release_body(self.record), current["draft"])
        except (OSError, ValueError, KeyError, TypeError) as error:
            self.report["warning_error"] = str(error)


def main() -> int:
    """Publish an approved retained candidate and always write unverified failures."""
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--candidate-directory", type=Path, required=True)
    parser.add_argument("--candidate-record", type=Path, required=True)
    parser.add_argument("--repository", required=True)
    parser.add_argument("--verification-report", type=Path, required=True)
    args = parser.parse_args()
    report = {"status": "unverified"}
    try:
        record = candidate.load_json(args.candidate_record)
        api = GitHub(args.repository, os.environ["GH_TOKEN"])
        report = Publisher(api, record, args.candidate_directory).run()
    except (OSError, ValueError, KeyError) as error:
        report["error"] = str(error)
    candidate.write_json(args.verification_report, report)
    print("TUI publication: " + report["status"])
    return int(report["status"] != "verified")


if __name__ == "__main__":
    sys.exit(main())
