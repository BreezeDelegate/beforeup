#!/usr/bin/env python3
"""Verify Windows release packaging without Docker Engine or network calls."""
from __future__ import annotations

import hashlib
import re
import subprocess
import tempfile
from pathlib import Path
from zipfile import ZipFile

root = Path(__file__).resolve().parents[1]
source = (root / "main.go").read_text()
version = re.search(r'^const version = "([^"]+)"$', source, re.MULTILINE)
assert version, "version not declared"
version = version.group(1)
assert re.fullmatch(r"[0-9]+\.[0-9]+\.[0-9]+", version), version
asset = f"beforeup_{version}_windows_amd64_experimental.zip"

with tempfile.TemporaryDirectory(prefix="beforeup-win-package-contract-") as d:
    output = Path(d)
    completed = subprocess.run(
        ["sh", str(root / "packaging/build-windows.sh"), version, str(output)],
        cwd=root, check=True, capture_output=True, text=True, timeout=90
    )
    assert asset in completed.stdout, completed.stdout
    archive = output / asset
    assert archive.is_file(), asset
    checksum = (output / "SHA256SUMS").read_text().strip().split()
    assert checksum[1] == asset, checksum
    assert checksum[0] == hashlib.sha256(archive.read_bytes()).hexdigest()
    with ZipFile(archive) as z:
        assert set(z.namelist()) == {"beforeup.exe", "README-WINDOWS.md", "LICENSE"}
        assert z.testzip() is None
        assert z.read("beforeup.exe")[:2] == b"MZ"
        readme = z.read("README-WINDOWS.md").decode("utf-8")
        assert "experimental" in readme.lower()
        assert version in readme
    invalid = subprocess.run(
        ["sh", str(root / "packaging/build-windows.sh"), "0.4.5-preview.1", str(output / "wrong")],
        cwd=root, capture_output=True, text=True, timeout=15
    )
    assert invalid.returncode == 2
    assert not (output / "wrong").exists()
print(f"windows_package_contract: PASS ({asset}, ZIP, SHA256, PE signature, version gate)")
