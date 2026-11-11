"""Approved, immutable runtimes. Installation is called only by an explicit model order.

Signed path: all packages are wheels named and hashed by the trusted publisher. pip resolves
nothing and runs no source build. On-demand path (#370, ADR 0056): pip resolves a library of a
closed family table by name and version, `pip freeze` is recorded, and pip's output is only ever
mapped to stable causes. Inference uses this venv, never the Worker's ML environment. Failed
revisions stay separate from the previous runtime.
"""
from __future__ import annotations

import contextlib
import hashlib
import json
import logging
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

from snitch_worker import log, voice_models
from snitch_worker.voice_download import without_token

ADAPTERS = {"pyannote": ("pyannote", {"diarization", "fingerprint"}), "nemo": ("sortformer", {"diarization"}),
            "nemo-titanet": ("titanet", {"fingerprint"}), "speechbrain-ecapa": ("ecapa-tdnn", {"fingerprint"})}
_VERSION = re.compile(r"^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$")
# An engine installed from a pinned commit: the version is "git-" and the full commit. Only a family whose table entry
# names a repository accepts it, and the repository is that entry's, never the order's.
_GIT_VERSION = re.compile(r"^git-([0-9a-f]{40})$")
_SHA = re.compile(r"^[0-9a-f]{64}$")
_LOCK = threading.RLock()
_REQUIRED = {"schema", "engine", "version", "adapter_version", "python", "platform", "cuda", "license", "kind", "architecture", "disk_bytes", "vram_bytes", "wheels"}
_OPTIONAL = {"ram_bytes", "capabilities", "input_contract", "output_contract", "cpu_fallback"}
# Source separation is not an approved capability: the V2 separation stays inactive.
_CAPABILITIES = {"diarization", "fingerprint"}
_CONTRACT = re.compile(r"^[a-z0-9][a-z0-9._-]{0,63}$")
LOG_TAIL_BYTES = 1500  # the end of a failed step's stderr that reaches the Worker log (scrubbed by log.py)
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
    """Historical setting reported for compatibility; it never permits a CUDA release to run on CPU."""
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
    if manifest["cuda"] not in ("cpu", "") and hw.get("cuda") == "":
        raise RuntimeFailed("cuda_incompatible")


def identity(manifest: Mapping[str, Any]) -> str:
    return hashlib.sha256(json.dumps(manifest, sort_keys=True, separators=(",", ":")).encode()).hexdigest()


def root(environ: Mapping[str, str]) -> Path:
    return Path(voice_models.models_root(environ)).parent / "runtimes"


def location(manifest: Mapping[str, Any], environ: Mapping[str, str]) -> Path:
    if is_on_demand(manifest):
        return root(environ) / manifest["engine"] / _slot(manifest)
    return root(environ) / manifest["engine"] / identity(manifest)


def installed(manifest: Mapping[str, Any], environ: Mapping[str, str]) -> bool:
    path = location(manifest, environ)
    if is_on_demand(manifest):
        try:
            return (path / ".ready").read_text() == _on_demand_identity(path, manifest) and (path / "bin/python").is_file()
        except (OSError, ValueError):
            return False
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


def _step(name: str, command: list[str], env: Mapping[str, str], timeout: int, check: bool = True) -> int:
    """Run one install step. A failure is logged with its step and the scrubbed end of its stderr: without it every
    cause collapses into "runtime_incompatible" and the operator cannot tell what broke."""
    try:
        done = subprocess.run(command, env=env, stdin=subprocess.DEVNULL, stdout=subprocess.DEVNULL, stderr=subprocess.PIPE, timeout=timeout)
    except subprocess.TimeoutExpired:
        log.emit(logging.WARNING, "engine_install_step_failed", step=name, reason_code="timeout", timeout_seconds=timeout)
        raise
    if done.returncode != 0:
        tail = done.stderr[-LOG_TAIL_BYTES:].decode("utf-8", "replace") if isinstance(done.stderr, bytes) else ""
        log.emit(logging.WARNING, "engine_install_step_failed", step=name, exit_code=done.returncode, stderr_tail=tail)
        if check:
            raise subprocess.CalledProcessError(done.returncode, command)
    return done.returncode


