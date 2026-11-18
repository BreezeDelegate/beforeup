# Windows Docker Desktop (experimental)

**Experimental Windows package:** BeforeUp v0.4.6 for Windows amd64, included alongside the stable Linux binaries. Windows support is still limited to Docker Desktop with its Linux container Engine.

The supported test configuration is Windows 11 with Docker Desktop Linux containers on WSL 2 and Compose 5.7.0. Other configurations, Windows containers and general Windows host parity have not been validated.

## Install the experimental Windows ZIP

1. Install and start Docker Desktop with its **Linux containers** Engine and Docker Compose.
2. Download `beforeup_0.4.6_windows_amd64_experimental.zip` and `SHA256SUMS` from [v0.4.6](https://github.com/BreezeDelegate/beforeup/releases/tag/v0.4.6) into the same folder.
3. In PowerShell, verify the ZIP digest against the corresponding line of `SHA256SUMS`:

```powershell
$expected = ((Get-Content .\SHA256SUMS | Where-Object { $_ -match 'beforeup_0.4.6_windows_amd64_experimental.zip$' }) -split '\s+')[0]
$actual = (Get-FileHash .\beforeup_0.4.6_windows_amd64_experimental.zip -Algorithm SHA256).Hash
if (-not $expected -or $actual -ne $expected) { throw 'Release checksum mismatch' }
Expand-Archive .\beforeup_0.4.6_windows_amd64_experimental.zip -DestinationPath .\beforeup-windows
.\beforeup-windows\beforeup.exe version
.\beforeup-windows\beforeup.exe compare -f .\compose.yaml
```

Use the correct path to your Compose project. The EXE is portable and does not require an installer or administrator elevation by BeforeUp.

## Verified scope and limits

- Resolves Compose using the Docker CLI.
- Compares selected proposed ports, data mounts, declared environment settings and image references against existing containers of the same Compose project.
- Detects duplicate published host-port declarations across enabled Compose services. Inactive profiles are not treated as active.
- Flags selected high-risk Compose security declarations, including privileged mode, host namespaces and Docker socket mounts, without executing containers.
- Uses local Docker Desktop named-pipe endpoints and the Linux Engine. With a remote context, only Compose configuration is checked.
- Reports `DESKTOP_VM` and **does not check** Windows host sockets, Windows mount-source existence or free space in Docker's Linux VM. The `check` command is therefore **not equivalent to native Linux host preflight**.
- Reports an inaccessible Engine or missing baseline explicitly. A matching report does not guarantee safe deployment.
- Does not install Docker, alter Docker permissions, pull images, or modify containers. It does not send telemetry.
- Windows containers, Windows ARM64 and other Windows/Docker versions remain unverified.

The EXE is **not code-signed**. Verify its published SHA-256 before running it and follow your organization's executable-security policy. See [QA evidence](https://github.com/BreezeDelegate/beforeup/blob/v0.4.6/docs/windows-qa-handoff.md) and [platform matrix](https://github.com/BreezeDelegate/beforeup/blob/v0.4.6/docs/platform-support.md).
