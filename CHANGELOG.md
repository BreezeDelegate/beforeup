# Changelog

## 0.2.0-preview.1

- Compare proposed Compose port bindings and data mounts against the running service.
- Identify when a published host port belongs to the current service or another Docker project, including host-address changes.
- Recognize Compose-generated named volume identifiers to avoid spurious migration warnings.
- Suppress potentially sensitive resolved bind-source paths in diagnostic reports.
- Treat projects with no selected active profiles as a valid, empty selection.
- Validate host-side diagnosis on synthetic fixtures and running projects without changing services.

# Changelog

## 0.1.0 (preview)

- First Linux CLI for checking a Docker Compose project against the current host.
- Port listeners and collisions, bind sources, Docker access and disk headroom.
- JSON output and stable finding IDs.
- Explicit warning behavior for remote Docker contexts and uncertain host state.

This is an early version; compatibility and false-positive behavior need independent testing.