def install(manifest: dict[str, Any], environ: Mapping[str, str], report: Any, detailed: bool = False) -> Path:
    if is_on_demand(manifest):
        return _install_on_demand(manifest, environ, report)
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
                _step("venv", [sys.executable, "-m", "venv", str(target)], env, 120)
                checkpoint()
                python = str(target / "bin/python")
                # Wheel paths are explicit and already hashed. --no-deps forbids a second resolver.
                _step("pip_install", [python, "-I", "-m", "pip", "--isolated", "install", "--no-index", "--no-deps",
                                      "--only-binary=:all:", *[str(wheels / w["name"]) for w in manifest["wheels"]]], env, 1800)
                checkpoint()
                _step("pip_check", [python, "-I", "-m", "pip", "--isolated", "check"], env, 60)
                checkpoint()
                if detailed:
                    report("verifying")
                # Protocol import proves the approved adapter is usable before recording the runtime.
                module = {"pyannote": "pyannote.audio", "nemo": "nemo.collections.asr.models",
                          "nemo-titanet": "nemo.collections.asr.models", "speechbrain-ecapa": "speechbrain.lobes.models.ECAPA_TDNN"}[manifest["engine"]]
                _step("import_adapter", [python, "-I", "-c", "import importlib; importlib.import_module(" + repr(module) + ")"], env, 120)
                # Hardware is checked in this runtime, not against the base image's torch.
                # Exit 2: CUDA absent or another version. Exit 3: not enough VRAM. A CUDA release without a GPU
                # is refused: CPU execution requires a distinct approved CPU release.
                probe = ("import sys,torch; cuda=sys.argv[1]; need=int(sys.argv[2])\n"
                         "if cuda=='cpu': sys.exit(0)\n"
                         "if not torch.cuda.is_available(): sys.exit(2)\n"
                         "if torch.version.cuda!=cuda: sys.exit(2)\n"
                         "sys.exit(0 if torch.cuda.get_device_properties(0).total_memory>=need else 3)")
                verdict = _step("cuda_probe", [python, "-I", "-c", probe, manifest["cuda"], str(manifest["vram_bytes"])], env, 120, check=False)
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


# -- On-demand engines (#370, ADR 0056) -----------------------------------------------------------------------------
# The closed family table of the Worker mirrors Snitch's: an order names a family, never a package to run. Anything
# not in this table, or that differs from it, is refused. Pip arguments are built only from these values and a
# version that matches _VERSION (it cannot start with "-"), so an order cannot inject a pip option.
FAMILIES = {"nemo": {"engine": "nemo-toolkit", "distribution": "nemo-toolkit", "extras": "asr", "kinds": ("diarization", "fingerprint"),
                     "probe": "nemo.collections.asr.models", "git": "https://github.com/NVIDIA/NeMo.git"},
            # SpeechBrain (#374): ECAPA-TDNN fingerprints run through a generic adapter, SepFormer separation through its
            # own (#401, voice_sepformer). Being ready opens no production job: no step of an analysis separates.
            # SpeechBrain declares `torchaudio` with no upper bound, so pip takes the newest one; from torchaudio 2.9 the
            # APIs it still calls at import (`list_audio_backends`) are gone, and the import fails. The pins are fixed
            # values of this table, never taken from an order; raise them together with a version that supports more.
            # torchaudio.load (engine_bridge) needs a decoding backend inside this venv: without `soundfile` it only has
            # ffmpeg, which depends on the host's FFmpeg 4-6 shared libraries. Same version as the signed Release lock.
            "speechbrain": {"engine": "speechbrain", "distribution": "speechbrain", "extras": "", "kinds": ("fingerprint", "separation"),
                            "probe": "speechbrain.lobes.models.ECAPA_TDNN", "pins": ("torch<2.9", "torchaudio<2.9",
                                                                                  "soundfile==0.14.0")},
            # WeSpeaker is not published on PyPI; its ONNX inference package is `wespeakerruntime` (#375). Fingerprint only.
            "wespeaker": {"engine": "wespeaker", "distribution": "wespeakerruntime", "extras": "", "kinds": ("fingerprint",),
                          "probe": "wespeakerruntime"}}
