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

## Evidence from the 9 November 2026 preview

- Go 1.27.3 on Linux: unit tests, race tests, vet, Windows amd64 cross-build and test cross-compilation passed.
- Native Windows amd64: full Go test suite, vet, local build and executable version smoke passed at source revision 0ef1865.
- Docker CLI 29.9.3 and Compose 5.7.0 were available. A disposable configuration resolved successfully without daemon access.
- With Docker Desktop installed but stopped, BeforeUp check reported COMPOSE pass and DAEMON blocker. Compare additionally reported BASELINE_UNKNOWN. A remote Docker context reported CONTEXT and skipped daemon probes.
- Docker Desktop installation succeeded without launching the GUI or changing system startup. The initial in-session Docker Desktop license agreement and actual Engine integration remain untested. The maintenance account used for remote shell is not the interactive desktop user; the latter must validate its own Docker context.
- Windows tests initially uncovered Linux-specific fixture assumptions; those tests now explicitly run Linux logic on both test hosts, and a Windows temporary path is JSON-encoded rather than interpolated raw. No assertions were removed.

### Suggested supervised final integration

In the interactive Windows desktop session, have the owner or a local operator review and accept Docker Desktop's agreement, then start the Linux Engine. Do not use the maintenance shell account as proof of owner-session access. On the development branch, verify engine reachability, make one disposable Compose fixture with a cached harmless image, and compare it before and after intentionally changing a declared environment value and mount. Confirm only the expected findings and no secret or host-path disclosure. Repeat with a remote context and Windows-container mode as negative controls without touching unrelated projects. Record results and cleanup before deciding on a release.

## Boundaries

- Do not change Docker Desktop settings, restart Windows, switch engine modes, grant broad Docker socket permissions or start/stop other users' containers without authorization.
- No code signing, Windows release, README compatibility claim or package distribution until a Windows native Docker Desktop integration test has passed.
- A Windows binary compiling and passing mocked tests alone is insufficient.
