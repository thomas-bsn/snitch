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
_REQUIRED = {"schema", "engine", "version", "adapter_version", "python", "platform", "cuda", "license", "kind", "architecture", "disk_bytes", "vram_bytes", "wheels"}
_OPTIONAL = {"ram_bytes", "capabilities", "input_contract", "output_contract", "cpu_fallback"}
# Source separation is not an approved capability: the V2 separation stays inactive.
_CAPABILITIES = {"diarization", "fingerprint"}
_CONTRACT = re.compile(r"^[a-z0-9][a-z0-9._-]{0,63}$")
CPU_FALLBACK_ENV = "SNITCH_VOICE_CPU_FALLBACK"


class RuntimeFailed(RuntimeError):
    def __init__(self, cause: str = "runtime_incompatible") -> None:
        super().__init__(cause)
        self.cause = cause


class Cancelled(RuntimeFailed):
    """The operator asked to stop this installation; it stopped at a safe point, leaving only its own files."""

    def __init__(self) -> None:
        super().__init__("cancelled")


# Cooperative cancellation (#351): the thread that runs an install tells which question to ask between steps.
_STOP = threading.local()


@contextlib.contextmanager
def stop_when(predicate: Any) -> Any:
    _STOP.predicate = predicate
    try:
        yield
    finally:
        _STOP.predicate = None


def checkpoint() -> None:
    predicate = getattr(_STOP, "predicate", None)
    if predicate is not None and predicate():
        raise Cancelled()


def validate(manifest: Any, engine: str, kind: str) -> None:
    """Defense in depth on an authenticated server order; never interpret an import path."""
    def require(condition: Any) -> None:
        if not condition:
            raise ValueError()
    try:
        architecture, kinds = ADAPTERS[engine]
        require(isinstance(manifest, dict) and type(manifest["schema"]) is int and manifest["schema"] == 1)
        require(_REQUIRED <= set(manifest) <= _REQUIRED | _OPTIONAL)
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
        # Optional constraints and contracts; a manifest without them stays valid (legacy).
        if "ram_bytes" in manifest:
            require(type(manifest["ram_bytes"]) is int and 0 <= manifest["ram_bytes"] <= 1 << 40)
        if "capabilities" in manifest:
            caps = manifest["capabilities"]
            require(isinstance(caps, list) and len(caps) <= 4 and len(set(caps)) == len(caps))
            require(all(isinstance(c, str) and c in _CAPABILITIES for c in caps) and (not caps or kind in caps))
        for contract in ("input_contract", "output_contract"):
            if contract in manifest:
                require(isinstance(manifest[contract], str) and (manifest[contract] == "" or _CONTRACT.fullmatch(manifest[contract])))
        if "cpu_fallback" in manifest:
            require(type(manifest["cpu_fallback"]) is bool)
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

GPU_DEVICE_NODES = ("/dev/nvidiactl", "/dev/dxg")  # native Linux, WSL2


def gpu_state(exists: Any = os.path.exists) -> tuple[str, int, int] | None:  # pragma: no cover - needs torch
    """(usable CUDA version or "", total VRAM, free VRAM); None when it cannot be told (a GPU may exist, torch is absent)."""
    try:
        import torch  # noqa: PLC0415 - lazy: the Worker must run without the ML stack
    except Exception:  # noqa: BLE001
        return None if any(exists(node) for node in GPU_DEVICE_NODES) else ("", 0, 0)
    try:
        if not torch.cuda.is_available():
            return "", 0, 0
        free, total = torch.cuda.mem_get_info()
        return str(torch.version.cuda or ""), int(total), int(free)
    except Exception:  # noqa: BLE001
        return "", 0, 0


def cpu_fallback_enabled(environ: Mapping[str, str]) -> bool:
    """The Worker's own configuration; it is never enough alone, the release contract must allow it too."""
    return environ.get(CPU_FALLBACK_ENV, "").strip() == "1"


def hardware(environ: Mapping[str, str], gpu: Any = gpu_state) -> dict[str, Any] | None:
    """What this machine offers, as reported to Snitch. None when the GPU cannot be assessed: Snitch then
    does not constrain the Worker and the Worker checks by itself before installing. Observation only."""
    state = gpu()
    if state is None:
        return None
    cuda, vram, _free = state
    try:
        ram = int(os.sysconf("SC_PAGE_SIZE") * os.sysconf("SC_PHYS_PAGES"))
    except (AttributeError, ValueError, OSError):
        ram = 0
    probe = root(environ)
    while not probe.exists() and probe.parent != probe:
        probe = probe.parent
    try:
        disk = shutil.disk_usage(probe).free
    except OSError:
        disk = 0
    return {"python": f"{sys.version_info[0]}.{sys.version_info[1]}", "platform": f"{platform.system().lower()}-{platform.machine()}",
            "cuda": cuda, "ram_bytes": max(ram, 0), "vram_bytes": vram if cuda else 0, "disk_free_bytes": max(disk, 0),
            "cpu_fallback": cpu_fallback_enabled(environ)}