_ON_DEMAND_ENGINES = {family["engine"] for family in FAMILIES.values()}
# Kinds a family installs for but no adapter runs: such a kind gets an engine and no model (ADR 0056). None today.
# Mirrors ActiveKinds of Snitch's table.
_INACTIVE_KINDS: dict[str, tuple[str, ...]] = {}
# Adapters that only exist as a signed Release (no embedded copy), and every engine name a fingerprint speaker
# adapter runs: those plus the families that cover fingerprint only. Mirrors signedOnlyAdapters of Snitch's table;
# scripts/check_engine_families.py fails the build when this file and the Go table drift apart.
SIGNED_ONLY_ADAPTERS = frozenset({"nemo-titanet", "speechbrain-ecapa"})
SPEAKER_ENGINES = SIGNED_ONLY_ADAPTERS | {family["engine"] for family in FAMILIES.values() if family["kinds"] == ("fingerprint",)}


def runs_kind(runtime: Any) -> bool:
    """True for an on-demand runtime whose kind is run by an adapter of its family."""
    return is_on_demand(runtime) and runtime.get("kind") not in _INACTIVE_KINDS.get(runtime.get("family"), ())
_ON_DEMAND_FIELDS = {"source", "family", "engine", "distribution", "version", "kind", "kinds", "python", "platform", "cuda",
                     "disk_bytes", "vram_bytes", "ram_bytes", "estimated_seconds"}
# Where packages come from is the operator's choice, read from the Worker's own environment (default: PyPI) and
# never forwarded to a child process. Tests point it to a local directory with --no-index.
INDEX_URL_ENV, FIND_LINKS_ENV, NO_INDEX_ENV = "SNITCH_ENGINE_INDEX_URL", "SNITCH_ENGINE_FIND_LINKS", "SNITCH_ENGINE_NO_INDEX"
PIP_TIMEOUT_SECONDS = 3600


def is_on_demand(runtime: Any) -> bool:
    return isinstance(runtime, dict) and runtime.get("source") == "on_demand"


def validate_on_demand(runtime: Any) -> None:
    def require(condition: Any) -> None:
        if not condition:
            raise ValueError()
    try:
        require(isinstance(runtime, dict) and set(runtime) == _ON_DEMAND_FIELDS and runtime["source"] == "on_demand")
        family = FAMILIES[runtime["family"]]
        require(runtime["engine"] == family["engine"] and runtime["distribution"] == family["distribution"])
        require(isinstance(runtime["version"], str) and _VERSION.fullmatch(runtime["version"]))
        if runtime["version"].startswith("git-"):
            require(family.get("git") and _GIT_VERSION.fullmatch(runtime["version"]))
        require(runtime["kind"] in family["kinds"] and runtime["kinds"] == list(family["kinds"]))
        require(runtime["python"] == "3.12" and runtime["platform"] == "linux-x86_64" and runtime["cuda"] == "")
        for field in ("disk_bytes", "vram_bytes", "ram_bytes", "estimated_seconds"):
            require(type(runtime[field]) is int and 0 <= runtime[field] <= 1 << 40)
        require(runtime["disk_bytes"] > 0)
    except (KeyError, TypeError, ValueError, AttributeError):
        raise RuntimeFailed() from None


def validate_runtime(runtime: Any) -> None:
    """Validate a runtime of either kind: a signed manifest or an on-demand description."""
    if is_on_demand(runtime):
        validate_on_demand(runtime)
    elif isinstance(runtime, dict):
        validate(runtime, runtime.get("engine", ""), runtime.get("kind", ""))
    else:
        raise RuntimeFailed()


def _slot(runtime: Mapping[str, Any]) -> str:
    """The directory of a family and version: one environment per version, whatever pip resolves into it."""
    return hashlib.sha256(f"{runtime['family']}\n{runtime['version']}".encode()).hexdigest()


def _on_demand_identity(path: Path, runtime: Mapping[str, Any] | None = None) -> str:
    """Local identity of an installed on-demand engine: family, version and the hash of the recorded pip freeze."""
    meta = json.loads((path / "ondemand.json").read_text()) if runtime is None else {"family": runtime["family"], "version": runtime["version"]}
    freeze = hashlib.sha256((path / "pip-freeze.txt").read_bytes()).hexdigest()
    return hashlib.sha256(json.dumps({"family": meta["family"], "version": meta["version"], "freeze_sha256": freeze},
                                     sort_keys=True, separators=(",", ":")).encode()).hexdigest()


def local_identity(runtime: Mapping[str, Any], environ: Mapping[str, str]) -> str:
    if not is_on_demand(runtime):
        return identity(runtime)
    try:
        return _on_demand_identity(location(runtime, environ), runtime)
    except OSError:
        raise RuntimeFailed() from None


