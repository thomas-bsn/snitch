"""Approved, immutable runtimes. Installation is called only by an explicit model order.

All packages are wheels named and hashed by the trusted publisher. pip resolves
nothing and runs no source build. Inference uses this venv, never the Worker's
ML environment. Failed revisions stay separate from the previous runtime.
"""
from __future__ import annotations

import contextlib
import hashlib
import json
import os
from pathlib import Path
import platform
import re
import shutil
import subprocess
import sys
import tempfile
import threading
import urllib.request
import urllib.parse
import zipfile
from typing import Any, Mapping

from snitch_worker import voice_models
from snitch_worker.voice_download import without_token

ADAPTERS = {"pyannote": ("pyannote", {"diarization", "fingerprint"}), "nemo": ("sortformer", {"diarization"}),
            "nemo-titanet": ("titanet", {"fingerprint"}), "speechbrain-ecapa": ("ecapa-tdnn", {"fingerprint"})}
_VERSION = re.compile(r"^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$")
_SHA = re.compile(r"^[0-9a-f]{64}$")
_LOCK = threading.RLock()


class RuntimeFailed(RuntimeError):
    def __init__(self, cause: str = "runtime_incompatible") -> None:
        super().__init__(cause)
        self.cause = cause


def validate(manifest: Any, engine: str, kind: str) -> None:
    """Defense in depth on an authenticated server order; never interpret an import path."""
    def require(condition: Any) -> None:
        if not condition:
            raise ValueError()
    try:
        architecture, kinds = ADAPTERS[engine]
        require(isinstance(manifest, dict) and type(manifest["schema"]) is int and manifest["schema"] == 1)
        require(set(manifest) == {"schema", "engine", "version", "adapter_version", "python", "platform", "cuda", "license", "kind", "architecture", "disk_bytes", "vram_bytes", "wheels"})
        require(manifest["engine"] == engine and manifest["kind"] == kind and kind in kinds)
        require(manifest["architecture"] == architecture and type(manifest["adapter_version"]) is int and manifest["adapter_version"] == 1)
        require(isinstance(manifest["version"], str) and _VERSION.fullmatch(manifest["version"]))
        require(manifest["python"] == "3.12" and manifest["platform"] == "linux-x86_64")
        require(isinstance(manifest["cuda"], str) and re.fullmatch(r"cpu|[0-9]{1,2}\.[0-9]{1,2}", manifest["cuda"]))
        require(isinstance(manifest["license"], str) and 0 < len(manifest["license"].strip()) <= 100)
        require(type(manifest["vram_bytes"]) is int and 0 <= manifest["vram_bytes"] <= 256 << 30)
        require(type(manifest["disk_bytes"]) is int)
        require(isinstance(manifest["disk_bytes"], int) and 0 < manifest["disk_bytes"] <= 40 << 30)
        require(isinstance(manifest["wheels"], list) and 0 < len(manifest["wheels"]) <= 256)
        names = set()
        total = 0
        for wheel in manifest["wheels"]:
            require(isinstance(wheel, dict) and set(wheel) == {"name", "url", "sha256", "size"})
            name = wheel["name"]
            require(isinstance(name, str) and len(name) <= 240 and name.endswith(".whl"))
            require("/" not in name and "\\" not in name and ".." not in name and name not in names)
            names.add(name)
            require(_SHA.fullmatch(wheel["sha256"]))
            url = urllib.parse.urlsplit(wheel["url"])
            require(url.scheme == "https" and url.netloc == "files.pythonhosted.org")
            require(not url.query and not url.fragment)
            require(type(wheel["size"]) is int and 0 < wheel["size"] <= 3 << 30)
            total += wheel["size"]
        require(total <= manifest["disk_bytes"])
    except (KeyError, TypeError, ValueError, AttributeError):
        raise RuntimeFailed() from None

def identity(manifest: Mapping[str, Any]) -> str:
    return hashlib.sha256(json.dumps(manifest, sort_keys=True, separators=(",", ":")).encode()).hexdigest()


def root(environ: Mapping[str, str]) -> Path:
    return Path(voice_models.models_root(environ)).parent / "runtimes"


