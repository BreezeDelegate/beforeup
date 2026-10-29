# Changelog

## 0.4.2-preview.1

- Warn about missing local Compose build contexts even when Docker Compose config validation succeeds; remote Git contexts are excluded.
- Check six publicly available Docker Compose examples without launching their full applications.
- Verify a full disposable Compose startup/comparison cycle with a changed environment value and bind mount, using an opt-in integration script.
- Document a real non-root container write denial and the rootless cgroup limitation without claiming full rootless support.


## 0.4.1-preview.1

- Keep unrelated running-service comparisons when a scaled service is present, including orphaned replicas.
- Avoid needless Docker inspections for services with multiple running containers.
- Prevent mount target paths from appearing in reports, and summarize mount changes by service.
- Keep orphan findings in deterministic order for stable machine-readable output.
- Make `make check` non-mutating and keep the roadmap focused on unfinished work.


## 0.4.0-preview.1

- Skip workstation-side socket, filesystem and disk probes when Docker Desktop for Linux uses a VM.
- Warn about rootless Docker low host ports and unverified bind-mount UID/GID mapping, without claiming certain failure.
- Mark potentially overlapping IPv4/IPv6 wildcard bindings as uncertain rather than definite collisions.
- Add optional `--strict` exit behavior for CI; finding levels remain unchanged.
- Round the README banner corners with a small transparent WebP mask.
- Add regression tests and read-only IPv6 listener, dual-stack, CI and isolated rootless daemon checks.
- Preserve confirmed collisions when two services publish the same host port with default IP bindings.


## 0.3.3-preview.1

- Keep comparing unrelated services when a Compose service has multiple running replicas.
- Report the skipped service explicitly instead of returning an opaque project-wide unknown.
- Add an isolated real-Docker integration test for replicas and a second running service.


## 0.3.2-preview.2

- Use GitHub-compatible Debian package filenames so downloaded SHA-256 manifests verify correctly.
- Package Linux amd64 and arm64 binaries as native Debian packages, without installer scripts or privilege changes.
- Add reproducible install/remove checks on Debian 13 and Ubuntu 24.04/26.04.
- Document executable smoke tests on ten Linux distribution base images, distinctly from full host integration.

## 0.3.1-preview.1

- Detect when a local image tag points to a different image ID than the running container.
- Add regression tests for unchanged tags, changed local images and unavailable local image IDs.
- Update troubleshooting guidance for Compose image pulls and no-pull checks.

## 0.3.0-preview.1

- Compare declared environment settings against running container values without printing names or secrets.
- Identify changes to image references, without claiming to detect digest changes.
- Warn when a running service disappears from the proposed resolved Compose model.
- Preserve explicit unknown results when Docker metadata cannot be inspected.
- Document common Docker Compose problems and safe native workarounds.


## 0.2.0-preview.1

- Compare proposed Compose port bindings and data mounts against the running service.
- Identify when a published host port belongs to the current service or another Docker project, including host-address changes.
- Recognize Compose-generated named volume identifiers to avoid spurious migration warnings.
- Suppress potentially sensitive resolved bind-source paths in diagnostic reports.
- Treat projects with no selected active profiles as a valid, empty selection.
- Validate host-side diagnosis on synthetic fixtures and running projects without changing services.

## 0.1.0 (preview)

- First Linux CLI for checking a Docker Compose project against the current host.
- Port listeners and collisions, bind sources, Docker access and disk headroom.
- JSON output and stable finding IDs.
- Explicit warning behavior for remote Docker contexts and uncertain host state.

This is an early version; compatibility and false-positive behavior need independent testing.
