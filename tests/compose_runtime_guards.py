#!/usr/bin/env python3
"""Read-only Compose CLI regression for runtime mismatches on four tested Compose versions."""
import argparse
import json
import subprocess
import tempfile
from pathlib import Path

def call(*args):
    return subprocess.run(args, capture_output=True, text=True, timeout=25, check=False)

def run(binary):
    version = call("docker", "compose", "version", "--short")
    if version.returncode or version.stdout.strip().lstrip("v") not in ("5.5.0", "5.5.1", "5.6.0", "5.6.1"):
        print("compose_runtime_guards: SKIP (Compose version not independently reproduced)")
        return
    fixtures = {
        "pre-start-unsupported": (
            "services:\n  app:\n    image: nginx:stable-alpine\n"
            "    pre_start:\n      - image: nginx:stable-alpine\n"
            "        per_replica: true\n    deploy:\n      replicas: 3\n",
            "PRE_START_REPLICA_UNSUPPORTED",
        ),
        "restart-silent": (
            "services:\n  app:\n    image: nginx:stable-alpine\n"
            "    restart: always\n    deploy:\n      restart_policy:\n        delay: 5s\n"
            "    environment:\n      TOKEN: secret-must-never-appear\n",
            "RESTART_POLICY_OVERRIDDEN",
        ),
        "restart-rejected": (
            "services:\n  app:\n    image: nginx:stable-alpine\n"
            "    restart: always\n    deploy:\n      restart_policy:\n        max_attempts: 3\n",
            "RESTART_POLICY_INVALID",
        ),
        "restart-explicit": (
            "services:\n  app:\n    image: nginx:stable-alpine\n"
            "    restart: always\n    deploy:\n      restart_policy:\n        condition: any\n        delay: 5s\n", None
        ),
        "restart-only": (
            "services:\n  app:\n    image: nginx:stable-alpine\n    restart: always\n", None
        ),
        "hook-normal": (
            "services:\n  app:\n    image: nginx:stable-alpine\n"
            "    pre_start:\n      - image: nginx:stable-alpine\n        command: [\"true\"]\n", None
        ),
        "profile-inactive": (
            "services:\n  app:\n    image: nginx:stable-alpine\n"
            "    profiles: [debug]\n    pre_start:\n      - image: nginx:stable-alpine\n"
            "        per_replica: true\n", None
        ),
    }
    fixtures.update({
        'onfailure-delay': ('services:\n  app:\n    image: nginx:stable-alpine\n    restart: on-failure\n    deploy:\n      restart_policy:\n        delay: 5s\n', 'RESTART_POLICY_OVERRIDDEN'),
        'unless-delay': ('services:\n  app:\n    image: nginx:stable-alpine\n    restart: unless-stopped\n    deploy:\n      restart_policy:\n        delay: 5s\n', 'RESTART_POLICY_OVERRIDDEN'),
        'max-without-restart': ('services:\n  app:\n    image: nginx:stable-alpine\n    deploy:\n      restart_policy:\n        max_attempts: 3\n', 'RESTART_POLICY_INVALID'),
        'any-with-max': ('services:\n  app:\n    image: nginx:stable-alpine\n    deploy:\n      restart_policy:\n        condition: any\n        max_attempts: 3\n', 'RESTART_POLICY_INVALID'),
        'empty-policy': ('services:\n  app:\n    image: nginx:stable-alpine\n    restart: always\n    deploy:\n      restart_policy: {}\n', 'RESTART_POLICY_OVERRIDDEN'),
        'delay-no-restart': ('services:\n  app:\n    image: nginx:stable-alpine\n    deploy:\n      restart_policy:\n        delay: 5s\n', None),
        'valid-onfailure-max': ('services:\n  app:\n    image: nginx:stable-alpine\n    restart: on-failure\n    deploy:\n      restart_policy:\n        condition: on-failure\n        max_attempts: 3\n', None),
    })
    with tempfile.TemporaryDirectory(prefix="beforeup-runtime-guards-") as td:
        for name, (source, finding_id) in fixtures.items():
            file = Path(td) / (name + ".yml")
            file.write_text(source)
            resolved = call("docker", "compose", "-f", str(file), "config", "-q")
            assert resolved.returncode == 0, f"{name}: native Compose config failed"
            args = [str(binary), "check", "-f", str(file), "--min-free-gib", "0", "--format", "json"]
            checked = call(*args)
            assert checked.returncode == 0, f"{name}: non-strict status {checked.returncode}"
            report = json.loads(checked.stdout)
            found = [f for f in report["findings"]
                     if f["id"] in ("PRE_START_REPLICA_UNSUPPORTED", "RESTART_POLICY_OVERRIDDEN", "RESTART_POLICY_INVALID")]
            if finding_id:
                assert len(found) == 1 and found[0]["id"] == finding_id and found[0]["level"] == "warning", (name, found)
                strict = call(*args, "--strict")
                assert strict.returncode == 1, f"{name}: strict warning should fail CI"
                assert json.loads(strict.stdout) == report, f"{name}: strict modified finding"
            else:
                assert not found, (name, found)
            assert "secret-must-never-appear" not in checked.stdout, name
    print(f"compose_runtime_guards: PASS ({len(fixtures)} fixtures; Compose {version.stdout.strip()}; no Engine writes)")

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--binary", type=Path, required=True)
    args = parser.parse_args()
    run(args.binary.resolve())
