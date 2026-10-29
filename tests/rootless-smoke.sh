#!/bin/sh
# Optional isolated, read-only BeforeUp smoke test against a real rootless daemon.
# Does not alter the system Docker service, contexts or Docker permissions.
set -eu
if [ "$#" -ne 1 ]; then
  echo "Usage: tests/rootless-smoke.sh path-to-beforeup-binary" >&2
  exit 2
fi
bin=$(realpath "$1")
[ -x "$bin" ] || { echo "BeforeUp binary is not executable" >&2; exit 2; }
for cmd in docker dockerd-rootless.sh rootlesskit python3; do
  command -v "$cmd" >/dev/null || { echo "Missing prerequisite: $cmd" >&2; exit 2; }
done
if [ "$(id -u)" -eq 0 ]; then
  echo "Run as a non-root user with subordinate UID/GID ranges" >&2
  exit 2
fi
base="$HOME/.local/state/beforeup-rootless-smoke-$$"
[ ! -e "$base" ] || exit 2
umask 077
mkdir -p "$base/run" "$base/data" "$base/exec" "$base/source"
export XDG_RUNTIME_DIR="$base/run"
export DOCKER_HOST="unix://$base/run/docker.sock"
export DOCKERD_ROOTLESS_ROOTLESSKIT_NET=none
export DOCKERD_ROOTLESS_ROOTLESSKIT_MTU=0
export DOCKERD_ROOTLESS_ROOTLESSKIT_DETACH_NETNS=false
unset DOCKER_CONTEXT || true
cat > "$base/compose.yaml" <<EOF
name: beforeup-rootless-smoke
services:
  web:
    image: alpine:3.22
    ports:
      - "80:80"
    volumes:
      - type: bind
        source: $base/source
        target: /data
        bind:
          create_host_path: false
EOF
dockerd-rootless.sh --data-root="$base/data" --exec-root="$base/exec" --pidfile="$base/docker.pid" --host="$DOCKER_HOST" --iptables=false --ip6tables=false --bridge=none >"$base/daemon.log" 2>&1 &
pid=$!
cleanup() {
  kill "$pid" 2>/dev/null || true
  wait "$pid" 2>/dev/null || true
  sleep 1
  if docker info >/dev/null 2>&1; then
    echo "Rootless daemon remained reachable. QA state retained at $base" >&2
  else
    rm -rf -- "$base"
  fi
}
trap cleanup EXIT HUP INT TERM
ready=false
attempt=1
while [ "$attempt" -le 30 ]; do
  if docker info >/dev/null 2>&1; then ready=true;break; fi
  if ! kill -0 "$pid" 2>/dev/null; then break;fi
  sleep 1
  attempt=$((attempt + 1))
done
if [ "$ready" != true ]; then
  echo "Rootless daemon could not start:" >&2
  tail -n 15 "$base/daemon.log" >&2
  exit 3
fi
"$bin" check -f "$base/compose.yaml" --format json > "$base/report.json" || {
  code=$?
  if [ "$code" -ne 1 ]; then exit "$code"; fi
}
python3 - "$base/report.json" <<'PY'
import json,sys
r=json.load(open(sys.argv[1]))
found={f["id"] for f in r["findings"]}
assert "DAEMON" in found and "ROOTLESS_BIND_UNVERIFIED" in found, found
assert not r["summary"]["blockers"],r
print("PASS: real isolated rootless daemon and conservative bind-mount warning")
PY