def check_resources(manifest: Mapping[str, Any], hw: Mapping[str, Any] | None) -> None:
    """Refuse before any download with a stable cause. Unknown values never refuse; the runtime probe decides CUDA and VRAM."""
    if hw is None:
        return
    ram = manifest.get("ram_bytes", 0)
    if ram and hw.get("ram_bytes", 0) and ram > hw["ram_bytes"]:
        raise RuntimeFailed("insufficient_ram")
    if manifest["cuda"] != "cpu" and hw.get("cuda") == "" and not (manifest.get("cpu_fallback") is True and hw.get("cpu_fallback")):
        raise RuntimeFailed("cuda_incompatible")


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
                checkpoint()
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
        raise RuntimeFailed("platform_incompatible")
    target = location(manifest, environ)
    with _LOCK, _volume_lock(target.parent):
        if installed(manifest, environ):
            return target
        check_resources(manifest, hardware(environ))
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
                    checkpoint()
                    _download(wheel, wheels / wheel["name"])
                _wheel_budget(wheels, manifest["disk_bytes"] - sum(w["size"] for w in manifest["wheels"]))
                # A model order reports one "verifying" step; an engine order tells installing and verifying apart.
                checkpoint()
                report("installing" if detailed else "verifying")
                subprocess.run([sys.executable, "-m", "venv", str(target)], env=env, check=True,
                               stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, timeout=120)
                checkpoint()
                python = str(target / "bin/python")
                # Wheel paths are explicit and already hashed. --no-deps forbids a second resolver.
                subprocess.run([python, "-I", "-m", "pip", "--isolated", "install", "--no-index", "--no-deps",
                                "--only-binary=:all:", *[str(wheels / w["name"]) for w in manifest["wheels"]]],
                               env=env, check=True, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, timeout=1800)
                checkpoint()
                subprocess.run([python, "-I", "-m", "pip", "--isolated", "check"], env=env, check=True,
                               stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, timeout=60)
                checkpoint()
                if detailed:
                    report("verifying")
                # Protocol import proves the approved adapter is usable before recording the runtime.
                module = {"pyannote": "pyannote.audio", "nemo": "nemo.collections.asr.models",
                          "nemo-titanet": "nemo.collections.asr.models", "speechbrain-ecapa": "speechbrain.lobes.models.ECAPA_TDNN"}[manifest["engine"]]
                subprocess.run([python, "-I", "-c", "import importlib; importlib.import_module(" + repr(module) + ")"],
                               env=env, check=True, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, timeout=120)
                # Hardware is checked in this runtime, not against the base image's torch.
                # Exit 2: CUDA absent or another version. Exit 3: not enough VRAM. A CUDA release without a GPU only
                # passes when its contract and this Worker's configuration both allow the CPU.
                probe = ("import sys,torch; cuda=sys.argv[1]; need=int(sys.argv[2]); cpu_ok=sys.argv[3]=='1'\n"
                         "if cuda=='cpu': sys.exit(0)\n"
                         "if not torch.cuda.is_available(): sys.exit(0 if cpu_ok else 2)\n"
                         "if torch.version.cuda!=cuda: sys.exit(2)\n"
                         "sys.exit(0 if torch.cuda.get_device_properties(0).total_memory>=need else 3)")
                allowed = "1" if manifest.get("cpu_fallback") is True and cpu_fallback_enabled(environ) else "0"
                verdict = subprocess.run([python, "-I", "-c", probe, manifest["cuda"], str(manifest["vram_bytes"]), allowed], env=env,
                                         stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, timeout=120).returncode
                if verdict != 0:
                    raise RuntimeFailed({2: "cuda_incompatible", 3: "insufficient_vram"}.get(verdict, "runtime_incompatible"))
            checkpoint()
            (target / ".ready").write_text(identity(manifest))
        except RuntimeFailed:
            raise
        except OSError as exc:
            raise RuntimeFailed("disk_full" if exc.errno == 28 else "runtime_incompatible") from None
        except (subprocess.SubprocessError, ValueError):
            raise RuntimeFailed() from None
    return target


def cleanup(manifest: dict[str, Any], environ: Mapping[str, str]) -> bool:
    """Remove the incomplete directory of this exact release, and nothing else.

    Attributable files are those under the digest-qualified location of the manifest. A validated runtime (ready
    marker matching its identity, interpreter present) is never removed, nor is anything outside the runtime
    root. Temporary wheel directories are shared by every install of an engine and are not attributable.
    Returns whether something was removed."""
    validate(manifest, manifest.get("engine", ""), manifest.get("kind", ""))
    target = location(manifest, environ)
    base = root(environ).resolve()
    with _LOCK, _volume_lock(target.parent):
        if installed(manifest, environ) or not (target.exists() or target.is_symlink()):
            return False
        if target.is_symlink():
            target.unlink()
            return True
        if base not in target.resolve().parents:
            return False
        shutil.rmtree(target)
        return True


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


def referenced(manifest: Mapping[str, Any], environ: Mapping[str, str]) -> bool:
    """Whether a model on this volume is still bound to this exact runtime. Read-only."""
    wanted = identity(manifest)
    try:
        bindings = list(Path(voice_models.models_root(environ)).rglob(".snitch-runtime.json"))
    except OSError:
        return True  # what cannot be read cannot prove that the runtime is free
    for binding in bindings:
        try:
            if identity(json.loads(binding.read_text())) == wanted:
                return True
        except (OSError, ValueError, TypeError):
            continue
    return False


def uninstall(manifest: dict[str, Any], environ: Mapping[str, str]) -> None:
    """Remove the one runtime the manifest designates. Idempotent: an absent runtime is already removed.

    Refused while a model still points to it. Other releases of the engine, wheels and models are untouched."""
    validate(manifest, manifest.get("engine", ""), manifest.get("kind", ""))
    target = location(manifest, environ)
    with _LOCK, _volume_lock(target.parent):
        if referenced(manifest, environ):
            raise RuntimeFailed("runtime_in_use")
        if target.is_symlink():
            raise RuntimeFailed()
        if target.exists():
            shutil.rmtree(target)


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
