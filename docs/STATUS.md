# Project status

## 29 October 2026

BeforeUp is a public Linux-only preview. The baseline `check` command verifies Docker Compose configuration against selected local host conditions. The `compare` command reads active containers belonging to the same project and identifies changes in published ports, data mounts, declared container environment values and image references.

### Verified

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
- Multiple replicas of one Compose service are not yet comparable. No baseline is reported as unknown, never as healthy.
- Rootless Docker, IPv6 dual-stack and multiple network namespaces need additional independent coverage.
- No users outside the maintainer infrastructure have evaluated the program yet. No unsolicited promotional replies were posted; Stack Overflow discussions used as source cases were already answered. No success or GitHub popularity can be inferred from internal tests.

### Discoverability

The public README leads with the actual supported use cases and links to native Docker solutions and troubleshooting documentation. Google Search guidance does not justify a separate AI keyword file or synthetic mentions. Stack Overflow and GitHub contributions should answer the specific question independently of any link and disclose project affiliation if relevant. No promotional comments have been posted.

### Next decision

Arrange independent tests using varied real Compose deployments. Collect actual false-positive cases and feedback about whether comparisons answer a question better than Docker native commands and existing preflight projects. Do not claim stable readiness or superior adoption before that evidence exists.

### Workspace

Canonical source: `/opt/repos/beforeup` on the personal Netcup VPS. GitHub: `https://github.com/BreezeDelegate/beforeup`. Use the project AGENTS.md and canonical workspace authorship/storage rules before changing it. Build/test locally on the VPS, never in external hosted CI while the VPS is sufficient. Remove temporary Go caches, fixture directories and release staging after each operation. No BeforeUp background service is deployed.
