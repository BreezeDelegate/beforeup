<p align="center">
  <img src="assets/readme-banner.webp" alt="BeforeUp — Docker Compose preflight and deployment-change checks" width="800" />
</p>

# BeforeUp

**Docker Compose preflight and deployment-change checks for Linux.** Find host port conflicts, missing bind mounts, changed environment settings and stale local images before changing a running stack.

A Compose file can be valid while its host is not ready. The published port may already be in use, a required bind-mount path may be missing, or Docker may be inaccessible to the account running the deployment.

BeforeUp checks those conditions without starting containers. It also warns when a local Compose build context directory is missing. It relies on Docker Compose to parse the project rather than maintaining another Compose parser.

**Linux only, for now.** This is an early release. Read the [limitations](#what-it-does-not-know) before using it in automation.

## Try it

Requirements: Docker CLI with Compose v2, Linux, and `ss` (from `iproute2`). Syntax validation works without Engine access; host and running-container checks require permission to access Docker. BeforeUp reports missing access instead of requesting privilege escalation.

Build from source using Go 1.24 or newer:

```sh
git clone https://github.com/BreezeDelegate/beforeup.git
cd beforeup
go build -o beforeup .
./beforeup check -f compose.yaml
```

Or use the Linux executable and SHA-256 checksum attached to a [release](https://github.com/BreezeDelegate/beforeup/releases). Check the checksum before running a downloaded executable.

For Debian and Ubuntu, a native `.deb` package is also available starting with the 0.3.2 preview release. See [installation instructions](docs/install.md) and the [compatibility matrix](docs/platform-support.md).

```sh
beforeup check
beforeup check -f compose.yaml --format json
beforeup check -f compose.yaml --profile debug --env-file staging.env
beforeup compare -f compose.yaml
beforeup compare -f compose.yaml --strict --format json
```

Run it in a directory containing your Compose project, or provide `-f`. Repeated `-f`, `--profile`, and `--env-file` options are supported.

## Compare a planned change to what is running

`beforeup compare -f compose.yaml` checks the proposed Compose configuration against **running containers belonging to the same Compose project**. It can tell when:

- a published host port is added or removed;
- an existing container's bind source or named volume would change;
- a currently mounted data path would no longer be mounted;
- a declared environment setting or image reference differs from the running container (without printing secret values);
- a local image with the **same tag** is now different from the image used by the running container (when the image IDs can be inspected);
- an existing running service is absent from the proposed resolved Compose model;
- an intended service has no running counterpart.

Use repeated `-f` options for overrides, just as you would with Docker Compose. This is read-only: it does not simulate a replacement, stop containers or create volumes.

A matching report means only the **inspected port bindings, data mounts, declared environment settings and image reference** appear unchanged. It may also compare a locally available image ID. It does not prove that the remote registry image, inherited environment, command, networks or application behavior are unchanged. Stopped services do not have a reliable running baseline. When a service has multiple running replicas, BeforeUp reports `REPLICAS_UNSUPPORTED` for that service while still checking unrelated services; it does not claim the whole stack is unchanged. A missing baseline is reported explicitly, never treated as healthy.

## What it checks

| Finding | What it means |
| --- | --- |
| `COMPOSE` | The Docker configuration was resolved successfully |
| `DAEMON` | Docker Engine is reachable by the current account |
| `PORT_IN_USE` | A requested TCP/UDP host port has an unassigned listener |
| `PORT_OWNED_BY_SERVICE` | The running version of this service already owns its published port |
| `PORT_BINDING_CHANGED` | The running service uses the requested port on a different host address |
| `PORT_USED_BY_OTHER` | Another Docker service publishes the requested port |
| `PORT_COLLISION` | Two services in the same project have a confirmed host-port collision |
| `PORT_DUAL_STACK_UNCERTAIN` | IPv4/IPv6 bindings may conflict, depending on host dual-stack policy |
| `ROOTLESS_LOW_PORT` | A rootless Docker host port below 1024 may need additional permissions |
| `ROOTLESS_BIND_UNVERIFIED` | Existing bind sources may still fail due to rootless UID/GID mapping |
| `DESKTOP_VM` | Docker Desktop on Linux uses a VM; local host probes are skipped |
| `BIND_MISSING` | A local bind source is absent; an explicitly non-creatable path blocks startup |
| `DISK_LOW` | The filesystem holding Docker's data has less free space than the configured threshold |
| `CONTEXT` | A remote Docker endpoint prevents meaningful local host checks |
| `ENV_CHANGED` | Declared container environment differs from running settings, with no values printed |
| `IMAGE_REFERENCE_CHANGED` | Proposed image reference differs from running container reference |
| `LOCAL_IMAGE_UPDATED` | A locally available image with the same tag now differs from the running container image |
| `SERVICE_ABSENT_FROM_PROPOSAL` | Running service missing from proposed resolved model; may remain as an orphan |
| `PORT_ADDED` / `PORT_REMOVED` | A proposed deployment changes a published host port |
| `MOUNT_REMOVED` / `MOUNT_SOURCE_CHANGED` | A mounted data path or its source changes |
| `NO_ACTIVE_SERVICES` | No services are selected by the current Compose profiles |

Bind source paths are deliberately omitted from reports, because environment interpolation can place sensitive values in them. A missing bind source using legacy short syntax is a warning, not necessarily a failure: Docker may create a new empty directory. A port in use is also a warning. An already-running Compose stack can legitimately own its own published port.

The default disk warning threshold is 2 GiB. Change it with `--min-free-gib N`, or use zero to disable that check. This is a headroom warning, not an estimate of the space your images will need.

For rootless Docker, BeforeUp does not assume that an existing host bind source can be accessed by the container. Privileged host ports may be allowed through system or RootlessKit settings; those cases remain **warnings, never confirmed blockers**. On Docker Desktop for Linux, local socket, bind-source and disk checks are skipped because Docker Engine runs inside a VM. Container-metadata comparison can still run.

## Output and exit codes

The normal output is text suitable for a terminal; `--format json` emits findings with stable IDs, severity, service and advice.

You can reproduce 14 Docker Compose configuration scenarios locally after building BeforeUp:

```sh
go build -o beforeup .
python3 tests/compose_matrix.py --binary ./beforeup
```

This optional test needs Python 3 and authorized local Docker Engine access. It only calls Docker Compose configuration validation and read-only host queries. The running-project ownership and comparison scenarios are covered separately in Go tests and [validation notes](docs/validation.md).

- `0`: checks completed without a confirmed blocker. Warnings may be present.
- `1`: at least one blocker, **or a warning when `--strict` is set**.
- `2`: configuration could not be inspected or the command was invalid.

`--strict` affects **only the exit status**; it does not change the severity or certainty of any finding. Use it only when warnings should fail CI. A passed check is not a guarantee of a successful deployment. See [real troubleshooting examples](docs/common-issues.md) for cases where the native Docker commands already solve the problem.

## What it does not know

BeforeUp cannot predict whether an image will pull, whether a service will become healthy, whether an app will reach its database, or whether another process will claim a port between the check and startup. It does not scan container images for vulnerabilities.

Local filesystem and socket checks are skipped for remote or Docker Desktop VM endpoints. Run on the intended **native Docker host** for those checks. Docker rootless, IPv6-only sockets and SELinux configurations still need independent end-to-end validation.

Docker Compose still processes your project and its environment files. Although BeforeUp does not print resolved environment values, you should treat Compose configurations as sensitive material.

BeforeUp **does not run `up`, create directories, change permissions, clean Docker resources or modify your configuration**. It runs read-only inspection commands. See [security notes](SECURITY.md).

## Contributing

Issues about false positives, genuine missing checks and small reproducible Compose examples are welcome. Please remove credentials and customer details before sharing a configuration.

See [CONTRIBUTING.md](CONTRIBUTING.md), [common Docker Compose problems](docs/common-issues.md), [validation notes](docs/validation.md) and the [French README](README.fr.md).

MIT licensed. [LICENSE](LICENSE). Maintained by [BreezeDelegate](https://github.com/BreezeDelegate); professional work at [vitrine.bar](https://vitrine.bar/).