def location(manifest: Mapping[str, Any], environ: Mapping[str, str]) -> Path:
    return root(environ) / manifest["engine"] / identity(manifest)


def installed(manifest: Mapping[str, Any], environ: Mapping[str, str]) -> bool:
    path = location(manifest, environ)
    try:
        return (path / ".ready").read_text() == identity(manifest) and (path / "bin/python").is_file()
    except OSError:
        return False


class _NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, req: Any, fp: Any, code: int, msg: str, headers: Any, newurl: str) -> None:
        raise RuntimeFailed("hf_network")


def _download(wheel: Mapping[str, Any], destination: Path) -> None:
    digest = hashlib.sha256()
    received = 0
    try:
        opener = urllib.request.build_opener(urllib.request.ProxyHandler({}), _NoRedirect())
        with opener.open(wheel["url"], timeout=30) as response, destination.open("xb") as out:
            while chunk := response.read(1 << 20):
                received += len(chunk)
                if received > wheel["size"]:
                    raise RuntimeFailed()
                digest.update(chunk)
                out.write(chunk)
    except RuntimeFailed:
        raise
    except OSError as error:
        raise RuntimeFailed("disk_full" if error.errno == 28 else "hf_network") from None
    if received != wheel["size"] or digest.hexdigest() != wheel["sha256"]:
        raise RuntimeFailed()


def environment(environ: Mapping[str, str]) -> dict[str, str]:
    env = {key: value for key, value in without_token(environ).items()
           if key in {"PATH", "HOME", "TMPDIR", "LANG", "LC_ALL", "SYSTEMROOT", "CUDA_VISIBLE_DEVICES", "LD_LIBRARY_PATH", "SNITCH_VOICE_DEVICE"}}
    # pip configuration, Python hooks and remote model code are outside the contract.
    for key in list(env):
        if key.startswith(("PIP_", "UV_", "PYTHON")) or key in {"VIRTUAL_ENV", "LD_PRELOAD"}:
            env.pop(key, None)
    env.update(HF_HUB_OFFLINE="1", TRANSFORMERS_OFFLINE="1", HF_DATASETS_OFFLINE="1",
               PYTHONNOUSERSITE="1", PIP_CONFIG_FILE=os.devnull, PIP_DISABLE_PIP_VERSION_CHECK="1")
    return env


@contextlib.contextmanager
def _volume_lock(directory: Path) -> Any:
    # Linux/WSL file lock covers two Worker processes sharing a volume, including retries.
    import fcntl
    directory.mkdir(parents=True, exist_ok=True)
    with (directory / ".install.lock").open("a") as handle:
        fcntl.flock(handle, fcntl.LOCK_EX)
        try:
            yield
        finally:
            fcntl.flock(handle, fcntl.LOCK_UN)


def _wheel_budget(directory: Path, budget: int) -> None:
    total = 0
    try:
        for path in directory.iterdir():
            with zipfile.ZipFile(path) as archive:
                for entry in archive.infolist():
                    name = Path(entry.filename)
                    if name.is_absolute() or ".." in name.parts:
                        raise RuntimeFailed()
                    total += entry.file_size
                    if total > budget:
                        raise RuntimeFailed("disk_full")
    except (OSError, zipfile.BadZipFile):
        raise RuntimeFailed() from None


