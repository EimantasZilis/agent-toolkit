#!/usr/bin/env python3
"""Check retained candidate evidence before a publication job may mutate releases."""

from __future__ import annotations

import argparse
import json
import os
import re
import subprocess
import sys
from pathlib import Path

import publish_tui_release as publication
import tui_candidate as candidate


def check_context(record: dict[str, object]) -> None:
    """Bind the approved candidate to trusted outputs from this workflow run."""
    candidate.require_context(record)
    expected = {
        "repository": os.environ["GITHUB_REPOSITORY"],
        "candidate_record_id": os.environ["RECORD_ID"],
        "verification_attempt": os.environ["VERIFICATION_ATTEMPT"],
    }
    if any(record.get(field) != value for field, value in expected.items()):
        raise ValueError("publication candidate differs from trusted run outputs")
    if os.environ["GITHUB_REF"] != "refs/heads/" + os.environ["DEFAULT_BRANCH"]:
        raise ValueError("publication requires the trusted default-branch workflow")


def artifact_names(record: dict[str, object], result_id: str) -> dict[str, str]:
    """Name every exact artifact required for this candidate and verification attempt."""
    original = record["run_attempt"]
    verification = record["verification_attempt"]
    names = {
        record["payload_id"]: f"tui-payload-{original}",
        record["candidate_record_id"]: f"tui-candidate-{original}",
        result_id: f"tui-result-{verification}",
    }
    for target, artifact_id in record["report_artifact_ids"].items():
        names[artifact_id] = f"tui-report-{target}-{verification}"
    if len(names) != 7:
        raise ValueError("duplicate retained artifact identities")
    return names


def check_artifacts(
    api: publication.GitHub, record: dict[str, object], result_id: str
) -> None:
    """Require seven non-expired artifacts from this exact workflow run."""
    for artifact_id, name in artifact_names(record, result_id).items():
        if not re.fullmatch(r"[1-9][0-9]*", str(artifact_id)):
            raise ValueError("invalid publication artifact ID")
        item = api.optional(f"/actions/artifacts/{artifact_id}")
        if item is None or item.get("expired") is not False or item.get("name") != name:
            raise ValueError(
                f"missing, expired, or substituted artifact: {artifact_id} ({name})"
            )
        if (
            str(item.get("id")) != str(artifact_id)
            or str(item.get("workflow_run", {}).get("id")) != record["run_id"]
        ):
            raise ValueError(f"artifact belongs to another workflow run: {artifact_id}")


def check_reports(record: dict[str, object], directory: Path) -> None:
    """Compare original downloaded reports with the aggregate's retained snapshots."""
    expected = {
        f"report-{report['target']}.json": report for report in record["reports"]
    }
    paths = list(directory.iterdir())
    if {path.name for path in paths} != set(expected):
        raise ValueError("missing or unexpected native report files")
    for path in paths:
        if candidate.load_json(path) != expected[path.name]:
            raise ValueError(f"{path.name}: raw runtime report differs from aggregate")


def check_gate(
    api: publication.GitHub, record: dict[str, object], directory: Path, result_id: str
) -> None:
    """Check inputs, payload, current artifact availability, and trusted Git ancestry."""
    check_context(record)
    publication.validate_candidate(record, directory, api.repository)
    check_artifacts(api, record, result_id)
    resolved = candidate.preflight(
        Path("tooling"),
        record["version"],
        record["source_sha"],
        record["workflow_sha"],
        os.environ["DEFAULT_BRANCH"],
    )
    if resolved != record["source_sha"]:
        raise ValueError("publication source ancestry mismatch")


def summarize() -> int:
    """Retain an explicitly unverified report if a gate prevented publication."""
    path = Path("publication-report.json")
    if path.exists():
        report = candidate.load_json(path)
    else:
        report = {
            "status": "unverified",
            "error": "publication did not reach verification",
        }
        if Path("publication-gate.json").exists():
            report["gate"] = candidate.load_json(Path("publication-gate.json"))
        candidate.write_json(path, report)
    label = (
        "VERIFIED"
        if report.get("status") == "verified"
        else "UNVERIFIED — activation blocked"
    )
    with Path(os.environ["GITHUB_STEP_SUMMARY"]).open("a") as summary:
        summary.write(
            f"## TUI publication: {label}\n\n```json\n"
            + json.dumps(report, indent=2)
            + "\n```\n"
        )
    return int(report.get("status") != "verified")


def main() -> int:
    """Resolve report IDs read-only or repeat all checks immediately before writes."""
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("command", choices=("prepare", "check", "summary"))
    args = parser.parse_args()
    if args.command == "summary":
        return summarize()
    try:
        record = candidate.load_json(Path("candidate-result.json"))
        api = publication.GitHub(
            os.environ["GITHUB_REPOSITORY"], os.environ["GH_TOKEN"]
        )
        check_gate(api, record, Path("payload"), os.environ["RESULT_ID"])
        if args.command == "check":
            check_reports(record, Path("reports"))
        else:
            ids = ",".join(record["report_artifact_ids"].values())
            with Path(os.environ["GITHUB_OUTPUT"]).open("a") as output:
                output.write("report_ids=" + ids + "\n")
        candidate.write_json(Path("publication-gate.json"), {"status": "passed"})
    except (
        OSError,
        ValueError,
        KeyError,
        TypeError,
        subprocess.SubprocessError,
    ) as error:
        candidate.write_json(
            Path("publication-gate.json"), {"status": "failed", "error": str(error)}
        )
        print(f"TUI publication gate failed: {error}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
