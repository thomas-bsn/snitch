"""Marker written by the explicit bootstrap (``snitch_worker.bootstrap``) next to the models.

It records which repositories and revisions were downloaded, so the worker can read the
embedding model version when ``SNITCH_VOICE_EMBEDDING_MODEL_VERSION`` is absent and the
bootstrap can skip a download when the models are already present. Standard library only:
the worker imports this at startup.
"""

from __future__ import annotations

import json
import os
from typing import Any, Mapping

MANIFEST_NAME = ".snitch-voice-models.json"
SCHEMA = 1
DIARIZATION_PATH_ENV = "SNITCH_VOICE_DIARIZATION_MODEL_PATH"
EMBEDDING_PATH_ENV = "SNITCH_VOICE_EMBEDDING_MODEL_PATH"
DEFAULT_DIARIZATION_PATH = "/models/diarization"
DEFAULT_EMBEDDING_PATH = "/models/embedding"


def manifest_path(environ: Mapping[str, str]) -> str:
    """The marker lives in the directory that holds the embedding model directory."""
    embedding = environ.get(EMBEDDING_PATH_ENV, "").strip() or DEFAULT_EMBEDDING_PATH
    return os.path.join(os.path.dirname(os.path.normpath(embedding)), MANIFEST_NAME)


def read_manifest(environ: Mapping[str, str]) -> dict[str, Any] | None:
    try:
        with open(manifest_path(environ), encoding="utf-8") as handle:
            data = json.load(handle)
    except (OSError, ValueError):
        return None
    return data if isinstance(data, dict) and data.get("schema") == SCHEMA else None


def read_embedding_version(environ: Mapping[str, str]) -> str:
    manifest = read_manifest(environ)
    if not manifest:
        return ""
    embedding = manifest.get("embedding")
    version = embedding.get("revision") if isinstance(embedding, dict) else ""
    return version.strip() if isinstance(version, str) else ""


def write_manifest(environ: Mapping[str, str], data: dict[str, Any]) -> None:
    path = manifest_path(environ)
    temporary = path + ".tmp"
    with open(temporary, "w", encoding="utf-8") as handle:
        json.dump({"schema": SCHEMA, **data}, handle, indent=2, sort_keys=True)
    os.replace(temporary, path)
