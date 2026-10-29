# Project status

## 29 October 2026

BeforeUp is a public Linux-only preview. The baseline `check` command verifies Docker Compose configuration against selected local host conditions. The `compare` command reads active containers belonging to the same project and identifies changes in published ports, data mounts, declared container environment values and image references.

### Verified

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

Arrange independent tests using varied real Compose deployments. Collect actual false-positive cases and feedback about whether comparisons answer a question better than Docker native commands and existing preflight projects. Do not claim stable readiness or superior adoption before that evidence exists.

### Workspace

Canonical source: `/opt/repos/beforeup` on the personal Netcup VPS. GitHub: `https://github.com/BreezeDelegate/beforeup`. Use the project AGENTS.md and canonical workspace authorship/storage rules before changing it. Build/test locally on the VPS, never in external hosted CI while the VPS is sufficient. Remove temporary Go caches, fixture directories and release staging after each operation. No BeforeUp background service is deployed.
