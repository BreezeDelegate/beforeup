#!/bin/sh
# Build a native Debian package without publishing or installing anything.
set -eu
if [ "$#" -ne 2 ]; then
    echo "Usage: packaging/build-deb.sh amd64|arm64 output-directory" >&2
    exit 2
fi
arch=$1
case "$arch" in
    amd64|arm64) ;;
    *) echo "Unsupported architecture: $arch" >&2; exit 2 ;;
esac
out=$2
command -v go >/dev/null 2>&1 || { echo "Go is required to build BeforeUp" >&2; exit 2; }
command -v dpkg-deb >/dev/null 2>&1 || { echo "dpkg-deb is required" >&2; exit 2; }
root=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
mkdir -p "$out"
out=$(CDPATH= cd -- "$out" && pwd)
umask 022
work=$(mktemp -d "$out/.beforeup-deb.XXXXXXXX")
trap 'rm -rf -- "$work"' EXIT HUP INT TERM
mkdir -p "$work/package/DEBIAN" "$work/package/usr/bin" "$work/package/usr/share/doc/beforeup"
(cd "$root" && GOOS=linux GOARCH="$arch" CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o "$work/package/usr/bin/beforeup" .)
version=$(sed -n 's/^const version = "\([^" ]*\)".*$/\1/p' "$root/main.go")
[ -n "$version" ] || { echo "Cannot read version from main.go" >&2; exit 2; }
case "$version" in
    *-preview.*)
        upstream=${version%%-preview.*}
        preview=${version##*-preview.}
        case "$preview" in *[!0-9]*|'') echo "Invalid preview version" >&2; exit 2;; esac
        debversion="$upstream~preview$preview"
        ;;
    *) debversion=$version ;;
esac
case "$debversion" in
    *[!0-9a-zA-Z.~+:-]*|'') echo "Invalid Debian package version" >&2; exit 2;;
esac
cat > "$work/package/DEBIAN/control" <<EOF
Package: beforeup
Version: $debversion
Section: utils
Priority: optional
Architecture: $arch
Maintainer: BreezeDelegate
Homepage: https://github.com/BreezeDelegate/beforeup
Description: Read-only Docker Compose preflight and change checks
 BeforeUp finds common host-side startup problems and compares
 planned Compose changes with running containers without modifying them.
 Requires a compatible Docker CLI, Compose v2 and access to the intended host.
EOF
install -m 644 "$root/README.md" "$work/package/usr/share/doc/beforeup/README"
install -m 644 "$root/LICENSE" "$work/package/usr/share/doc/beforeup/copyright"
asset_version=$(printf '%s' "$debversion" | tr '~' '.')
dpkg-deb --root-owner-group --build "$work/package" "$out/beforeup_${asset_version}_${arch}.deb" >/dev/null
printf '%s\n' "$out/beforeup_${asset_version}_${arch}.deb"
