# Common Docker Compose problems

These examples are based on recurring questions in developer communities. Start with Docker's own tools. BeforeUp adds a read-only comparison against the target machine or an already-running Compose project, but it does not apply fixes.

## "Port is already allocated"

**What happens:** Compose accepts the file, but a container cannot publish its requested host port.

Try `docker ps --format 'table {{.Names}}\t{{.Ports}}'` and `ss -ltn` on the Docker host. Check whether the port belongs to the **same** running Compose service, another container, or a different program. Do not blindly kill the listener: it may be serving traffic.

```sh
beforeup check -f compose.yaml
beforeup compare -f compose.yaml
```

BeforeUp identifies running Docker owners where possible and distinguishes an already-published port from a proposed change. It reports uncertain listeners as warnings, not proof of deployment failure.

Examples: [Stack Overflow, port already allocated](https://stackoverflow.com/questions/24993704/docker-error-cannot-start-container-port-has-already-been-allocated) and [duplicate published ports](https://stackoverflow.com/questions/56285989/docker-compose-bind-failed-port-is-already-allocated).

## "I changed my .env, but the running container still has the old value"

**Why it happens:** `docker compose restart` restarts the existing container with the same metadata; it does not apply new Compose environment variables.

```sh
beforeup compare -f compose.yaml --env-file staging.env
```

When a proposed **declared** environment setting differs from the value in a running container, BeforeUp reports `ENV_CHANGED` and the count of differences. It deliberately prints neither variable names nor values. Then inspect your Compose environment sources and use `docker compose up -d` when recreation is appropriate. BeforeUp does not itself recreate anything.

Important: `env_file:` inside a service supplies variables **to the container**; `--env-file` supplies variables **for Compose interpolation**. These are not interchangeable. BeforeUp uses Compose's resolved configuration, but cannot automatically diagnose every mistake in that distinction. A change in database initialization variables also does not update an existing database's stored credentials.

Examples: [changes not reflected after restart](https://stackoverflow.com/questions/79036842/docker-not-updating-environment-variables-from-env-file-on-restart); [environment interpolation vs container environment](https://stackoverflow.com/questions/78922940/even-if-i-define-env-file-in-docker-compose-it-is-not-loaded); [Komodo issue about environment and deployment recreation](https://github.com/moghtech/komodo/issues/1205).

## "My data disappeared after changing a Compose volume"

**Possible cause:** a replacement container no longer mounts the same storage source, or an initialization step writes into a different volume. The original data may still exist; do not immediately delete containers or volumes.

```sh
beforeup compare -f compose.yaml
```

BeforeUp warns when a data mount disappears or its source changes, including different named volumes. It does not prove data has been lost or validate its contents. Preserve volumes and backups before changing a live database configuration.

See [a PostgreSQL bind-mount example](https://stackoverflow.com/questions/47214644/permission-denied-when-running-mkdir-inside-of-a-docker-container) and [Docker Compose orphan-volume discussion](https://github.com/docker/compose/issues/13465).

## "The image changed but my service still runs the old one"

**One possibility:** the proposed image reference can differ from the one used to create the current container. A tag can also point to a new digest without changing its text.

```sh
beforeup compare -f compose.yaml
```

BeforeUp detects changed image references and, when both image IDs can be read, whether the **locally available image for the same tag** differs from the image used by the running container (`LOCAL_IMAGE_UPDATED`). This requires no registry request. It cannot tell whether a registry has published a newer image that has not been pulled. Check with `docker compose pull` if updating the image is appropriate, then review `docker compose up -d`. Do not remove the live container merely to force an update; plan downtime and backups where needed.

See [a longstanding question about updating images without changing tags](https://stackoverflow.com/questions/37685581/how-to-get-docker-compose-to-use-the-latest-image-from-the-repository), [a more specific Compose question](https://stackoverflow.com/questions/73429215/force-docker-compose-to-pull-latest-image-for-a-container) and [the Compose up documentation](https://docs.docker.com/reference/cli/docker/compose/up/).

## What BeforeUp intentionally does not answer

- Whether the application is healthy after startup.
- Container write permissions inside a bind mount (UID/GID, user namespaces and ACLs matter).
- Whether an image in the remote registry has changed since the last pull.
- Compatibility of stopped containers and scaled Compose services with the new configuration.
- Whether external load balancers, databases and clients will tolerate a change.

For those cases, native Docker and application-specific diagnostics remain necessary. If BeforeUp has no reliable evidence, it should never present a guess as a successful check.
