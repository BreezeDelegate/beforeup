# Windows Docker Desktop preview

**Experimental release:** BeforeUp v0.4.5-preview.1 for Windows 64-bit. The stable Linux release remains v0.4.4.

This preview was tested with Windows 11, Docker Desktop 4.99.0, Docker Engine 29.9.3 (Linux containers in WSL 2) and Docker Compose 5.7.0. It is not a Windows-container Engine implementation.

## Install from the preview release

1. Install and start Docker Desktop with its **Linux containers** Engine and Docker Compose.
2. Download `beforeup_0.4.5-preview.1_windows_amd64.zip` and `SHA256SUMS` from [the preview release](https://github.com/BreezeDelegate/beforeup/releases/tag/v0.4.5-preview.1) into the same folder.
3. In PowerShell, verify the ZIP digest against the corresponding line of `SHA256SUMS`:

```powershell
$expected = ((Get-Content .\SHA256SUMS | Where-Object { $_ -match 'beforeup_0.4.5-preview.1_windows_amd64.zip$' }) -split '\s+')[0]
$actual = (Get-FileHash .\beforeup_0.4.5-preview.1_windows_amd64.zip -Algorithm SHA256).Hash
if (-not $expected -or $actual -ne $expected) { throw 'Release checksum mismatch' }
Expand-Archive .\beforeup_0.4.5-preview.1_windows_amd64.zip -DestinationPath .\beforeup-preview
.\beforeup-preview\beforeup.exe version
.\beforeup-preview\beforeup.exe compare -f .\compose.yaml
```

Use the correct path to your Compose project. The EXE is portable and does not require an installer or administrator elevation by BeforeUp.

## Verified scope and limits

- Resolves Compose using the Docker CLI.
- Compares selected proposed ports, data mounts, declared environment settings and image references against existing containers of the same Compose project.
- Detects duplicate published host-port declarations across enabled Compose services. Inactive profiles are not treated as active.
- Uses local Docker Desktop named-pipe endpoints and the Linux Engine. With a remote context, only Compose configuration is checked.
- Reports `DESKTOP_VM` and **does not check** Windows host sockets, Windows mount-source existence or free space in Docker's Linux VM. The `check` command is therefore **not equivalent to native Linux host preflight**.
- Reports an inaccessible Engine or missing baseline explicitly. A matching report does not guarantee safe deployment.
- Does not install Docker, alter Docker permissions, pull images, or modify containers. It does not send telemetry.
- Windows containers, Windows ARM64 and other Windows/Docker versions remain unverified.

The EXE is **not code-signed**. Verify its published SHA-256 before running it and follow your organization's executable-security policy. See [QA evidence](https://github.com/BreezeDelegate/beforeup/blob/v0.4.5-preview.1/docs/windows-qa-handoff.md) and [platform matrix](https://github.com/BreezeDelegate/beforeup/blob/v0.4.5-preview.1/docs/platform-support.md).
