#!/usr/bin/env python3
"""Bind native pin reports and proposal writes to exact same-run retained artifacts."""

from __future__ import annotations

import argparse
import json
import os
import re
import subprocess
import sys
from pathlib import Path

import check_tui_release_pin as pin
import propose_tui_release_pin as proposal
import tui_candidate as candidate
import tui_publication_gate as publication_gate


def artifact(
    api: proposal.GitHub, record: dict[str, object], artifact_id: str, name: str
) -> None:
    """Reject expired, foreign-run, renamed, or missing downstream evidence artifacts."""
    if not re.fullmatch(r"[1-9][0-9]*", artifact_id):
        raise ValueError("invalid pin evidence artifact ID")
    item = api.optional("/actions/artifacts/" + artifact_id)
    if (
        item is None
        or item.get("expired") is not False
        or item.get("name") != name
        or str(item.get("id")) != artifact_id
        or str(item.get("workflow_run", {}).get("id")) != record["run_id"]
    ):
        raise ValueError(
            "pin artifact is unavailable or differs from trusted job outputs"
        )


def retained_context(
    api: proposal.GitHub,
) -> tuple[dict[str, object], dict[str, object]]:
    """Repeat the original read-only candidate gate plus exact publication identity."""
    record = candidate.load_json(Path("candidate-result.json"))
    report = candidate.load_json(Path("publication-report.json"))
    publication_gate.check_gate(api, record, Path("payload"), os.environ["RESULT_ID"])
    pin.check_publication(record, report, Path("payload"))
    if report.get("publication_attempt") != os.environ["PUBLICATION_ATTEMPT"]:
        raise ValueError("publication belongs to another attempt")
    artifact(
        api,
        record,
        os.environ["PUBLICATION_ID"],
        "tui-publication-" + os.environ["PUBLICATION_ATTEMPT"],
    )
    return record, report


def output(key: str, value: str) -> None:
    """Write trusted helper outputs for dependent workflow jobs."""
    with Path(os.environ["GITHUB_OUTPUT"]).open("a") as stream:
        stream.write(f"{key}={value}\n")


def prepare(api: proposal.GitHub) -> None:
    """Resolve a proposal base only after publication/candidate gates pass read-only."""
    record, _ = retained_context(api)
    if os.environ["PROPOSE_PIN"] != "true" or os.environ["PUBLISH"] != "true":
        raise ValueError("proposal requires explicit publish and propose inputs")
    sha = api.default_base(os.environ["DEFAULT_BRANCH"])
    candidate.git(
        Path("tooling"), "merge-base", "--is-ancestor", record["source_sha"], sha
    )
    output("base_sha", sha)


def native() -> None:
    """Run source/base acceptance without API credentials and retain attempt identity."""
    record = candidate.load_json(Path("candidate-result.json"))
    candidate.require_context(record)
    report = candidate.load_json(Path("publication-report.json"))
    repository = os.environ["GITHUB_REPOSITORY"]
    result = pin.check_pin(
        record,
        report,
        Path("payload"),
        Path("source").resolve(),
        Path("proposal-base").resolve(),
        os.environ["BASE_SHA"],
        os.environ["TARGET"],
        test_repository=repository if repository != pin.PRODUCTION_REPOSITORY else None,
    )
    result["launcher_attempt"] = os.environ["GITHUB_RUN_ATTEMPT"]
    candidate.write_json(Path("pin-" + os.environ["TARGET"] + ".json"), result)


def resolve_reports(api: proposal.GitHub) -> dict[str, str]:
    """Find exactly one current-attempt native pin artifact per target in this run."""
    attempt = os.environ["GITHUB_RUN_ATTEMPT"]
    items = run_artifacts(api)
    ids = {}
    for target in pin.validator.TARGETS:
        name = f"tui-pin-{target}-{attempt}"
        matches = [item for item in items if item.get("name") == name]
        if len(matches) != 1 or matches[0].get("expired") is not False:
            raise ValueError(
                "missing, duplicate, or expired native pin artifact: " + name
            )
        artifact_id = str(matches[0]["id"])
        artifact(api, {"run_id": os.environ["GITHUB_RUN_ID"]}, artifact_id, name)
        ids[target] = artifact_id
    return ids


def run_artifacts(api: proposal.GitHub) -> list[dict[str, object]]:
    """Read the Actions artifact envelope with bounded pagination."""
    result = []
    for page in range(1, 101):
        batch = api.api(
            "GET",
            f"/actions/runs/{os.environ['GITHUB_RUN_ID']}/artifacts?per_page=100&page={page}",
        )["artifacts"]
        result.extend(batch)
        if len(batch) < 100:
            return result
    raise ValueError("pin artifact inventory exceeds pagination limit")


