#!/usr/bin/env python3
"""Validate BeforeUp CI statuses, JSON and redaction on a native Linux Docker host."""
import argparse
import json
import subprocess
import sys
import tempfile
from pathlib import Path


def run(binary, config, *options):
    p = subprocess.run(
        [str(binary), "check", "-f", str(config), "--format", "json", *options],
        capture_output=True, text=True, timeout=30, check=False,
    )
    try:
        report = json.loads(p.stdout)
    except json.JSONDecodeError as exc:
        raise AssertionError("BeforeUp did not emit valid JSON") from exc
    assert isinstance(report.get("findings"), list), "Missing findings array"
    assert isinstance(report.get("summary"), dict), "Missing summary"
    for key in ("blockers", "warnings", "passed"):
        assert isinstance(report["summary"].get(key), int), key
    return p.returncode, report


def verify(binary):
    with tempfile.TemporaryDirectory(prefix="beforeup-ci-contract-") as folder:
        work = Path(folder)
        secret_name = "BEFOREUP_CI_SECRET_NAME"
        secret_value = "beforeup-ci-secret-value-should-never-leak"
        valid = work / "valid.yaml"
        valid.write_text(
            "services:\n  web:\n    image: nginx:alpine\n"
            "    environment:\n"
            f"      {secret_name}: {secret_value}\n", encoding="utf-8",
        )
        code, clean = run(binary, valid, "--min-free-gib", "0")
        assert code == 0 and clean["summary"]["blockers"] == 0
        assert clean["summary"]["warnings"] == 0

        options = ("--min-free-gib", "999999")
        code, warning = run(binary, valid, *options)
        assert code == 0 and warning["summary"]["warnings"] > 0
        assert warning["summary"]["blockers"] == 0
        assert any(f["id"] == "DISK_LOW" for f in warning["findings"])

        code, strict = run(binary, valid, *options, "--strict")
        assert code == 1, "Strict warnings must exit 1"
        assert strict == warning, "Strict mode must not change finding severities"

        missing_source = work / "private-missing-bind"
        assert not missing_source.exists()
        blocked = work / "blocked.yaml"
        blocked.write_text(
            "services:\n  web:\n    image: nginx:alpine\n"
            "    volumes:\n      - type: bind\n"
            f"        source: {missing_source}\n"
            "        target: /data\n"
            "        bind:\n          create_host_path: false\n",
            encoding="utf-8",
        )
        code, blocker = run(binary, blocked, "--min-free-gib", "0")
        assert code == 1 and blocker["summary"]["blockers"] > 0
        assert any(f["id"] == "BIND_MISSING" for f in blocker["findings"])
        assert not missing_source.exists(), "The missing directory was created"

        invalid = work / "invalid.yaml"
        invalid.write_text("services: [invalid-service-map]\n", encoding="utf-8")
        code, failure = run(binary, invalid)
        assert code == 2 and any(f["id"] == "COMPOSE" for f in failure["findings"])

        for report in (clean, warning, strict, blocker, failure):
            content = json.dumps(report)
            for private in (secret_name, secret_value, str(missing_source)):
                assert private not in content, "Sensitive value or path in report"


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--binary", type=Path, required=True)
    args = parser.parse_args()
    binary = args.binary.expanduser().resolve()
    if not binary.is_file():
        parser.error("BeforeUp executable does not exist")
    try:
        verify(binary)
    except (AssertionError, OSError, subprocess.TimeoutExpired) as exc:
        print(f"ci_contract: FAIL: {exc}", file=sys.stderr)
        return 1
    print("ci_contract: PASS (clean, warning, strict, blocker, invalid, JSON redaction)")
    return 0


if __name__ == "__main__":
    sys.exit(main())