def _index_arguments(environ: Mapping[str, str]) -> list[str]:
    arguments = ["--retries=2", "--timeout=30"]
    index, links = environ.get(INDEX_URL_ENV, "").strip(), environ.get(FIND_LINKS_ENV, "").strip()
    if environ.get(NO_INDEX_ENV, "").strip() == "1":
        arguments.append("--no-index")
    elif index:
        arguments.append("--index-url=" + index)
    if links:
        arguments.append("--find-links=" + links)
    return arguments


def classify_pip(output: str) -> str:
    """Map pip's output to a stable cause. Only that cause is sent to Snitch; the end of the text goes to the Worker log, scrubbed."""
    text = output.lower()
    if "no space left on device" in text:
        return "disk_full"
    if "resolutionimpossible" in text or "conflicting dependencies" in text or "the conflict is caused by" in text:
        return "resolution_failed"
    if "failed building wheel" in text or "subprocess-exited-with-error" in text or "failed to build" in text or "getting requirements to build" in text:
        return "build_failed"
    if any(mark in text for mark in ("failed to establish a new connection", "temporary failure in name resolution", "name or service not known",
                                     "connection timed out", "read timed out", "max retries exceeded", "could not fetch url", "network is unreachable",
                                     "proxyerror", "sslerror", "connection refused")):
        return "hf_network"
    if "no matching distribution" in text or "could not find a version" in text:
        return "package_not_found"
    return "runtime_incompatible"


def _pip(python: str, arguments: list[str], env: Mapping[str, str], timeout: int) -> str:
    """Run pip in the engine environment with its output captured; it is returned only to be classified or recorded as the freeze."""
    step = "pip_" + arguments[0]
    try:
        done = subprocess.run([python, "-I", "-m", "pip", "--isolated", *arguments], env=env, stdin=subprocess.DEVNULL,
                              stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True, timeout=timeout)
    except subprocess.TimeoutExpired:
        log.emit(logging.WARNING, "engine_install_step_failed", step=step, reason_code="timeout", timeout_seconds=timeout)
        raise RuntimeFailed("hf_network") from None
    if done.returncode != 0:
        cause = classify_pip(done.stdout or "")
        # The operator's only way to tell a real incompatibility from an unrecognised pip message: the end of the
        # output, scrubbed by log.py (tokens, credentialed URLs, paths), never sent to Snitch as a cause.
        log.emit(logging.WARNING, "engine_install_step_failed", step=step, exit_code=done.returncode, cause_code=cause,
                 stderr_tail=(done.stdout or "")[-LOG_TAIL_BYTES:])
        raise RuntimeFailed(cause)
    return done.stdout or ""


def _normal(name: str) -> str:
    return re.sub(r"[-_.]+", "-", name).lower()


def on_demand_requirements(family: Mapping[str, Any], version: str) -> list[str]:
    """The pip requirements of a family at a version: the library itself, then the fixed pins of its table entry."""
    extras = f"[{family['extras']}]" if family["extras"] else ""
    git = _GIT_VERSION.fullmatch(version)
    if git:
        # A pinned commit of the repository of the table entry; pip fetches it with the git of the Worker image.
        return [f"{family['distribution']}{extras} @ git+{family['git']}@{git.group(1)}", *family.get("pins", ())]
    return [f"{family['distribution']}{extras}=={version}", *family.get("pins", ())]


def _installed_version(freeze: str, family: Mapping[str, Any], version: str) -> bool:
    """Whether the freeze of the new environment shows the library at the version the order named."""
    wanted = _normal(family["distribution"])
    git = _GIT_VERSION.fullmatch(version)
    for line in freeze.splitlines():
        if git and " @ " in line:
            name, source = line.split(" @ ", 1)
            if _normal(name) == wanted and source.strip().endswith("@" + git.group(1)):
                return True
        elif not git and "==" in line:
            name, installed = line.split("==", 1)
            if _normal(name) == wanted and installed.strip() == version:
                return True
    return False


