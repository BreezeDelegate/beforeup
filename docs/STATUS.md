# Project status

## 9 November 2026: public README onboarding update

The English and French README now lead with the read-only `compare` and `check` use cases, display the existing verified preflight GIF inline, and label its limitations. No CLI behavior, release binaries or Docker services changed.

Dynamic GitHub badges now show release-asset download requests, latest stable version and license in both READMEs. The download caption explicitly excludes an interpretation as unique users or installs; no clone or visitor count is publicly claimed.

## 2 November 2026 : 0.4.4 public release verified

- Version 0.4.4 includes three warning IDs: `RESTART_POLICY_OVERRIDDEN`, `RESTART_POLICY_INVALID` and `PRE_START_REPLICA_UNSUPPORTED`, limited to independently exercised Compose clients 5.5.0, 5.5.1, 5.6.0 and 5.6.1.
- On Debian 13 amd64, rootful Engine 29.8.1, 20 primary and 16 edge-case disposable startup checks reproduced runtime failures and policy downgrades; four valid controls started successfully.
- Read-only CLI regression passed 56/56 cases. Original 14-case Compose matrix, Go unit and race tests, vet, output redaction/exit-code contract and fail-closed format gate also passed; statement coverage 84.0 percent.
- Runtime compatibility remains **unverified** for SELinux-enforcing hosts, writable rootless binds, Docker Desktop, ARM64 Engine integration and other Compose clients. Cross-built ARM64 assets alone are not ARM64 runtime proof.
- GitHub release v0.4.4 is public and marked latest (5 assets: 2 Debian packages, 2 Linux archives and SHA256SUMS). All uploaded assets were downloaded and independently compared byte-for-byte to the VPS-built originals; checksum verification passed.
- Release contents are limited to read-only diagnostics. No production Docker service is modified; publication does not expand compatibility claims.

## 30 October 2026

BeforeUp 0.4.3 is the first stable Linux release for the documented native Docker Engine scope. Stable means packaged and regression-tested, not that every Engine environment or deployment is certified. The baseline `check` command verifies Docker Compose configuration against selected local host conditions. The `compare` command reads active containers belonging to the same project and identifies changes in published ports, data mounts, declared container environment values and image references.

### Verified

- CI/CD verification (30 October 2026): published v0.4.3 binary passed the five-case clean/warning/strict/blocker/invalid JSON status contract on the native Debian VPS, including redaction of synthetic environment names, values and bind paths. Added a reproducible black-box script and a self-managed runner guide; no hosted GitHub runner, Go runtime changes or production Docker mutations were introduced. The existing 14-case read-only Compose matrix also passed; Go unit tests were not rerun because the compiler was unavailable in this session.

- Final quality review (0.4.3 preview): existing local build-context files, published-port finding order and failed Docker port-ownership lookups have regression coverage. A failed owner lookup no longer produces a green ports finding; warnings remain non-blocking without `--strict`.

- User-scenario coverage (0.4.2 preview): six representative official Docker Compose files passed static analysis. A missing local build-context check was added after the multi-service NGINX/Node.js/Redis example reproduced the gap. A separate real disposable Compose service validated matching and changed environment/mount diagnostics without host-path disclosure. Rootless application container writes remain unverified because this VPS lacks the required AppArmor/cgroup delegation in its test session.

- Comparison robustness (0.4.1 preview): scaled orphan services are no longer silently omitted, mount-target details are no longer echoed into findings, and deterministic, aggregated mount warnings reduce noise. Five regressions were reproduced before the fixes; the scaled-orphan case was also validated with three isolated Docker containers.

- Host-context safeguards (0.4 preview): Docker Desktop VM detection, conservative rootless port/bind warnings, dual-stack ambiguity and optional strict CI mode have dedicated regressions. IPv6 loopback listener, dual-bind Compose config/dry-run and strict exit codes were exercised on the Debian VPS. A real isolated rootless daemon was also exercised against a bind-source fixture without launching an application container. Docker Desktop VM and SELinux-enforcing hosts were not tested.
- The README banner now has transparent, rounded corners from a deterministic image-mask script; the source design was not regenerated.


- A disposable Docker integration test with two worker replicas and an independent API container confirmed `REPLICAS_UNSUPPORTED` without hiding the remaining service or claiming a full match. The test containers were removed immediately after validation.

- Linux/amd64 static binary version smoke passed on ten isolated distribution images. Debian/Ubuntu `.deb` amd64 package install, version command and removal passed in Debian 13, Ubuntu 24.04 and 26.04. ARM64 packaging was built but was not installed on an ARM64 host. See [platform matrix](platform-support.md).

- Added regression coverage for locally updated images with unchanged tags. The check compares only Docker image IDs already available on the host, without fetching from the registry.

- New isolated regression tests cover changed environment values, image references, unavailable runtime metadata and missing services; all values are redacted from output.
- A local image ID comparison was added for unchanged tags when both IDs are available and the service does not pin another platform. Read-only inspection of a 30-container sample found 30 matching local/running image IDs; this is a compatibility smoke test, not evidence of independent adoption.
- Read-only smoke checks on two existing Compose projects observed an unchanged database and configuration differences in a separate application.

- Unit and regression tests for port ownership, changed host bindings, mount comparisons, Docker contexts, wildcard profiles, output redaction and failure paths.
- Read-only Docker Compose integration tests, plus a portable 14-case Compose matrix and two running-project owner cases.
- Running-stack comparison on existing VPS projects, including a proposed-but-unapplied PostgreSQL port modification.
- Linux amd64/arm64 binaries with SHA-256 checksums; assets downloaded from GitHub and verified.

### Known limitations

- Only Linux is supported. Remote Docker contexts do not get local host checks.
- Compare checks port bindings, data mounts, declared environment values and image references; it does not check remote registry digests, health, inherited environment settings, resource reservations or network topology.
- Multiple replicas of one Compose service receive an explicit `REPLICAS_UNSUPPORTED` warning; unrelated services are still checked. No baseline is reported as unknown, never as healthy.
- Rootless Docker, Docker Desktop VM behavior, IPv6 dual-stack and multiple network namespaces need independent cross-platform integration coverage; the host checks were made conservative but not certified.
- No users outside the maintainer infrastructure have evaluated the program yet. No unsolicited promotional replies were posted; Stack Overflow discussions used as source cases were already answered. No success or GitHub popularity can be inferred from internal tests.

### Discoverability

The public README leads with the actual supported use cases and links to native Docker solutions and troubleshooting documentation. Google Search guidance does not justify a separate AI keyword file or synthetic mentions. Stack Overflow and GitHub contributions should answer the specific question independently of any link and disclose project affiliation if relevant. No promotional comments have been posted.

### Next decision

Gather independent real-world feedback on false positives, rootless/SELinux/Desktop behavior and whether comparisons add value beyond native Docker commands. Do not imply that a stable release proves these environments or external adoption.

### Workspace

Canonical source: `/opt/repos/beforeup` on the personal Netcup VPS. GitHub: `https://github.com/BreezeDelegate/beforeup`. Use the project AGENTS.md and canonical workspace authorship/storage rules before changing it. Build/test locally on the VPS, never in external hosted CI while the VPS is sufficient. Remove temporary Go caches, fixture directories and release staging after each operation. No BeforeUp background service is deployed.
