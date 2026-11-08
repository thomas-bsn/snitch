#!/usr/bin/env python3
"""Prints, with their SHA-256 hashes, the packages of worker/voice-runtime.lock that the Worker unit tests need
(the generated API client and what it imports). Same versions and hashes as the image: nothing is resolved afresh.
Usage: worker_test_deps.py > requirements.txt && pip install --require-hashes --no-deps -r requirements.txt
"""
import re
import sys
from pathlib import Path

NEEDED = {"anyio", "attrs", "certifi", "h11", "httpcore", "httpx", "idna", "python-dateutil", "pyyaml", "six", "sniffio", "typing-extensions"}
lock = (Path(__file__).resolve().parents[1] / "worker/voice-runtime.lock").read_text()
found = set()
for name, hashes in re.findall(r"^([A-Za-z0-9_.-]+==\S+) \\\n((?:\s+--hash=\S+(?: \\)?\n?)+)", lock, re.M):
    if name.split("==")[0].lower() in NEEDED:
        found.add(name.split("==")[0].lower())
        print(name + " \\\n" + hashes.rstrip("\n").rstrip("\\").rstrip())
missing = {"anyio", "attrs", "certifi", "h11", "httpcore", "httpx", "idna", "sniffio"} - found
if missing:
    sys.exit("not in voice-runtime.lock: %s" % sorted(missing))