def install(manifest: dict[str, Any], environ: Mapping[str, str], report: Any, detailed: bool = False) -> Path:
    validate(manifest, manifest.get("engine", ""), manifest.get("kind", ""))
    if sys.version_info[:2] != (3, 12) or platform.system() != "Linux" or platform.machine() != "x86_64":
        raise RuntimeFailed()
    target = location(manifest, environ)
    with _LOCK, _volume_lock(target.parent):
        if installed(manifest, environ):
            return target
        target.parent.mkdir(parents=True, exist_ok=True)
        if shutil.disk_usage(target.parent).free < manifest["disk_bytes"]:
            raise RuntimeFailed("disk_full")
        # The final directory has no symlink to a previous release and is never overwritten.
        # venv scripts contain absolute paths, so build at its permanent, digest-qualified location.
        if target.is_symlink():
            raise RuntimeFailed()
        if target.exists():
            shutil.rmtree(target)
        target.mkdir()
        env = environment(environ)
        try:
            report("downloading")
            with tempfile.TemporaryDirectory(dir=target.parent, prefix=".wheels-") as temporary:
                wheels = Path(temporary)
                for wheel in manifest["wheels"]:
                    _download(wheel, wheels / wheel["name"])
                _wheel_budget(wheels, manifest["disk_bytes"] - sum(w["size"] for w in manifest["wheels"]))
                # A model order reports one "verifying" step; an engine order tells installing and verifying apart.
                report("installing" if detailed else "verifying")
                subprocess.run([sys.executable, "-m", "venv", str(target)], env=env, check=True,
                               stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, timeout=120)
                python = str(target / "bin/python")
                # Wheel paths are explicit and already hashed. --no-deps forbids a second resolver.
                subprocess.run([python, "-I", "-m", "pip", "--isolated", "install", "--no-index", "--no-deps",
                                "--only-binary=:all:", *[str(wheels / w["name"]) for w in manifest["wheels"]]],
                               env=env, check=True, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, timeout=1800)
                subprocess.run([python, "-I", "-m", "pip", "--isolated", "check"], env=env, check=True,
                               stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, timeout=60)
                if detailed:
                    report("verifying")
                # Protocol import proves the approved adapter is usable before recording the runtime.
                module = {"pyannote": "pyannote.audio", "nemo": "nemo.collections.asr.models",
                          "nemo-titanet": "nemo.collections.asr.models", "speechbrain-ecapa": "speechbrain.lobes.models.ECAPA_TDNN"}[manifest["engine"]]
                subprocess.run([python, "-I", "-c", "import importlib; importlib.import_module(" + repr(module) + ")"],
                               env=env, check=True, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, timeout=120)
                # Hardware is checked in this runtime, not against the base image's torch.
                probe = "import sys,torch; cuda=sys.argv[1]; need=int(sys.argv[2]); valid = cuda == 'cpu' or (torch.cuda.is_available() and torch.version.cuda == cuda and torch.cuda.get_device_properties(0).total_memory >= need); sys.exit(0 if valid else 1)"
                subprocess.run([python, "-I", "-c", probe, manifest["cuda"], str(manifest["vram_bytes"])], env=env, check=True,
                               stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, timeout=120)
            (target / ".ready").write_text(identity(manifest))
        except RuntimeFailed:
            raise
        except OSError as exc:
            raise RuntimeFailed("disk_full" if exc.errno == 28 else "runtime_incompatible") from None
        except (subprocess.SubprocessError, ValueError):
            raise RuntimeFailed() from None
    return target


def inventory(environ: Mapping[str, str]) -> list[dict[str, str]]:
    """What this volume proves it holds: runtimes whose ready marker matches their own identity. Read-only."""
    found: list[dict[str, str]] = []
    try:
        engines = sorted(path for path in root(environ).iterdir() if path.is_dir() and path.name in ADAPTERS)
    except OSError:
        return found
    for engine in engines:
        try:
            candidates = sorted(engine.iterdir())
        except OSError:
            continue
        for path in candidates:
            try:
                if _SHA.fullmatch(path.name) and (path / ".ready").read_text() == path.name and (path / "bin/python").is_file():
                    found.append({"engine": engine.name, "local_identity": path.name})
            except OSError:
                continue
    return found


def model_binding(path: str) -> dict[str, Any] | None:
    try:
        value = json.loads((Path(path) / ".snitch-runtime.json").read_text())
        validate(value, value.get("engine", ""), value.get("kind", ""))
        return value
    except FileNotFoundError:
        return None
    except (OSError, ValueError, RuntimeFailed, AttributeError):
        raise RuntimeFailed() from None


def bind(path: str, manifest: dict[str, Any]) -> None:
    target = Path(path) / ".snitch-runtime.json"
    descriptor, temporary = tempfile.mkstemp(dir=path, prefix=".runtime-")
    try:
        with os.fdopen(descriptor, "w") as out:
            json.dump(manifest, out, sort_keys=True)
        os.replace(temporary, target)
    finally:
        if os.path.exists(temporary):
            os.unlink(temporary)
