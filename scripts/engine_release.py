#!/usr/bin/env python3
"""Builds and signs an approved-engine release (ADR 0055). Publisher tool: the private key never enters Snitch.

    engine_release.py build  --release worker/engine-releases/speechbrain-ecapa-1.0.3-r1 --out manifest.json
    engine_release.py sign   --manifest manifest.json --key-file KEY --key-id KEY_ID --out envelope.json
    engine_release.py verify --envelope envelope.json --public-key BASE64_RAW_PUBLIC_KEY

``build`` reads ``<release>/release.json`` (identity, constraints, contracts) and ``<release>/pylock.toml`` (every
wheel, with URL, size and SHA-256, produced by ``uv pip compile --format pylock.toml``) and writes the manifest as
canonical JSON: the same inputs always give the same bytes. ``sign`` signs exactly those bytes with Ed25519; the
private key is read from the file the operator provides (raw 32 bytes, 64 hex characters, base64, or a PKCS#8 PEM
from ``openssl genpkey -algorithm ed25519``), which must not be readable by group or others. Only the standard
library is needed: Ed25519 is the RFC 8032 reference algorithm, checked against the RFC test vectors.
"""
from __future__ import annotations

import argparse
import base64
import binascii
import hashlib
import json
import os
import re
import stat
import sys
import tomllib
import urllib.parse
from pathlib import Path
from typing import Any

