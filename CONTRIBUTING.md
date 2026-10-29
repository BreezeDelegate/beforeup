# Contributing

BeforeUp is deliberately small. Please start with an issue that shows a concrete failure Docker Compose would not already explain well.

Useful reports include the Compose version, Linux distribution, BeforeUp version, command, expected output and actual output. A minimal synthetic Compose file is ideal. Never attach real `.env` files, tokens, host credentials or private paths.

Changes should stay focused. Include tests for the successful case and a realistic failure case, especially when changing port detection, bind-mount handling, Docker context resolution or output redaction.

Run locally on Linux:

```sh
make check
go build -trimpath -o beforeup .
```

To run the opt-in Compose CLI integration test:

```sh
BEFOREUP_INTEGRATION=1 go test ./...
```

The integration test resolves a temporary Compose project but does not start containers. An additional, explicitly opt-in check uses `sh tests/live-compare-smoke.sh ./beforeup`. It starts one network-isolated Alpine container from an image already cached on the host, then removes the temporary Compose project. It never publishes ports or pulls images; run only on a Docker host where starting a disposable test container is permitted. `make check` verifies formatting without editing source files; `make fmt` applies formatting intentionally.

Use the existing Go standard library when practical. Don't add container-management features, a daemon or a second Compose parser without an explicit need.

Documentation is in English with a French counterpart. Keep both accurate when changing user-facing behavior. External contributions must respect the MIT license and retain third-party notices where relevant.
