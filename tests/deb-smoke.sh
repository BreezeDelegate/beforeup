#!/bin/sh
# Verify install, executable startup and clean removal in disposable Linux containers.
set -eu
if [ "$#" -ne 1 ]; then
    echo "Usage: tests/deb-smoke.sh path-to-linux-amd64.deb" >&2
    exit 2
fi
pkg=$1
pkg=$(realpath "$pkg")
[ -f "$pkg" ] || { echo "Package not found" >&2; exit 2; }
command -v dpkg-deb >/dev/null 2>&1 || { echo "dpkg-deb missing" >&2; exit 2; }
expected=$(dpkg-deb -f "$pkg" Version | sed "s/~preview/-preview./")
command -v docker >/dev/null 2>&1 || { echo "Docker CLI missing" >&2; exit 2; }
for image in debian:13-slim ubuntu:24.04 ubuntu:26.04; do
    echo "Testing install/remove on $image"
    docker run --rm --network=none --pids-limit=64 --memory=256m -e "EXPECTED_VERSION=$expected" \
        --mount "type=bind,source=$pkg,target=/tmp/beforeup.deb,readonly" \
        --entrypoint /bin/sh "$image" -ec '
            dpkg -i /tmp/beforeup.deb >/dev/null
            test "$(beforeup version)" = "$EXPECTED_VERSION"
            dpkg -r beforeup >/dev/null
            test ! -e /usr/bin/beforeup
        '
done
echo "All package installation and removal checks passed."
