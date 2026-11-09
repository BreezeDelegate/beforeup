# Windows Docker Desktop preview: verification handoff

The Windows changes live on an experimental branch. They are not a supported release.

## Contract

- Native Windows amd64 binary works with Docker Compose v2 and Docker Desktop using a **local named-pipe endpoint** and **Linux containers**.
- Resolve Compose with Docker's CLI, without running or modifying the stack.
- The check command may report Compose validity and Docker Engine availability, but MUST warn DESKTOP_VM and skip host port listener, local bind-source and VM disk checks.
- The compare command may inspect running containers in the same Compose project. An absent or inaccessible baseline is never healthy; a matching port/mount/environment/image reference does not prove rollout safety.
- Remote Docker contexts, Docker Engine unavailable, and Windows containers must not be reported as healthy native Linux host checks. No sensitive paths, environment values, or raw Docker errors in output.
- The Linux release and diagnostics remain unchanged.

## Automated checks

Run Go unit, race and vet checks on the trusted personal VPS, cross-build a Windows amd64 executable, and run native Windows Go tests and executable smoke. Validate a disposable Docker Compose configuration without starting containers.

For Docker Desktop Linux Engine integration, after the owner has started the app and accepted its agreement, use an isolated disposable project with a cached benign image. Validate normal and changed environment/mount/image/port cases, NO_RUNNING_STACK and BASELINE_UNKNOWN. Check that no ss, sysctl, host-disk or Windows-path probes are invoked, and that no secrets or private mount-source paths appear in output. Run the same cases with inactive profiles and a remote context; no false RUNNING_MATCH is allowed. Capture the binary hash, Compose and Engine versions, command exit codes, and scrubbed findings. Use project-scoped cleanup only; never prune unrelated Docker state.

## Boundaries

- Do not change Docker Desktop settings, restart Windows, switch engine modes, grant broad Docker socket permissions or start/stop other users' containers without authorization.
- No code signing, Windows release, README compatibility claim or package distribution until a Windows native Docker Desktop integration test has passed.
- A Windows binary compiling and passing mocked tests alone is insufficient.
