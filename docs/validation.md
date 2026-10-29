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

Two existing Compose projects on the same host were also inspected without launching or changing their services. Both produced only the expected busy-port warnings and no confirmed blockers. This is a local compatibility check, not independent field validation.

## Checks run for the prototype

- Go unit tests with fake command execution, socket outputs, Docker contexts and filesystem state.
- Optional integration test against the installed Docker Compose CLI.
- A non-mutating inspection on the live Linux host.
- Go formatting and vet.
- Output review for synthetic secret disclosure.

## What remains to validate

Before widespread recommendation, obtain feedback from independent developers working with real Compose projects. Investigate false positives in rootless Docker, unusual network namespaces and IPv6 dual-stack listeners. Extend operating-system support only when these checks can be tested there.

Results from independent users have **not** yet been collected. Usage, downloads and stars should not be presented as proof of quality.

## Running-stack comparison and synthetic matrix (29 October 2026)

The preview comparison was tested against two running Compose projects on the maintainer's Linux VPS. In an unmodified PostgreSQL service, BeforeUp recognized that its published port belonged to the same running service and found no port or mount changes. An alternate Compose override proposed a different published port. Docker Compose accepted the override; BeforeUp reported both the port being removed and the proposed replacement. No containers were restarted.

Another running application was inspected. The initial comparison mistook a Compose-generated named volume identifier for a volume change. A regression test was written and the comparison fixed to resolve the logical Compose volume name. After the correction, the remaining differences related to actual host bind-source paths from different release directories.

A **16-case synthetic validation matrix** was executed with `docker compose config` and the BeforeUp Linux binary. The cases covered valid and invalid configuration, required and auto-created bind sources, ports owned by the same or another Compose project, duplicate published ports, active and inactive Compose profiles, low disk thresholds, existing bind sources, IPv4 binding and protocol distinctions, and image references that are not checked at this stage. Fifteen configurations were accepted by Compose; one missing required variable was rejected by Compose and BeforeUp. BeforeUp identified extra host-side findings in the relevant accepted scenarios.

The portable subset of 14 cases is reproducible via `python3 tests/compose_matrix.py --binary ./beforeup` after compiling the tool. The two host-dependent port ownership scenarios were exercised separately against the actual running VPS and through isolated Go regression tests. This fixture script never starts or removes containers.

The matrix also caught a bug: Compose can validly select zero services when all are disabled by profiles. BeforeUp now reports `NO_ACTIVE_SERVICES` without claiming the Compose file is invalid. It does **not** verify whether a referenced image can be downloaded; the image-reference case remains a documented gap.

All these cases are internal tests. No independent developer adoption has yet been measured, and competing tools were compared at the documentation/feature level rather than through an independently validated execution benchmark. These results do not establish superiority over other products.

## Community-derived scenarios and runtime metadata (29 October 2026)

Stack Overflow questions about environment values not updating after `docker compose restart`, host ports already allocated and bind-mount problems were reviewed. An existing Komodo GitHub issue discusses Compose interpolation and container recreation. References, native workarounds and the checks BeforeUp can or cannot provide are recorded in [common issues](common-issues.md).

For the environment-change case, new Go tests compare only environment settings explicitly present in the proposed resolved Compose model against running container values. Extra variables inherited from an image are not treated as configuration changes. Sensitive names and values are withheld from diagnostics. Other tests cover changed image references, failed metadata inspection and services that no longer appear in the proposal.

The change was exercised in read-only mode against two running Compose projects on the maintainer VPS. This is not independent field validation. No replies or promotional posts were made in community discussions.

## Same-tag local image updates (29 October 2026)

Questions about `docker compose pull`, unchanged image tags and whether `docker compose up` recreates a container led to a targeted read-only check. BeforeUp now compares the image ID used by the running container against the ID of the image currently available **locally** for that tag. It reports `LOCAL_IMAGE_UPDATED` only when both IDs are valid and different. If the image is unavailable locally, the check makes no claim. Services pinned to another platform are excluded from this comparison until platform-sensitive checks are supported. The check makes no remote registry request.

The regression was reproduced before the change and passes after it. It includes a case with matching local IDs and one where the local image cannot be inspected. No image was pulled and no container was recreated.
