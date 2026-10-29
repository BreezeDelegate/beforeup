# Roadmap

## v0.1

- Linux-only CLI that delegates Compose resolution to Docker.
- Check occupied published ports, duplicated host port bindings, missing local bind sources, Docker Engine access and disk headroom.
- Preserve Compose profiles, support repeated Compose files and environment files.
- Text and machine-readable JSON; useful exit codes.
- No automatic changes to the host.

## v0.2 preview

- Identify owners of published Docker ports instead of treating every live binding as an unexplained conflict.
- Compare running Compose services with proposed host port and data-mount configurations.
- Distinguish named Docker volumes from changed data sources.
- Confirm behavior with a 16-case read-only synthetic validation matrix.

## v0.3 preview

- Detect differences in explicitly declared environment values without displaying credentials.
- Warn about changed image references and running services missing from the proposal.
- Link practical troubleshooting to documented user questions, including where native Docker commands are sufficient.

## 0.3.1 preview

- Identify locally updated image IDs even when the image tag stays unchanged.
- Link troubleshooting information to the related Docker questions and native fixes.

## Before a stable release

- Validate against additional real Compose projects from users outside the maintainer's infrastructure.
- Test rootless engines, dual-stack IPv4/IPv6 and network namespaces.
- Identify recurring false positives and either fix them or narrow the advertised scope.
- Provide verified reproducible builds for supported CPU architectures.

## Not on the current roadmap

- Automatic fixes, container cleanup, secret scanning, cloud dashboards, agents, subscriptions or a Compose replacement.
- A success guarantee for deployments. Application health remains outside this tool's scope.
