# Roadmap

## v0.1

- Linux-only CLI that delegates Compose resolution to Docker.
- Check occupied published ports, duplicated host port bindings, missing local bind sources, Docker Engine access and disk headroom.
- Preserve Compose profiles, support repeated Compose files and environment files.
- Text and machine-readable JSON; useful exit codes.
- No automatic changes to the host.

## Before a stable release

- Validate against additional real Compose projects from users outside the maintainer's infrastructure.
- Test rootless engines, dual-stack IPv4/IPv6 and network namespaces.
- Identify recurring false positives and either fix them or narrow the advertised scope.
- Provide verified reproducible builds for supported CPU architectures.

## Not on the current roadmap

- Automatic fixes, container cleanup, secret scanning, cloud dashboards, agents, subscriptions or a Compose replacement.
- A success guarantee for deployments. Application health remains outside this tool's scope.
