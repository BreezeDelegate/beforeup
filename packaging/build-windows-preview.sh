#!/bin/sh
set -eu

# Package only a Windows preview, never silently replace a stable release.
if [ "$#" -ne 2 ]; then
  echo "usage: build-windows-preview.sh VERSION OUTPUT_DIR" >&2
  exit 2
fi
version="$1"
out="$2"
root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$root"
if ! grep -Fxq "const version = \"$version\"" main.go; then
  echo "main.go version and requested release do not match" >&2
  exit 2
fi
case "$version" in
  *-preview.*) ;;
  *) echo "only prerelease packaging is supported by this script" >&2; exit 2 ;;
esac
mkdir -p "$out"
out=$(CDPATH= cd -- "$out" && pwd)
archive="beforeup_$version""_windows_amd64.zip"
scratch=$(mktemp -d)
trap 'rm -rf -- "$scratch"' EXIT HUP INT TERM

GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o "$scratch/beforeup.exe" .
cp LICENSE "$scratch/LICENSE"
cp docs/windows-preview.md "$scratch/README-WINDOWS.md"
python3 - "$scratch" "$out/$archive" <<'PY'
from pathlib import Path
import sys
from zipfile import ZipFile, ZipInfo, ZIP_DEFLATED
directory = Path(sys.argv[1])
destination = Path(sys.argv[2])
with ZipFile(destination, "w") as z:
    for name in ("beforeup.exe", "README-WINDOWS.md", "LICENSE"):
        data = (directory / name).read_bytes()
        info = ZipInfo(name, date_time=(2026, 11, 9, 0, 0, 0))
        info.create_system = 3
        info.external_attr = 0o100644 << 16
        info.compress_type = ZIP_DEFLATED
        z.writestr(info, data, compress_type=ZIP_DEFLATED, compresslevel=9)
    assert z.testzip() is None
PY
(cd "$out" && sha256sum "$archive" > SHA256SUMS && sha256sum -c SHA256SUMS)
echo "Windows preview package: $out/$archive"