# -- Ed25519 (RFC 8032, section 6 reference implementation) --------------------------------------------------------
_P = 2**255 - 19
_L = 2**252 + 27742317777372353535851937790883648493
_D = -121665 * pow(121666, _P - 2, _P) % _P
_I = pow(2, (_P - 1) // 4, _P)


def _inv(x: int) -> int:
    return pow(x, _P - 2, _P)


def _x_recover(y: int) -> int:
    xx = (y * y - 1) * _inv(_D * y * y + 1)
    x = pow(xx, (_P + 3) // 8, _P)
    if (x * x - xx) % _P:
        x = x * _I % _P
    return _P - x if x & 1 else x


_BY = 4 * _inv(5) % _P
_B = (_x_recover(_BY), _BY, 1, _x_recover(_BY) * _BY % _P)


def _add(p: tuple[int, int, int, int], q: tuple[int, int, int, int]) -> tuple[int, int, int, int]:
    a = (p[1] - p[0]) * (q[1] - q[0]) % _P
    b = (p[1] + p[0]) * (q[1] + q[0]) % _P
    c = 2 * p[3] * q[3] * _D % _P
    d = 2 * p[2] * q[2] % _P
    e, f, g, h = b - a, d - c, d + c, b + a
    return e * f % _P, g * h % _P, f * g % _P, e * h % _P


def _mul(scalar: int, point: tuple[int, int, int, int]) -> tuple[int, int, int, int]:
    result = (0, 1, 1, 0)
    while scalar:
        if scalar & 1:
            result = _add(result, point)
        point = _add(point, point)
        scalar >>= 1
    return result


def _encode(point: tuple[int, int, int, int]) -> bytes:
    zi = _inv(point[2])
    x, y = point[0] * zi % _P, point[1] * zi % _P
    return int.to_bytes(y | ((x & 1) << 255), 32, "little")


def _expand(seed: bytes) -> tuple[int, bytes]:
    h = hashlib.sha512(seed).digest()
    a = int.from_bytes(h[:32], "little")
    a &= (1 << 254) - 8
    a |= 1 << 254
    return a, h[32:]


def public_key(seed: bytes) -> bytes:
    return _encode(_mul(_expand(seed)[0], _B))


def sign_bytes(seed: bytes, message: bytes) -> bytes:
    a, prefix = _expand(seed)
    public = _encode(_mul(a, _B))
    r = int.from_bytes(hashlib.sha512(prefix + message).digest(), "little") % _L
    big_r = _encode(_mul(r, _B))
    k = int.from_bytes(hashlib.sha512(big_r + public + message).digest(), "little") % _L
    return big_r + int.to_bytes((r + k * a) % _L, 32, "little")


def verify_bytes(public: bytes, message: bytes, signature: bytes) -> bool:
    if len(public) != 32 or len(signature) != 64:
        return False
    try:
        y = int.from_bytes(public, "little") & ((1 << 255) - 1)
        x = _x_recover(y)
        if x & 1 != public[31] >> 7:
            x = _P - x
        a_point = (x, y, 1, x * y % _P)
        ry = int.from_bytes(signature[:32], "little") & ((1 << 255) - 1)
        rx = _x_recover(ry)
        if rx & 1 != signature[31] >> 7:
            rx = _P - rx
        r_point = (rx, ry, 1, rx * ry % _P)
    except (ValueError, ZeroDivisionError):
        return False
    s = int.from_bytes(signature[32:], "little")
    if s >= _L:
        return False
    k = int.from_bytes(hashlib.sha512(signature[:32] + public + message).digest(), "little") % _L
    return _encode(_mul(s, _B)) == _encode(_add(r_point, _mul(k, a_point)))


# -- key file --------------------------------------------------------------------------------------------------------
_PKCS8_PREFIX = bytes.fromhex("302e020100300506032b657004220420")


def read_seed(path: str | os.PathLike[str]) -> bytes:
    mode = os.stat(path).st_mode
    if mode & (stat.S_IRWXG | stat.S_IRWXO):
        raise SystemExit("key file must not be accessible to group or others (chmod 600)")
    raw = Path(path).read_bytes()
    if len(raw) == 32:
        return raw
    text = raw.decode("ascii", "replace").strip()
    if text.startswith("-----BEGIN PRIVATE KEY-----"):
        der = base64.b64decode("".join(text.splitlines()[1:-1]))
        if len(der) == 48 and der.startswith(_PKCS8_PREFIX):
            return der[16:]
    elif re.fullmatch(r"[0-9a-fA-F]{64}", text):
        return bytes.fromhex(text)
    else:
        try:
            decoded = base64.b64decode(text, validate=True)
        except binascii.Error:
            decoded = b""
        if len(decoded) == 32:
            return decoded
    raise SystemExit("unsupported key file: expected an Ed25519 seed (32 bytes, hex, base64 or PKCS#8 PEM)")


# -- release ---------------------------------------------------------------------------------------------------------
_MANIFEST_FIELDS = {"schema", "engine", "version", "adapter_version", "python", "platform", "cuda", "license", "kind",
                    "architecture", "disk_bytes", "vram_bytes", "ram_bytes", "capabilities", "input_contract",
                    "output_contract", "cpu_fallback"}


_MAX_GLIBC = (2, 28)


def _wheel_rank(filename: str) -> tuple[int, tuple[int, int]] | None:
    """How well a wheel suits CPython 3.12 on Linux x86_64 with glibc 2.28 (higher wins); None when it cannot be installed there."""
    parts = filename[:-4].split("-")
    if len(parts) not in (5, 6):
        return None
    python_tags, abi_tags, platform_tags = (set(part.split(".")) for part in parts[-3:])
    glibc = (0, 0)
    if platform_tags != {"any"}:
        usable = []
        for tag in platform_tags:
            legacy = {"manylinux1_x86_64": (2, 5), "manylinux2010_x86_64": (2, 12), "manylinux2014_x86_64": (2, 17)}.get(tag)
            match = re.fullmatch(r"manylinux_(\d+)_(\d+)_x86_64", tag)
            version = legacy or ((int(match.group(1)), int(match.group(2))) if match else None)
            if version is not None and version <= _MAX_GLIBC:
                usable.append(version)
        if not usable:
            return None
        glibc = max(usable)
    if "cp312" in python_tags and "cp312" in abi_tags:
        kind = 3
    elif "abi3" in abi_tags and any(re.fullmatch(r"cp3\d+", tag) and int(tag[3:]) <= 12 for tag in python_tags):
        kind = 2
    elif abi_tags == {"none"} and python_tags & {"py3", "py312", "cp312"}:
        kind = 1
    else:
        return None
    return kind, glibc


def canonical(value: Any) -> bytes:
    return json.dumps(value, sort_keys=True, separators=(",", ":"), ensure_ascii=True).encode()


def build_manifest(release: str | os.PathLike[str]) -> dict[str, Any]:
    folder = Path(release)
    spec = json.loads((folder / "release.json").read_text())
    lock = tomllib.loads((folder / "pylock.toml").read_text())
    if set(spec) - _MANIFEST_FIELDS - {"disk_headroom_bytes", "disk_factor"} or "wheels" in spec:
        raise SystemExit("release.json: unknown or forbidden field")
    wheels = []
    for package in lock.get("packages", []):
        files = package.get("wheels") or []
        if len(files) > 1:
            # Several files for one pin (a pure-Python fallback next to a native build, an abi3 wheel next to a
            # CPython 3.12 one): keep the one pip would pick on the target (CPython 3.12, Linux x86_64, glibc 2.28),
            # most specific first. Only a tie that this rule cannot break stays an error.
            ranked = sorted(((_wheel_rank(f["url"].rsplit("/", 1)[-1]), f) for f in files), key=lambda item: item[0] or (-1, (0, 0)), reverse=True)
            ranked = [item for item in ranked if item[0] is not None]
            files = [ranked[0][1]] if ranked and (len(ranked) == 1 or ranked[0][0] != ranked[1][0]) else []
        if len(files) != 1 or package.get("sdist") or package.get("vcs") or package.get("directory"):
            raise SystemExit("%s: exactly one wheel is required (no sdist, no source)" % package.get("name"))
        entry = files[0]
        url = urllib.parse.urlsplit(entry["url"])
        name = url.path.rsplit("/", 1)[-1]
        if url.scheme != "https" or url.netloc != "files.pythonhosted.org" or not name.endswith(".whl"):
            raise SystemExit("%s: wheel URL is not on files.pythonhosted.org" % package["name"])
        digest = entry.get("hashes", {}).get("sha256", "")
        if not re.fullmatch(r"[0-9a-f]{64}", digest) or int(entry.get("size", 0)) <= 0:
            raise SystemExit("%s: missing sha256 or size" % package["name"])
        wheels.append({"name": name, "url": entry["url"], "sha256": digest, "size": int(entry["size"])})
    if not wheels:
        raise SystemExit("pylock.toml holds no package")
    wheels.sort(key=lambda w: w["name"])
    total = sum(w["size"] for w in wheels)
    manifest = {key: value for key, value in spec.items() if key in _MANIFEST_FIELDS}
    # Installed size is bounded by the unpacked wheels: twice the archives plus a fixed headroom, rounded up.
    manifest["disk_bytes"] = -(-(total * spec.get("disk_factor", 2) + spec.get("disk_headroom_bytes", 1 << 30)) // (1 << 20)) * (1 << 20)
    manifest["wheels"] = wheels
    return manifest


def make_envelope(manifest_bytes: bytes, seed: bytes, key_id: str) -> dict[str, str]:
    return {"key_id": key_id, "payload": base64.b64encode(manifest_bytes).decode(),
            "signature": base64.b64encode(sign_bytes(seed, manifest_bytes)).decode()}


def verify_envelope(envelope: dict[str, str], public_b64: str) -> dict[str, Any]:
    payload = base64.b64decode(envelope["payload"], validate=True)
    if not verify_bytes(base64.b64decode(public_b64, validate=True), payload, base64.b64decode(envelope["signature"], validate=True)):
        raise SystemExit("signature invalid")
    return json.loads(payload)


def main(argv: list[str]) -> int:
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    commands = parser.add_subparsers(dest="command", required=True)
    build = commands.add_parser("build")
    build.add_argument("--release", required=True)
    build.add_argument("--out", required=True)
    build.add_argument("--check", action="store_true", help="fail if --out differs from what the release builds")
    sign = commands.add_parser("sign")
    sign.add_argument("--manifest", required=True)
    sign.add_argument("--key-file", required=True)
    sign.add_argument("--key-id", required=True)
    sign.add_argument("--out", required=True)
    verify = commands.add_parser("verify")
    verify.add_argument("--envelope", required=True)
    verify.add_argument("--public-key", required=True, help="raw 32-byte public key, base64 (as in SNITCH_ENGINE_TRUST_KEYS)")
    args = parser.parse_args(argv)
    if args.command == "build":
        data = canonical(build_manifest(args.release)) + b"\n"
        if args.check:
            return 0 if Path(args.out).read_bytes() == data else sys.exit("manifest differs from its inputs")
        Path(args.out).write_bytes(data)
        print("manifest sha256=" + hashlib.sha256(data.rstrip(b"\n")).hexdigest())
    elif args.command == "sign":
        manifest_bytes = Path(args.manifest).read_bytes().rstrip(b"\n")
        json.loads(manifest_bytes)
        seed = read_seed(args.key_file)
        envelope = make_envelope(manifest_bytes, seed, args.key_id)
        Path(args.out).write_text(json.dumps(envelope, indent=2) + "\n")
        print("public key (trust key %s): %s" % (args.key_id, base64.b64encode(public_key(seed)).decode()))
    else:
        manifest = verify_envelope(json.loads(Path(args.envelope).read_text()), args.public_key)
        print("signature valid: %s %s" % (manifest["engine"], manifest["version"]))
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
