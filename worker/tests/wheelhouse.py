"""A tiny offline stand-in for a package index (#370): a directory of pure-python wheels and one failing sdist,
built with the standard library only. pip reads it with --no-index --find-links, so no test needs the network.

Distributions (all synthetic):
  nemo-toolkit 3.0.0   provides nemo.collections.asr.models; extra "asr" pulls tinydep 1.0
  nemo-toolkit 5.0.0   needs pkg-a==1.0 and pkg-b, while pkg-b 1.0 needs pkg-a==2.0  -> resolution impossible
  nemo-toolkit 6.0.0   only as an sdist whose in-tree build backend fails            -> build failure
  nemo-toolkit 7.0.0   needs a distribution the directory does not hold              -> not found
"""
from __future__ import annotations

import base64
import hashlib
import io
import tarfile
import zipfile
from pathlib import Path


def _normal(name: str) -> str:
    return name.replace("-", "_").replace(".", "_")


def build_wheel(directory: Path, name: str, version: str, files: dict[str, str] | None = None,
                requires: tuple[str, ...] = (), extras: dict[str, tuple[str, ...]] | None = None) -> Path:
    """One pure-python wheel with valid METADATA, WHEEL and RECORD."""
    stem = f"{_normal(name)}-{version}"
    dist_info = f"{stem}.dist-info"
    metadata = [f"Metadata-Version: 2.1", f"Name: {name}", f"Version: {version}"]
    for requirement in requires:
        metadata.append(f"Requires-Dist: {requirement}")
    for extra, extra_requires in (extras or {}).items():
        metadata.append(f"Provides-Extra: {extra}")
        for requirement in extra_requires:
            metadata.append(f'Requires-Dist: {requirement}; extra == "{extra}"')
    content = {path: text.encode() for path, text in (files or {f"{_normal(name)}/__init__.py": ""}).items()}
    content[f"{dist_info}/METADATA"] = ("\n".join(metadata) + "\n").encode()
    content[f"{dist_info}/WHEEL"] = b"Wheel-Version: 1.0\nGenerator: synthetic\nRoot-Is-Purelib: true\nTag: py3-none-any\n"
    record = []
    for path, data in content.items():
        digest = base64.urlsafe_b64encode(hashlib.sha256(data).digest()).rstrip(b"=").decode()
        record.append(f"{path},sha256={digest},{len(data)}")
    record.append(f"{dist_info}/RECORD,,")
    content[f"{dist_info}/RECORD"] = ("\n".join(record) + "\n").encode()
    target = directory / f"{stem}-py3-none-any.whl"
    with zipfile.ZipFile(target, "w") as archive:
        for path, data in content.items():
            archive.writestr(path, data)
    return target


def build_failing_sdist(directory: Path, name: str, version: str) -> Path:
    """A source distribution whose in-tree build backend always fails, without needing any build dependency."""
    stem = f"{name}-{version}"
    files = {
        "pyproject.toml": '[build-system]\nrequires = []\nbuild-backend = "failing_backend"\nbackend-path = ["."]\n',
        "failing_backend.py": "def build_wheel(*args, **kwargs):\n    raise RuntimeError('synthetic compilation error')\n",
        "PKG-INFO": f"Metadata-Version: 2.1\nName: {name}\nVersion: {version}\n",
    }
    target = directory / f"{stem}.tar.gz"
    with tarfile.open(target, "w:gz") as archive:
        for path, text in files.items():
            data = text.encode()
            info = tarfile.TarInfo(f"{stem}/{path}")
            info.size = len(data)
            archive.addfile(info, io.BytesIO(data))
    return target


def build_nemo_wheelhouse(directory: Path) -> Path:
    directory.mkdir(parents=True, exist_ok=True)
    asr = {
        "nemo/__init__.py": "",
        "nemo/collections/__init__.py": "",
        "nemo/collections/asr/__init__.py": "",
        "nemo/collections/asr/models.py": "READY = True\n",
    }
    build_wheel(directory, "tinydep", "1.0")
    build_wheel(directory, "nemo-toolkit", "3.0.0", asr, extras={"asr": ("tinydep==1.0",)})
    build_wheel(directory, "nemo-toolkit", "3.1.0", asr, extras={"asr": ("tinydep==1.0",)})
    build_wheel(directory, "pkg-a", "1.0")
    build_wheel(directory, "pkg-a", "2.0")
    build_wheel(directory, "pkg-b", "1.0", requires=("pkg-a==2.0",))
    build_wheel(directory, "nemo-toolkit", "5.0.0", asr, requires=("pkg-a==1.0", "pkg-b"), extras={"asr": ()})
    build_failing_sdist(directory, "nemo-toolkit", "6.0.0")
    build_wheel(directory, "nemo-toolkit", "7.0.0", asr, requires=("not-in-the-house==1.0",), extras={"asr": ()})
    return directory


def build_speechbrain_wheelhouse(directory: Path) -> Path:
    """The SpeechBrain family (#374): two versions that provide the module the install probe imports, and one that
    does not (pip accepts it, but the family's adapters could not run in it)."""
    directory.mkdir(parents=True, exist_ok=True)
    package = {
        "speechbrain/__init__.py": "",
        "speechbrain/lobes/__init__.py": "",
        "speechbrain/lobes/models/__init__.py": "",
        "speechbrain/lobes/models/ECAPA_TDNN.py": "READY = True\n",
    }
    build_wheel(directory, "speechbrain", "1.0.3", package)
    build_wheel(directory, "speechbrain", "1.1.0", package)
    build_wheel(directory, "speechbrain", "2.0.0", {"speechbrain/__init__.py": ""})
    # The real SpeechBrain declares torch and torchaudio unbounded. Two releases of each, so that a test can tell the
    # newest one (2.9.1, which no longer has the APIs SpeechBrain imports) from the one the family pins (2.8.0).
    for name in ("torch", "torchaudio"):
        build_wheel(directory, name, "2.8.0", {f"{name}/__init__.py": ""})
        build_wheel(directory, name, "2.9.1", {f"{name}/__init__.py": ""})
    return directory


def build_wespeaker_wheelhouse(directory: Path) -> Path:
    """The real distribution is `wespeakerruntime` (ONNX inference); `wespeaker` itself is not on PyPI (#375)."""
    directory.mkdir(parents=True, exist_ok=True)
    build_wheel(directory, "wespeakerruntime", "1.0.1", {"wespeakerruntime/__init__.py": "class Speaker:\n    pass\n"})
    build_wheel(directory, "wespeakerruntime", "1.0.0", {"wespeakerruntime/__init__.py": "class Speaker:\n    pass\n"})
    return directory
