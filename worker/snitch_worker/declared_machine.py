"""Non-secret container and hardware information announced with the worker pulse."""
from __future__ import annotations

import os
import platform
import re
import subprocess
import tomllib
from functools import lru_cache
from importlib.metadata import PackageNotFoundError, version
from pathlib import Path

from snitch_client.models.worker_machine import WorkerMachine
from snitch_worker.declared_models import _clean

_VALUE = re.compile(r"^[A-Za-z0-9][A-Za-z0-9 _.,:+()/-]{0,127}$")


def _processor() -> str:
    try:
        for line in Path("/proc/cpuinfo").read_text().splitlines():
            key, separator, value = line.partition(":")
            if separator and key.strip() in {"model name", "Hardware"}:
                return value.strip()
    except OSError:
        pass
    return platform.processor()


@lru_cache(maxsize=1)
def _hardware() -> dict[str, str | int]:
    data: dict[str, str | int] = {}
    cpu = _clean(_processor(), _VALUE)
    if cpu:
        data["processor"] = cpu
    try:
        memory = os.sysconf("SC_PAGE_SIZE") * os.sysconf("SC_PHYS_PAGES")
        # Respect a container memory limit when one is supplied by cgroups v2.
        try:
            limit = Path("/sys/fs/cgroup/memory.max").read_text().strip()
            if limit != "max":
                memory = min(memory, int(limit))
        except (OSError, ValueError):
            pass
        if 0 < memory <= 109951162777600:
            data["memory_bytes"] = memory
    except (OSError, ValueError, AttributeError):
        pass
    try:
        result = subprocess.run(["nvidia-smi", "--query-gpu=name,memory.total,driver_version", "--format=csv,noheader,nounits"], capture_output=True, text=True, timeout=2, check=True)
        name, memory, driver = [part.strip() for part in result.stdout.splitlines()[0].split(",", 2)]
        for key, value in (("gpu_name", name), ("gpu_driver", driver)):
            safe = _clean(value, _VALUE)
            if safe:
                data[key] = safe
        size = int(memory) * 1024 * 1024
        if 0 < size <= 109951162777600:
            data["gpu_memory_bytes"] = size
    except (OSError, ValueError, IndexError, subprocess.SubprocessError):
        pass
    return data


def declared_machine() -> WorkerMachine:
    values = dict(_hardware())
    # A configured name denotes the host; otherwise the hostname is the container's.
    name = _clean(os.environ.get("SNITCH_WORKER_MACHINE_NAME", "").strip() or platform.node(), _VALUE)
    system = _clean(platform.system(), _VALUE)
    try:
        worker_version = _clean(version("snitch-worker"), _VALUE)
    except PackageNotFoundError:
        try:
            project = tomllib.loads((Path(__file__).resolve().parent.parent / "pyproject.toml").read_text())
            worker_version = _clean(str(project["project"]["version"]), _VALUE)
        except (OSError, KeyError, ValueError, TypeError):
            worker_version = None
    for key, value in (("name", name), ("system", system), ("worker_version", worker_version)):
        if value:
            values[key] = value
    return WorkerMachine.from_dict(values)
