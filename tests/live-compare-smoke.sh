#!/bin/sh
# Opt-in disposable Engine integration; never use on a production Compose project.
set -eu
if [ "$#" -ne 1 ]; then
  echo "Usage: tests/live-compare-smoke.sh path-to-beforeup" >&2
  exit 2
fi
binary=$(realpath "$1")
[ -x "$binary" ] || { echo "BeforeUp executable missing" >&2; exit 2; }
command -v docker >/dev/null || exit 2
command -v python3 >/dev/null || exit 2
docker image inspect alpine:3.22 >/dev/null 2>&1 || {
  echo "alpine:3.22 must be available locally (no images will be pulled)" >&2
  exit 2
}
mkdir -p "$HOME/.local/state"
base=$(mktemp -d "$HOME/.local/state/beforeup-live-XXXXXXXX")
project=$(basename "$base" | tr "[:upper:]" "[:lower:]")
cleanup() {
  docker compose -p "$project" -f "$base/compose.yaml" down -v --remove-orphans >/dev/null 2>&1 || true
  rm -rf -- "$base"
}
trap cleanup EXIT HUP INT TERM
mkdir -p "$base/old" "$base/new"
cat > "$base/compose.yaml" <<EOF
name: $project
services:
  app:
    image: alpine:3.22
    network_mode: none
    read_only: true
    command: ["sleep", "90"]
    environment:
      MODE: before
    volumes:
      - type: bind
        source: $base/old
        target: /data
        bind:
          create_host_path: false
EOF
docker compose -p "$project" -f "$base/compose.yaml" up -d --no-build --pull never >/dev/null
"$binary" compare -f "$base/compose.yaml" --format json > "$base/baseline.json"
python3 - "$base/baseline.json" <<'PY'
import json,sys
r=json.load(open(sys.argv[1]))
assert any(x['id']=='RUNNING_MATCH' for x in r['findings']),r
PY
sed -i "s|source: $base/old|source: $base/new|;s/MODE: before/MODE: after/" "$base/compose.yaml"
"$binary" compare -f "$base/compose.yaml" --format json > "$base/changed.json"
python3 - "$base/changed.json" "$base" <<'PY'
import json,sys
r=json.load(open(sys.argv[1]));private=sys.argv[2]
ids={x['id'] for x in r['findings']}
assert {'ENV_CHANGED','MOUNT_SOURCE_CHANGED'} <= ids,ids
assert 'RUNNING_MATCH' not in ids,ids
assert private not in str(r),'private host path present in report'
PY
echo "PASS: running Compose baseline, changed environment/mount, and path redaction"
