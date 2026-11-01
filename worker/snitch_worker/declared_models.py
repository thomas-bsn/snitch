"""Models this worker declares to the application (shown to administrators).

Only non-secret identifiers are sent: provider, model name, version and, for
the transcription endpoint, ``host[:port]``. Credentials, tokens, API keys,
URL schemes, paths, queries and userinfo are never read into the declaration.
Anything the server would refuse is dropped here so a bad value never blocks
the worker's pulse.
"""

from __future__ import annotations

import re
from collections.abc import Mapping
from urllib.parse import urlsplit

from snitch_client.models.worker_model import WorkerModel
from snitch_client.models.worker_models import WorkerModels

from snitch_worker import voice_manifest, voice_models

_REVISION_LENGTH = 12

_NAME = re.compile(r"^[A-Za-z0-9][A-Za-z0-9_.:/+-]{0,127}$")
_VERSION = re.compile(r"^[A-Za-z0-9][A-Za-z0-9_.+-]{0,63}$")
_ENDPOINT = re.compile(r"^[A-Za-z0-9]([A-Za-z0-9.-]{0,251}[A-Za-z0-9])?(:[0-9]{1,5})?$")
_SECRET = re.compile(
    r"(?i)(^|[^a-z0-9])(hf_|sk-|rw_|ghp_|xox[a-z]-|bearer|token|secret|password|passwd|api[_-]?key)"
    r"|[A-Za-z0-9]{32,}"
)


def _clean(value: str, pattern: re.Pattern[str]) -> str | None:
    value = value.strip()
    if not value or _SECRET.search(value) or not pattern.match(value):
        return None
    return value


def _host_port(url: str) -> str | None:
    """Return ``host[:port]`` from an endpoint URL, discarding everything else."""
    url = url.strip()
    if not url:
        return None
    try:
        parts = urlsplit(url if "://" in url else f"//{url}")
        host = parts.hostname
        port = parts.port
    except ValueError:
        return None
    if not host:
        return None
    endpoint = host if port is None else f"{host}:{port}"
    if port is not None and not 1 <= port <= 65535:
        return None
    return _clean(endpoint, _ENDPOINT)


def _model(
    provider: str, model: str, version: str = "", endpoint: str | None = None
) -> WorkerModel | None:
    provider_clean = _clean(provider, _NAME)
    model_clean = _clean(model, _NAME)
    if provider_clean is None or model_clean is None:
        return None
    declared = WorkerModel(provider=provider_clean, model=model_clean)
    version_clean = _clean(version, _VERSION) if version.strip() else None
    if version_clean is not None:
        declared.version = version_clean
    if endpoint:
        declared.endpoint = endpoint
    return declared


def declared_models(env: Mapping[str, str], active_diarization: str = "") -> WorkerModels:
    """Build the declaration from the worker environment; absent roles stay unset."""
    models = WorkerModels()
    transcription = _model(
        "openai-compatible",
        env.get("TRANSCRIPTION_MODEL", ""),
        endpoint=_host_port(env.get("TRANSCRIPTION_API_URL", "")),
    )
    if transcription is not None:
        models.transcription = transcription
    # Voice roles follow the real provider's own settings (voice_pyannote.py):
    # a role is declared only when its local model path is configured.
    # What the models volume really holds (written by the bootstrap) wins over the environment.
    manifest = voice_manifest.read_manifest(env) or {}
    voice_roles = (
        ("diarization", "diarization", "SNITCH_VOICE_DIARIZATION", "pyannote/speaker-diarization-community-1"),
        ("fingerprint", "embedding", "SNITCH_VOICE_EMBEDDING", "pyannote/wespeaker-voxceleb-resnet34-LM"),
    )
    # A verified copy of the active diarization model on the models volume wins over the legacy layout.
    held = voice_models.ready_revision(env, active_diarization) if active_diarization else ""
    if held:
        voice = _model(voice_models.engine_of(voice_models.model_path(env, active_diarization, held)), active_diarization, held[:_REVISION_LENGTH])
        if voice is not None:
            models.diarization = voice
    for role, manifest_key, prefix, default_name in voice_roles:
        if role == "diarization" and held:
            continue
        if not env.get(prefix + "_MODEL_PATH", "").strip():
            continue
        entry = manifest.get(manifest_key)
        entry = entry if isinstance(entry, dict) else {}
        repo = entry.get("repo") if isinstance(entry.get("repo"), str) else ""
        revision = entry.get("revision") if isinstance(entry.get("revision"), str) else ""
        voice = _model(
            "pyannote",
            repo.strip() or env.get(prefix + "_MODEL_NAME", "").strip() or default_name,
            # A full commit SHA looks like a secret to the server filter: send its short form.
            revision.strip()[:_REVISION_LENGTH] or env.get(prefix + "_MODEL_VERSION", ""),
        )
        if voice is not None:
            setattr(models, role, voice)
    return models
