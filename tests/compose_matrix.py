#!/usr/bin/env python3
"""Read-only synthetic Compose comparison. Requires authorized local Docker access."""
import json
import subprocess
import tempfile
from pathlib import Path

from argparse import ArgumentParser
import os
p = ArgumentParser(description="Read-only Docker Compose preflight fixture comparison")
p.add_argument("--binary", required=True, help="Path to a built BeforeUp binary")
args = p.parse_args()
BIN = str(Path(args.binary).resolve())
MISSING = "/tmp/beforeup-validation-source-must-not-exist"
cases = [
    ("healthy-no-ports", "services:\n  web:\n    image: nginx:alpine\n", [], [], None),
    ("required-bind-absent", f"services:\n  web:\n    image: nginx:alpine\n    volumes:\n      - type: bind\n        source: {MISSING}\n        target: /data\n        bind:\n          create_host_path: false\n", ["BIND_MISSING"], [], None),
    ("short-bind-absent", f"services:\n  web:\n    image: nginx:alpine\n    volumes:\n      - {MISSING}:/data\n", ["BIND_MISSING"], [], None),
    ("duplicate-host-port", 'services:\n  one:\n    image: nginx:alpine\n    ports: ["18123:80"]\n  two:\n    image: nginx:alpine\n    ports: ["18123:80"]\n', ["PORT_COLLISION"], [], None),
    ("inactive-profile", f"services:\n  debug:\n    image: nginx:alpine\n    profiles: [debug]\n    volumes:\n      - type: bind\n        source: {MISSING}\n        target: /data\n        bind:\n          create_host_path: false\n", [], ["BIND_MISSING"], None),
    ("active-profile", f"services:\n  debug:\n    image: nginx:alpine\n    profiles: [debug]\n    volumes:\n      - type: bind\n        source: {MISSING}\n        target: /data\n        bind:\n          create_host_path: false\n", ["BIND_MISSING"], [], ["--profile","debug"]),
    ("low-disk-threshold", "services:\n  web:\n    image: nginx:alpine\n", ["DISK_LOW"], [], ["--min-free-gib","999999"]),
    ("normal-disk-threshold", "services:\n  web:\n    image: nginx:alpine\n", ["DISK"], ["DISK_LOW"], ["--min-free-gib","1"]),
    ("existing-bind", "services:\n  web:\n    image: nginx:alpine\n    volumes:\n      - /tmp:/data:ro\n", [], ["BIND_MISSING"], None),
    ("same-port-different-protocol", 'services:\n  tcp:\n    image: nginx:alpine\n    ports: ["18125:80/tcp"]\n  udp:\n    image: nginx:alpine\n    ports: ["18125:53/udp"]\n', [], ["PORT_COLLISION"], None),
    ("same-port-different-ip", 'services:\n  first:\n    image: nginx:alpine\n    ports: ["127.0.0.1:18126:80"]\n  second:\n    image: nginx:alpine\n    ports: ["127.0.0.2:18126:80"]\n', [], ["PORT_COLLISION"], None),
    ("same-port-same-ip", 'services:\n  first:\n    image: nginx:alpine\n    ports: ["127.0.0.1:18127:80"]\n  second:\n    image: nginx:alpine\n    ports: ["127.0.0.1:18127:80"]\n', ["PORT_COLLISION"], [], None),
    ("image-not-validated", 'services:\n  web:\n    image: example.invalid/will-not-pull:no-such-tag\n', [], [], None),
    ("missing-required-variable", 'services:\n  web:\n    image: "nginx:${REQUIRED_IMAGE_TAG:?tag needed}"\n', ["COMPOSE"], [], None),
]

probe = subprocess.run(["docker","info","--format","{{.ServerVersion}}"],stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)
if probe.returncode != 0:
    raise SystemExit("Docker Engine is not accessible to this account. Run the test with an authorized Docker account; BeforeUp never elevates privileges itself.")

with tempfile.TemporaryDirectory(prefix="beforeup-benchmark-") as d:
    root = Path(d)
    rows=[]
    for idx,(name,yaml,expected,excluded,flags) in enumerate(cases,1):
        casedir=root/f"{idx:02d}-{name}"
        casedir.mkdir()
        compose=casedir/"compose.yaml"
        compose.write_text(yaml)
        base=["-f",str(compose)]
        options=flags or []
        # Preserve the current Docker context while clearing fixture-dependent Compose settings.
        env=os.environ.copy()
        for k in ("COMPOSE_PROFILES","COMPOSE_FILE","REQUIRED_IMAGE_TAG","COMPOSE_ENV_FILES"):
            env.pop(k,None)
        native_args=base+(options[:2] if options and options[0]=="--profile" else [])
        native=subprocess.run(["docker","compose",*native_args,"config","-q"],env=env,stdout=subprocess.PIPE,stderr=subprocess.PIPE)
        cmd=[BIN,"check",*base,*options,"--format","json"]
        out=subprocess.run(cmd,env=env,stdout=subprocess.PIPE,stderr=subprocess.PIPE)
        try: report=json.loads(out.stdout)
        except json.JSONDecodeError:raise RuntimeError(f"{name}: output invalid; code={out.returncode}")
        ids={f["id"] for f in report["findings"]}
        if not set(expected).issubset(ids):raise RuntimeError(f"{name}: missing expected {set(expected)-ids}; got {ids}")
        if set(excluded)&ids:raise RuntimeError(f"{name}: unexpected {set(excluded)&ids}")
        if name=="missing-required-variable":
            if native.returncode==0:raise RuntimeError("Docker Compose should reject missing variable")
        elif native.returncode!=0:raise RuntimeError(f"{name}: unexpected Docker Compose reject")
        rows.append({"case":name,"compose":"accepted" if native.returncode==0 else "rejected","beforeup":",".join(sorted(ids)),"exit":out.returncode})
    for r in rows:print(f"{r['case']:<29} Compose={r['compose']:<9} BeforeUp={r['beforeup']} exit={r['exit']}")
    print(f"PASS: {len(rows)} synthetic cases; Docker Compose only resolves configuration, no containers started")
