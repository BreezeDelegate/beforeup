# Installation

BeforeUp is a small command-line tool with no account, daemon, telemetry or automatic server configuration. The stable v0.4.4 distribution targets Linux; a separate [experimental Windows Docker Desktop preview](windows-preview.md) is available for Linux-container Engines only.

## Downloaded executable

Choose the Linux `amd64` or `arm64` archive in [Releases](https://github.com/BreezeDelegate/beforeup/releases). Verify the SHA-256 checksum against the accompanying `SHA256SUMS` before extracting and installing the executable.

## Debian or Ubuntu package

From the release assets, download the `.deb` matching your CPU architecture, then install that local file:

```sh
sudo apt install ./beforeup_0.4.4_amd64.deb
beforeup version
beforeup check -f compose.yaml
beforeup compare -f compose.yaml
```

For ARM64 use the `arm64` package instead. Neither package installs Docker or grants access to its socket. You still need Docker CLI with Compose v2 and `ss` (`iproute2`) for full checks.

To remove the package:

```sh
sudo apt remove beforeup
```

There are no BeforeUp background services, package scripts or persistent application configuration to remove.

## Build from source

With Go 1.24 or newer:

```sh
go build -trimpath -o beforeup .
./beforeup version
```

To reproduce the Debian packages:

```sh
sh packaging/build-deb.sh amd64 dist
sh packaging/build-deb.sh arm64 dist
```

This requires `dpkg-deb` and Go. Package building does not install anything to the host.

For tested environments and important gaps, see [Linux compatibility](platform-support.md). The Windows preview supports only the tested Docker Desktop Linux Engine scope; macOS and Windows-container mode are unverified. Static executable startup in an OCI image does not guarantee host Docker integration.
