# BeforeUp

Check a Docker Compose project against the machine it will run on, before you start it.

A Compose file can be valid while its host is not ready. The published port may already be in use, a required bind-mount path may be missing, or Docker may be inaccessible to the account running the deployment.

BeforeUp checks those conditions without starting containers. It relies on Docker Compose to parse the project rather than maintaining another Compose parser.

**Linux only, for now.** This is an early release. Read the [limitations](#what-it-does-not-know) before using it in automation.

## Try it

Requirements: Docker CLI with Compose v2, Linux, and `ss` (provided by `iproute2`). Docker Engine access is needed for all checks; when unavailable, BeforeUp reports that problem as well.

Build from source using Go 1.24 or newer:

```sh
git clone https://github.com/BreezeDelegate/beforeup.git
cd beforeup
go build -o beforeup .
./beforeup check -f compose.yaml
```

Or use the Linux executable and SHA-256 checksum attached to a [release](https://github.com/BreezeDelegate/beforeup/releases). Check the checksum before running a downloaded executable.

```sh
beforeup check
beforeup check -f compose.yaml --format json
beforeup check -f compose.yaml --profile debug --env-file staging.env
beforeup compare -f compose.yaml
```

Run it in a directory containing your Compose project, or provide `-f`. Repeated `-f`, `--profile`, and `--env-file` options are supported.

## Compare a planned change to what is running

`beforeup compare -f compose.yaml` checks the proposed Compose configuration against **running containers belonging to the same Compose project**. It can tell when:

- a published host port is added or removed;
- an existing container's bind source or named volume would change;
- a currently mounted data path would no longer be mounted;
- an intended service has no running counterpart.

Use repeated `-f` options for overrides, just as you would with Docker Compose. This is read-only: it does not simulate a replacement, stop containers or create volumes.

A matching report means only the **inspected port bindings and data mounts** appear unchanged. It does not prove that the image, environment variables, command, networks or application behavior are unchanged. Stopped services and Compose replicas do not currently have full comparison support. A missing baseline is reported explicitly, not treated as a healthy deployment.

## What it checks

| Finding | What it means |
| --- | --- |
| `COMPOSE` | Docker Compose could resolve the project's configuration |
| `DAEMON` | Docker Engine is reachable by the current account |
| `PORT_IN_USE` | A requested TCP/UDP host port has an unassigned listener |
| `PORT_OWNED_BY_SERVICE` | The running version of this service already owns its published port |
| `PORT_BINDING_CHANGED` | The running service uses the requested port on a different host address |
| `PORT_USED_BY_OTHER` | Another Docker service publishes the requested port |
| `PORT_COLLISION` | Two services in the same project claim an overlapping host port |
| `BIND_MISSING` | A local bind source is absent; an explicitly non-creatable path blocks startup |
| `DISK_LOW` | The filesystem holding Docker's data has less free space than the configured threshold |
| `CONTEXT` | A remote Docker endpoint prevents meaningful local host checks |
| `PORT_ADDED` / `PORT_REMOVED` | A proposed deployment changes a published host port |
| `MOUNT_REMOVED` / `MOUNT_SOURCE_CHANGED` | A mounted data path or its source changes |
| `NO_ACTIVE_SERVICES` | No services are selected by the current Compose profiles |

Bind source paths are deliberately omitted from reports, because environment interpolation can place sensitive values in them. A missing bind source using legacy short syntax is a warning, not necessarily a failure: Docker may create a new empty directory. A port in use is also a warning. An already-running Compose stack can legitimately own its own published port.

The default disk warning threshold is 2 GiB. Change it with `--min-free-gib N`, or use zero to disable that check. This is a headroom warning, not an estimate of the space your images will need.

## Output and exit codes

The normal output is text suitable for a terminal; `--format json` emits findings with stable IDs, severity, service and advice.

You can reproduce 14 Docker Compose configuration scenarios locally after building BeforeUp:

```sh
go build -o beforeup .
python3 tests/compose_matrix.py --binary ./beforeup
```

This optional test needs Python 3 and authorized local Docker Engine access. It only calls Docker Compose configuration validation and read-only host queries. The running-project ownership and comparison scenarios are covered separately in Go tests and [validation notes](docs/validation.md).

- `0`: checks completed without a confirmed blocker. Warnings may be present.
- `1`: at least one confirmed blocker.
- `2`: configuration could not be inspected or the command was invalid.

A warning is never silently promoted to a blocker. A passed check is not a guarantee of a successful deployment.

## What it does not know

BeforeUp cannot predict whether an image will pull, whether a service will become healthy, whether an app will reach its database, or whether another process will claim a port between the check and startup. It does not scan container images for vulnerabilities.

Local filesystem and socket checks are skipped if the active Docker endpoint appears remote. Run it on the intended host instead.

Docker Compose still processes your project and its environment files. Although BeforeUp does not print resolved environment values, you should treat Compose configurations as sensitive material.

BeforeUp **does not run `up`, create directories, change permissions, clean Docker resources or modify your configuration**. It runs read-only inspection commands. See [security notes](SECURITY.md).

## Contributing

Issues about false positives, genuine missing checks and small reproducible Compose examples are welcome. Please remove credentials and customer details before sharing a configuration.

See [CONTRIBUTING.md](CONTRIBUTING.md), [the validation notes](docs/validation.md) and the [French README](README.fr.md).

MIT licensed. [LICENSE](LICENSE). Maintained by [BreezeDelegate](https://github.com/BreezeDelegate); professional work at [vitrine.bar](https://vitrine.bar/).
