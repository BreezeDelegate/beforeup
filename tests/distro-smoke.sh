#!/bin/sh
# Optional binary-only Linux distro smoke test; no Docker socket exposed.
set -eu
if [ "$#" -ne 1 ]; then
    echo "Usage: tests/distro-smoke.sh path-to-linux-amd64-binary" >&2
    exit 2
fi
[ "$(uname -m)" = "x86_64" ] || { echo "This matrix tests Linux/amd64 only." >&2; exit 2; }
bin=$(realpath "$1")
[ -f "$bin" ] || { echo "Binary not found" >&2; exit 2; }
expected=$("$bin" version)
for image in ubuntu:24.04 ubuntu:26.04 debian:12-slim debian:13-slim alpine:3.22 fedora:43 rockylinux:9 archlinux:latest opensuse/leap:15.6 amazonlinux:2023; do
    actual=$(docker run --rm --pull=missing --network=none --read-only --pids-limit=64 --memory=256m \
        --mount "type=bind,source=$bin,target=/beforeup,readonly" \
        --entrypoint /beforeup "$image" version)
    if [ "$actual" != "$expected" ]; then
        echo "FAIL $image: unexpected version output" >&2
        exit 1
    fi
    echo "PASS $image"
done
echo "10/10 executable startup smoke tests passed. Docker Engine integration was not tested."