def _install_on_demand(runtime: dict[str, Any], environ: Mapping[str, str], report: Any) -> Path:
    validate_on_demand(runtime)
    family = FAMILIES[runtime["family"]]
    if sys.version_info[:2] != (3, 12) or platform.system() != "Linux" or platform.machine() != "x86_64":
        raise RuntimeFailed("platform_incompatible")
    target = location(runtime, environ)
    with _LOCK, _volume_lock(target.parent):
        if installed(runtime, environ):
            return target
        check_resources(runtime, hardware(environ))
        target.parent.mkdir(parents=True, exist_ok=True)
        if shutil.disk_usage(target.parent).free < runtime["disk_bytes"]:
            raise RuntimeFailed("disk_full")
        if target.is_symlink():
            raise RuntimeFailed()
        if target.exists():
            shutil.rmtree(target)
        target.mkdir()
        env = environment(environ)
        if _GIT_VERSION.fullmatch(runtime["version"]):
            env["GIT_TERMINAL_PROMPT"] = "0"  # a repository that asks for a login fails instead of waiting on a prompt
        requirements = on_demand_requirements(family, runtime["version"])
        try:
            checkpoint()
            report("installing")
            _step("venv", [sys.executable, "-m", "venv", str(target)], env, 120)
            checkpoint()
            python = str(target / "bin/python")
            _pip(python, ["install", "--no-input", "--disable-pip-version-check", *_index_arguments(environ), *requirements], env, PIP_TIMEOUT_SECONDS)
            checkpoint()
            report("verifying")
            try:
                _pip(python, ["check"], env, 120)
            except RuntimeFailed:
                raise RuntimeFailed("resolution_failed") from None
            checkpoint()
            freeze = _pip(python, ["freeze"], env, 120)
            if not _installed_version(freeze, family, runtime["version"]):
                raise RuntimeFailed()
            (target / "pip-freeze.txt").write_text(freeze if freeze.endswith("\n") else freeze + "\n")
            (target / "ondemand.json").write_text(json.dumps({"family": runtime["family"], "version": runtime["version"]}, sort_keys=True))
            checkpoint()
            # The family's library must import before the runtime is recorded. No model weight is read or downloaded.
            _step("import_family", [python, "-I", "-c", "import importlib; importlib.import_module(" + repr(family["probe"]) + ")"], env, 300)
            checkpoint()
            (target / ".ready").write_text(_on_demand_identity(target, runtime))
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
    validate_runtime(manifest)
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
        engines = sorted(path for path in root(environ).iterdir() if path.is_dir() and (path.name in ADAPTERS or path.name in _ON_DEMAND_ENGINES))
    except OSError:
        return found
    for engine in engines:
        try:
            candidates = sorted(engine.iterdir())
        except OSError:
            continue
        for path in candidates:
            try:
                if not (_SHA.fullmatch(path.name) and (path / "bin/python").is_file()):
                    continue
                if engine.name in _ON_DEMAND_ENGINES:
                    proof = _on_demand_identity(path)
                    if (path / ".ready").read_text() == proof:
                        found.append({"engine": engine.name, "local_identity": proof})
                elif (path / ".ready").read_text() == path.name:
                    found.append({"engine": engine.name, "local_identity": path.name})
            except (OSError, ValueError):
                continue
    return found


def referenced(manifest: Mapping[str, Any], environ: Mapping[str, str]) -> bool:
    """Whether a model on this volume is still bound to this exact runtime. Read-only."""
    wanted = identity(manifest)
    # An on-demand environment is one directory per family and version, shared by every kind of the family: a model
    # bound to any kind of it keeps the directory, so the comparison ignores the kind.
    shared = (manifest.get("family"), manifest.get("version")) if is_on_demand(manifest) else None
    try:
        bindings = list(Path(voice_models.models_root(environ)).rglob(".snitch-runtime.json"))
    except OSError:
        return True  # what cannot be read cannot prove that the runtime is free
    for binding in bindings:
        try:
            bound = json.loads(binding.read_text())
            if shared is not None and is_on_demand(bound):
                if (bound.get("family"), bound.get("version")) == shared:
                    return True
            elif identity(bound) == wanted:
                return True
        except (OSError, ValueError, TypeError):
            continue
    return False


def uninstall(manifest: dict[str, Any], environ: Mapping[str, str]) -> None:
    """Remove the one runtime the manifest designates. Idempotent: an absent runtime is already removed.

    Refused while a model still points to it. Other releases of the engine, wheels and models are untouched."""
    validate_runtime(manifest)
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
        validate_runtime(value)
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
