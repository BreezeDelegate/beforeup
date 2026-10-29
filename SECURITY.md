# Security notes

BeforeUp reads Docker Compose configuration and inspects the local host. It does not start services, clean Docker resources or rewrite project files.

Docker Compose can expand environment variables and read local files, including secrets. The compare command also reads current container port bindings, mounts, image reference and environment values through Docker inspect. These sensitive values are processed in memory only: reports contain counts of changed declared settings, not environment names or values. BeforeUp does not include interpolated Compose values, resolved host bind-source paths or raw Compose error output in its reports. Avoid running untrusted Compose projects.

Docker access is security-sensitive. Membership in the `docker` group often amounts to administrator-level privileges. BeforeUp never requests `sudo` or broadens Docker permissions on your behalf.

Local socket and filesystem checks are skipped for remote or unconfirmed Docker endpoints. If the checks are incomplete, the report must say so. No check guarantees that an application will start successfully.

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