def reports(
    record: dict[str, object], ids: dict[str, str], api: proposal.GitHub
) -> list[dict[str, object]]:
    """Require the complete exact report file set and recheck its retained identities."""
    if set(ids) != set(pin.validator.TARGETS) or len(set(ids.values())) != 4:
        raise ValueError("missing or duplicate pin report artifact IDs")
    expected = {f"pin-{target}.json" for target in pin.validator.TARGETS}
    if {path.name for path in Path("pin-reports").iterdir()} != expected:
        raise ValueError("missing or unexpected native pin report files")
    for target, artifact_id in ids.items():
        artifact(
            api,
            record,
            artifact_id,
            f"tui-pin-{target}-{os.environ['GITHUB_RUN_ATTEMPT']}",
        )
    return [
        candidate.load_json(Path("pin-reports") / name) for name in sorted(expected)
    ]


def require_jobs(jobs: dict[str, object]) -> None:
    """Reject skipped/failed/missing prerequisites rather than trusting a partial matrix."""
    required = {"pin-preflight", "launcher", "publication"}
    if not required.issubset(jobs) or any(
        item.get("result") != "success" for item in jobs.values()
    ):
        raise ValueError("required publication or native launcher job did not succeed")


def aggregate(api: proposal.GitHub) -> None:
    """Approve only matching reports and a still-current proposal base."""
    require_jobs(json.loads(os.environ["PIN_NEEDS"]))
    record, report = retained_context(api)
    ids = candidate.load_json(Path("pin-report-ids.json"))
    native_reports = reports(record, ids, api)
    base = os.environ["BASE_SHA"]
    if api.default_base(os.environ["DEFAULT_BRANCH"]) != base:
        raise ValueError("proposal base moved during native checks")
    proposal.check_launcher_reports(
        record,
        report,
        (Path("payload") / "release.json").read_bytes(),
        base,
        native_reports,
        os.environ["GITHUB_RUN_ATTEMPT"],
    )
    result = {
        "status": "approved",
        "base_sha": base,
        "reports": native_reports,
        "report_artifact_ids": ids,
        "jobs": json.loads(os.environ["PIN_NEEDS"]),
    }
    candidate.write_json(Path("pin-result.json"), result)
    with Path(os.environ["GITHUB_STEP_SUMMARY"]).open("a") as summary:
        summary.write(
            "## TUI pin launcher acceptance\n\n```json\n"
            + json.dumps(result, indent=2)
            + "\n```\n\nManual UI review and merge remain pending.\n"
        )


def write(api: proposal.GitHub) -> None:
    """Recheck same-run raw evidence and base before handing the token to the proposer."""
    record, report = retained_context(api)
    acceptance = candidate.load_json(Path("pin-result.json"))
    require_jobs(acceptance["jobs"])
    artifact(
        api,
        record,
        os.environ["PIN_RESULT_ID"],
        "tui-pin-result-" + os.environ["GITHUB_RUN_ATTEMPT"],
    )
    native_reports = reports(record, acceptance["report_artifact_ids"], api)
    if (
        acceptance.get("status") != "approved"
        or native_reports != acceptance["reports"]
    ):
        raise ValueError("raw native pin reports differ from approved aggregate")
    manifest = pin.check_publication(record, report, Path("payload"))
    proposal.check_launcher_reports(
        record,
        report,
        manifest,
        acceptance["base_sha"],
        native_reports,
        os.environ["GITHUB_RUN_ATTEMPT"],
    )
    result = proposal.Proposer(
        api,
        record,
        report,
        manifest,
        acceptance["base_sha"],
        os.environ["DEFAULT_BRANCH"],
        native_reports,
        evidence_links(acceptance),
    ).run()
    summarize_proposal(result)


def evidence_links(acceptance: dict[str, object]) -> dict[str, str]:
    """Link the exact reports used by the gate for maintainer inspection."""
    return dict(
        acceptance["report_artifact_ids"],
        publication=os.environ["PUBLICATION_ID"],
        launcher_result=os.environ["PIN_RESULT_ID"],
    )


def summarize_proposal(result: dict[str, object]) -> None:
    """Retain the proposal link with manual review requirements still pending."""
    candidate.write_json(Path("proposal-report.json"), result)
    with Path(os.environ["GITHUB_STEP_SUMMARY"]).open("a") as summary:
        summary.write(
            "## TUI pin proposal\n\n"
            + json.dumps(result)
            + "\n\nManual UI review, required checks, approval, and merge remain pending.\n"
        )


def execute(command: str) -> None:
    """Keep native execution separate from all API credential use."""
    if command == "native":
        native()
        return
    api = proposal.GitHub(os.environ["GITHUB_REPOSITORY"], os.environ["GH_TOKEN"])
    if command == "prepare":
        prepare(api)
    elif command == "reports":
        ids = resolve_reports(api)
        candidate.write_json(Path("pin-report-ids.json"), ids)
        output("ids", ",".join(ids.values()))
    elif command == "aggregate":
        aggregate(api)
    else:
        write(api)


def main() -> int:
    """Persist explicit failure diagnostics without producing approval on exceptions."""
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "command", choices=("prepare", "native", "reports", "aggregate", "write")
    )
    args = parser.parse_args()
    try:
        execute(args.command)
    except (
        OSError,
        ValueError,
        KeyError,
        TypeError,
        subprocess.SubprocessError,
    ) as error:
        candidate.write_json(
            Path("pin-failure.json"), {"status": "failed", "error": str(error)}
        )
        print(f"TUI pin {args.command} failed: {error}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
