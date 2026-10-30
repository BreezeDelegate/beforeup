# BeforeUp in CI/CD

BeforeUp is a read-only preflight tool for **native Linux Docker Engine hosts**. Run it where the intended Docker daemon, host ports and bind sources are actually accessible. A generic cloud runner, Docker-in-Docker daemon or Docker Desktop VM cannot reliably inspect a different deployment host.

## Preflight before deployment

Install a verified BeforeUp Linux release on an authorized Linux host with Docker CLI, Compose v2 and `ss` (`iproute2`). The account running the check must have read access to the intended Docker Engine.

```sh
beforeup check -f compose.yaml --strict --format json > beforeup-report.json
```

The shell receives BeforeUp's exit status even when JSON is redirected to a file. Treat the report as operational metadata; do not automatically publish it as a CI artifact or upload real `.env` files.

| Exit code | Meaning | Recommended pipeline action |
| --- | --- | --- |
| `0` | No confirmed blockers; without `--strict`, warnings may remain | Continue only when the documented gaps are acceptable |
| `1` | At least one blocker, or at least one warning with `--strict` | Stop and review findings |
| `2` | Invalid Compose configuration or inspection could not complete | Stop and correct the inspection problem |

**`--strict` changes only the exit code**, not finding severity. For example, an existing service may legitimately own a published port. A warning does not prove that a deployment will fail.

To preserve the report and the exit status when collecting diagnostics:

```sh
set +e
beforeup check -f compose.yaml --strict --format json > beforeup-report.json
status=$?
set -e
case "$status" in
  0) echo "Preflight completed";;
  1) echo "Review BeforeUp findings" >&2; exit 1;;
  2) echo "BeforeUp could not inspect the project" >&2; exit 2;;
  *) echo "Unexpected BeforeUp exit status" >&2; exit "$status";;
esac
```

Do not use `|| true` to silently discard failing exit codes.

## Comparing a proposed change

On a host where the same Compose project already has running containers:

```sh
beforeup compare -f compose.yaml --strict --format json > beforeup-compare.json
```

Only selected changes in published ports, data mounts, declared environment settings and image references are compared. BeforeUp does not simulate an upgrade, pull images, check app health, verify remote registry digests or inspect every Compose setting. Multiple replicas generate `REPLICAS_UNSUPPORTED`; other services can still be inspected. A missing baseline is unknown, not healthy.

## Runner access and trust

- Run the check on an authorized **self-managed native Linux runner on the target host**, or invoke it there through an existing secured deployment mechanism.
- Never grant untrusted pull-request code access to a production Docker socket. Access to the daemon is highly privileged.
- A separate Docker daemon is not a substitute for the target host: it would report a different state.
- BeforeUp does not run `up`, `down`, `pull` or `build`. Deployment remains a separate, explicitly authorized step.
- Compose still processes the local environment and configuration. BeforeUp redacts environment values and bind-source paths in its output, but do not upload secrets or full resolved Compose files.

## Reproduce the exit-code contract

On an authorized native Linux Docker host with Compose v2:

```sh
python3 tests/ci_contract.py --binary /path/to/beforeup
```

The optional black-box check uses temporary **configuration files only**. It verifies clean, warning, strict-warning, blocker and invalid-config cases and checks JSON redaction. It does not pull images or start containers. Running it on one host does not certify Docker Desktop, enforced SELinux, rootless bind writes, multi-replica comparisons or other environments.
