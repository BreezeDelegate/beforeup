# Validation notes

## Why this tool exists

Docker Compose is the source of truth for syntax, profiles, interpolation and resolved configuration. BeforeUp does not attempt to replace it.

Two existing projects cover other parts of the problem:
- [juanfadev/compose-preflight](https://github.com/juanfadev/compose-preflight): checks operational configuration risks.
- [kost-tech/compose-preflight](https://github.com/kost-tech/compose-preflight): checks risky Compose security settings.

BeforeUp focuses instead on a small set of facts about the **actual host**: live listeners, missing bind sources, access to Docker Engine and free space in Docker's data filesystem.

## Reproduced comparison (29 October 2026)

On a Debian Linux VPS running Docker Compose v5.5.1:

1. A test Compose project declared port 8080 and a long bind mount pointing to an absent path with `create_host_path: false`.
2. `docker compose -f compose.yaml config --quiet` exited successfully.
3. BeforeUp reported `PORT_IN_USE` as a warning because a listener was present on the host and `BIND_MISSING` as a blocker because the required path was absent.
4. The tool did not create the path or start containers.
5. A synthetic environment variable included in the Compose project did not appear in the text or JSON report.

The port warning does not prove a restart would fail, since the listener could belong to an existing container in the same project. The bind finding is stronger because Compose is explicitly prohibited from creating the missing path.

Two services publishing the same host port were also accepted by Compose configuration validation; BeforeUp now reports that collision as a blocker.

A deliberately high disk threshold also triggered `DISK_LOW`. This validates the warning branch; it is not evidence that the VPS was short of disk space.

## Checks run for the prototype

- Go unit tests with fake command execution, socket outputs, Docker contexts and filesystem state.
- Optional integration test against the installed Docker Compose CLI.
- A non-mutating inspection on the live Linux host.
- Go formatting and vet.
- Output review for synthetic secret disclosure.

## What remains to validate

Before widespread recommendation, obtain feedback from independent developers working with real Compose projects. Investigate false positives in rootless Docker, unusual network namespaces and IPv6 dual-stack listeners. Extend operating-system support only when these checks can be tested there.

Results from independent users have **not** yet been collected. Usage, downloads and stars should not be presented as proof of quality.
