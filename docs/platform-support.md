# Linux compatibility and validation

BeforeUp currently supports Linux. A portable static executable is not the same as a verified Docker Engine integration. The table below separates these outcomes.

## Linux/amd64 executable smoke tests — 29 October 2026

The static BeforeUp executable was mounted read-only and run with `beforeup version` inside network-disabled, read-only OCI containers. No Docker socket was exposed to those containers.

| Distribution family | Base images tested |
| --- | --- |
| Ubuntu | 24.04, 26.04 |
| Debian | 12-slim, 13-slim |
| Alpine | 3.22 |
| Fedora | 43 |
| Rocky Linux | 9 |
| Arch Linux | rolling image |
| openSUSE | Leap 15.6 |
| Amazon Linux | 2023 |

All ten image tests passed `beforeup version`. No Docker Engine integration was tested inside these distribution images.

Reproduce the smoke tests on a Linux/amd64 machine with Docker access using `sh tests/distro-smoke.sh ./beforeup`. This optional test downloads missing base images and leaves them cached. The results check Linux ABI/static binary portability on ten base images. They do **not** establish full support for ten distributions. Docker CLI, Compose v2, `ss`, security confinement, rootless daemon permissions and host networking require separate integration tests.

## Native Debian package tests

The `amd64` package was installed, executed and removed successfully in isolated containers with Debian 13, Ubuntu 24.04 and Ubuntu 26.04. No services were created. The packages contain only the executable and documentation, without install-time scripts or automatic privilege changes.

The `arm64` package was built with `GOARCH=arm64` and its Debian metadata inspected, but **no ARM64 machine or emulator was used to install it**. Its runtime packaging compatibility remains unverified.

Reproduce the package install/remove check with an authorized local Docker Engine:

```sh
sh tests/deb-smoke.sh path/to/beforeup_0.3.2.preview2_amd64.deb
```

The script creates and removes temporary, network-disabled containers. It does not mount the host Docker socket.

## Full behavior tests

The current Go unit and integration tests run on the project maintainer's Debian 13 VPS with a conventional Docker Engine and Docker Compose. They cover Compose configuration resolution, socket inspection, bind mounts, current service metadata and JSON output. Running services are inspected in read-only mode.

### Not yet certified

- Docker rootless and user-namespace remapping with real writable bind mounts
- SELinux-enabled hosts such as Fedora/RHEL and enforcing labels
- Docker Desktop for Linux (the daemon lives in a VM)
- Multi-replica Compose services and Docker contexts across hosts
- IPv6-only host publishing and namespace-specific port listeners
- Full Docker Engine/Compose integration on the ten base images above
- ARM64 package installation on an ARM64 host

Do not report these environments as fully supported merely because `beforeup version` runs.
