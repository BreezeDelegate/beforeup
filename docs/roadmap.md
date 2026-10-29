# Roadmap

BeforeUp stays a small, read-only Docker Compose preflight CLI. Completed work belongs in [CHANGELOG.md](../CHANGELOG.md); this file tracks open decisions only.

## Before a stable release

- Test rootless bind-mount write access with real unprivileged container IDs, including a negative permissions case.
- Verify Docker Desktop for Linux and SELinux-enforcing hosts independently, or document those configurations as unsupported.
- Validate IPv4/IPv6 wildcard and port-ownership behavior on additional real network configurations.
- Get independent user feedback on false positives and whether each BeforeUp diagnostic offers value beyond native Docker commands.
- Verify ARM64 package installation on real ARM64 hardware before claiming it fully supported.

## Release discipline

- Reproduce a real user problem before adding a check; prefer an explicit unknown result to a confident guess.
- Test both success and failure cases, including secret redaction and read-only behavior.
- Build and verify release checksums on files downloaded from GitHub, not only local artifacts.
- Keep tarball and Debian package distribution while they remain useful and tested. Add other formats only when user demand and a real installation test justify them.

## Outside the current scope

Automatic fixes, container cleanup, secret scanning, cloud dashboards, agents, telemetry, subscriptions and a replacement Compose implementation. Application health and guaranteed deployment success are not promises BeforeUp makes.
