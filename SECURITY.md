# Security notes

BeforeUp reads Docker Compose configuration and inspects the local host. It does not start services, clean Docker resources or rewrite project files.

Both commands report a small set of static warnings for selected high-impact Compose declarations: privileged mode, sensitive added Linux capabilities, disabled seccomp/AppArmor/SELinux confinement, bind sources named `docker.sock`, and host network/PID/IPC namespaces. These warnings do not prove an exploit, and a clean result does not certify a service or image as secure. A socket warning is based on the declared bind source name, not a live-socket probe. No images are pulled or scanned, and no containers are started to test application health.

Docker Compose can expand environment variables and read local files, including secrets. The compare command also reads current container port bindings, mounts, image reference and environment values through Docker inspect. These sensitive values are processed in memory only: reports contain counts of changed declared settings, not environment names or values. BeforeUp does not include interpolated Compose values, resolved host bind-source paths or raw Compose error output in its reports. Avoid running untrusted Compose projects.

Docker access is security-sensitive. Membership in the `docker` group often amounts to administrator-level privileges. BeforeUp never requests `sudo` or broadens Docker permissions on your behalf.

Local socket and filesystem checks are skipped for remote, VM-backed Docker Desktop or unconfirmed Docker endpoints. If the checks are incomplete, the report must say so. Rootless UID/GID mapping, SELinux policies and IPv6 cross-family bindings cannot be inferred from file existence or a simple socket match; they are treated conservatively. No check guarantees that an application will start successfully.

## Report a vulnerability

Please use GitHub's private vulnerability reporting feature when available. Do not post exploit details, credentials, tokens, real environment files or customer configurations in public issues.

Include affected version, expected behavior, reproducible sanitized steps and observed consequences. For ordinary bugs, open a public issue with a minimal, synthetic Compose example.

## Design rules

- No telemetry, network service or account.
- No cleaning, automatic fixes or configuration writes.
- No shell evaluation of user-provided values.
- Redact resolved Compose content, Docker inspection environment values and command stderr from all reports.
- Keep command output bounded and operations time-limited.
- Treat non-verifiable conditions as warnings or unknowns, never as successful checks.
