"""Install orders from Snitch, run by the Worker and nothing else (ADR 0038).

Snitch owns the desired state: it records an order, the Worker pulls it on its usual outgoing request, checks
it at reception, acknowledges it, runs it with the model manager (download, load, synthetic inference) and
reports what it really holds. The Worker never chooses or downloads a model by itself here.

Checks at reception: recipient (the identity Snitch gave this Worker, pinned on the volume at first sight),
order identity, action, type, engine, model, repository, pinned revision, and idempotence (an order, or an
idempotency key, already seen is never executed twice, even after a restart).

The central Hugging Face token reaches the Worker only with the order of a gated download. It is written to the
private token file (never over a token the operator stored locally) and nowhere else: not in logs, arguments,
reports or the inference environment. Standard library only.
"""

from __future__ import annotations

import json
import logging
import os
import re
import shutil
import tempfile
import threading
from collections.abc import Mapping
from typing import Any

from snitch_worker import hf_token_sync, log, voice_download, voice_manifest, voice_models

_ORDER_ID = re.compile(r"^ord_[A-Za-z0-9_-]{16,64}$")
_WORKER_ID = re.compile(r"^wk_[A-Za-z0-9_-]{20,60}$")
_KEY = re.compile(r"^[A-Za-z0-9._:-]{8,100}$")
_MODEL_ID = re.compile(r"^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$")
# Engines an approved adapter exists for, by type (separation has none yet).
ENGINES = {"diarization": {"pyannote", "nemo"}, "fingerprint": {"pyannote"}, "vad": {"silero-jit"}}
STATE_NAME = "orders.json"
# A download needs room for the whole model; this margin covers what the archive and the cache add on top.
DISK_MARGIN = 1.1


def state_path(environ: Mapping[str, str]) -> str:
    return os.path.join(os.path.dirname(voice_manifest.manifest_path(environ)), STATE_NAME)


def _directory_size(path: str) -> int:
    total = 0
    for root, _dirs, files in os.walk(path):
        for name in files:
            try:
                total += os.path.getsize(os.path.join(root, name))
            except OSError:
                pass
    return total


def free_bytes(path: str) -> int:
    """Free space of the volume that holds (or will hold) path."""
    probe = path
    while probe and not os.path.exists(probe):
        parent = os.path.dirname(probe)
        if parent == probe:
            break
        probe = parent
    return shutil.disk_usage(probe or os.sep).free


class OrderExecutor:
    _free_bytes = staticmethod(free_bytes)

    def __init__(self, manager: voice_models.ModelManager, environ: Mapping[str, str], token_file: str) -> None:
        self._manager = manager
        self._env = environ
        self._token_file = token_file
        self._path = state_path(environ)
        self._lock = threading.Lock()
        self._queue: list[tuple[dict[str, Any], Any]] = []
        self._running: set[str] = set()
        self._thread: threading.Thread | None = None

    # -- persisted state: identity pin, outcomes and keys, so a restart repeats nothing ------------------------

    def _load(self) -> dict[str, Any]:
        try:
            with open(self._path, encoding="utf-8") as handle:
                value = json.load(handle)
        except (OSError, ValueError):
            value = {}
        if not isinstance(value, dict):
            value = {}
        value.setdefault("worker_id", "")
        value.setdefault("orders", {})
        value.setdefault("keys", {})
        return value

    def _save(self, state: dict[str, Any]) -> None:
        directory = os.path.dirname(self._path)
        os.makedirs(directory, exist_ok=True)
        descriptor, temporary = tempfile.mkstemp(dir=directory, prefix=".orders.")
        with os.fdopen(descriptor, "w", encoding="utf-8") as handle:
            json.dump(state, handle)
        os.replace(temporary, self._path)

    # -- reception ---------------------------------------------------------------------------------------------

    @staticmethod
    def _well_formed(order: Mapping[str, Any]) -> bool:
        kind, engine = order.get("kind"), order.get("engine")
        repo, revision = order.get("repo"), order.get("revision")
        return (
            order.get("action") in ("install", "remove")
            and isinstance(kind, str)
            and kind in ENGINES
            and isinstance(engine, str)
            and engine in ENGINES[kind]
            and isinstance(order.get("model_id"), str)
            and bool(_MODEL_ID.match(order["model_id"]))
            and isinstance(repo, str)
            and isinstance(revision, str)
            and voice_models.valid_model(repo, revision)
            and isinstance(order.get("idempotency_key"), str)
            and bool(_KEY.match(order["idempotency_key"]))
        )

    def handle(self, answer: Mapping[str, Any], http: Any) -> None:
        """Check, acknowledge and queue the orders of one pull; execution continues in the background."""
        orders = answer.get("orders")
        worker_id = answer.get("worker_id")
        if not isinstance(orders, list) or not orders or not isinstance(worker_id, str) or not _WORKER_ID.match(worker_id):
            return
        state = self._load()
        if state["worker_id"] and state["worker_id"] != worker_id:
            log.emit(logging.WARNING, "voice_order_identity_mismatch")
            return
        if not state["worker_id"]:
            state["worker_id"] = worker_id
            self._save(state)
        for order in orders:
            if isinstance(order, dict):
                self._receive(order, worker_id, state, http)

    def _receive(self, order: dict[str, Any], worker_id: str, state: dict[str, Any], http: Any) -> None:
        order_id = order.get("order_id")
        if not isinstance(order_id, str) or not _ORDER_ID.match(order_id):
            log.emit(logging.WARNING, "voice_order_ignored", reason="identity")
            return
        token = order.get("hf_token") if order.get("action") == "install" else None
        key = order.get("idempotency_key")
        seen = state["keys"].get(key) if isinstance(key, str) else None
        content = {"model_id": order.get("model_id"), "repo": order.get("repo"), "revision": order.get("revision")}
        valid = (
            order.get("worker_id") == worker_id
            and self._well_formed(order)
            and (seen is None or (seen.get("order_id") == order_id and seen.get("content") == content))
        )
        if not valid:
            log.emit(logging.WARNING, "voice_order_refused", order_id=order_id)
            if order.get("status", "pending") == "pending":
                self._post_ack(http, order_id, False)
            return
        outcome = state["orders"].get(order_id)
        # Snitch redelivers the token with an unfinished gated order: one that arrives after a crash restores the
        # file, but a file already there is not rewritten at each pull.
        if isinstance(token, str) and token and (order.get("status", "pending") == "pending" or not os.path.exists(self._token_file)):
            self._keep_token(token)
        if order.get("status", "pending") == "pending":
            self._post_ack(http, order_id, True)
        if seen is None:
            state["keys"][key] = {"order_id": order_id, "content": content}
            self._save(state)
        if outcome is not None:
            self._repeat_outcome(http, order_id, content, outcome)
            return
        with self._lock:
            if order_id in self._running or any(o["order_id"] == order_id for o, _ in self._queue):
                return
            self._queue.append((order, http))
            if self._thread is None or not self._thread.is_alive():
                self._thread = threading.Thread(target=self._drain, daemon=True)
                self._thread.start()

    @staticmethod
    def _post_ack(http: Any, order_id: str, accepted: bool) -> None:
        body = {"accepted": accepted, "cause_code": "" if accepted else "invalid_order"}
        http.post(f"/api/worker/voice-model-orders/{order_id}/ack", json=body).raise_for_status()

    def _keep_token(self, token: str) -> None:
        """The token of a download order goes to the private file, unless the operator stored one locally."""
        if not voice_download.valid_token(token):
            log.emit(logging.WARNING, "voice_order_token_ignored")
            return
        if hf_token_sync.current_source(self._token_file) == "local":
            return
        if not hf_token_sync.write_meta(self._token_file, "central", "", acked=False) or not voice_download.persist_token(self._token_file, token):
            log.emit(logging.WARNING, "voice_order_token_not_stored")

    # -- execution ---------------------------------------------------------------------------------------------

    def wait(self) -> None:
        thread = self._thread
        if thread is not None:
            thread.join()

    def _drain(self) -> None:
        while True:
            with self._lock:
                if not self._queue:
                    return
                order, http = self._queue.pop(0)
                self._running.add(order["order_id"])
            try:
                self._run(order, http)
            except Exception:  # noqa: BLE001 - one bad order must not stop the Worker
                log.emit(logging.WARNING, "voice_order_stopped", order_id=order["order_id"])
            finally:
                with self._lock:
                    self._running.discard(order["order_id"])

    def _run(self, order: dict[str, Any], http: Any) -> None:
        order_id, model_id, revision = order["order_id"], order["model_id"], order["revision"]
        last: dict[str, Any] = {}

        def post(payload: dict[str, Any]) -> None:
            state = payload.get("state")
            body = {
                "order_id": order_id,
                "model_id": model_id,
                "revision": revision,
                "state": state,
                "cause_code": payload.get("cause_code", ""),
                "bytes_received": payload.get("bytes_received", 0),
                "bytes_total": payload.get("bytes_total", 0),
                "occupied_bytes": _directory_size(voice_models.model_path(self._env, order["repo"], revision)) if state in ("verifying", "ready") else payload.get("bytes_received", 0),
            }
            http.post("/api/worker/voice-models/status", json=body).raise_for_status()
            last.update(body)

        log.emit(logging.INFO, "voice_order_started", order_id=order_id, model_id=model_id, revision=revision, action=order["action"])
        if order["action"] == "remove":
            # Only an order from Snitch removes a model, after it decided the switch and that no job uses it.
            self._manager.remove([{"model_id": model_id, "repo": order["repo"], "revision": revision}], post)
        elif not self._fits(order, post):
            pass
        else:
            self._install(order, post)
        if last.get("state") in ("ready", "failed", "removed"):
            state = self._load()
            state["orders"][order_id] = {"state": last["state"], "cause_code": last["cause_code"], "occupied_bytes": last["occupied_bytes"]}
            self._save(state)

    def _fits(self, order: dict[str, Any], post: Any) -> bool:
        """Refuse before downloading what the disk cannot hold: the old model is kept, nothing is erased."""
        size = order.get("size_bytes")
        if not isinstance(size, int) or size <= 0:
            return True
        already = _directory_size(voice_models.model_path(self._env, order["repo"], order["revision"]))
        if self._free_bytes(voice_models.models_root(self._env)) >= (size - already) * DISK_MARGIN:
            return True
        log.emit(logging.INFO, "voice_order_disk_too_small", order_id=order["order_id"], model_id=order["model_id"])
        post({"state": "failed", "cause_code": "disk_full"})
        return False

    def _install(self, order: dict[str, Any], post: Any) -> None:
        model_id, revision = order["model_id"], order["revision"]
        self._manager.prepare({"model_id": model_id, "kind": order["kind"], "repo": order["repo"], "revision": revision, "role": "ordered", "retry": False}, post)

    @staticmethod
    def _repeat_outcome(http: Any, order_id: str, content: Mapping[str, Any], outcome: Mapping[str, Any]) -> None:
        """Snitch may have lost the report: tell the recorded outcome again, never recompute it."""
        body = {
            "order_id": order_id,
            "model_id": content["model_id"],
            "revision": content["revision"],
            "state": outcome["state"],
            "cause_code": outcome["cause_code"],
            "bytes_received": 0,
            "bytes_total": 0,
            "occupied_bytes": outcome.get("occupied_bytes", 0),
        }
        http.post("/api/worker/voice-models/status", json=body).raise_for_status()


def sync_orders(client: Any, executor: OrderExecutor) -> None:
    """Pull the orders addressed to this Worker and let the executor run them in the background."""
    http = client.get_httpx_client()
    response = http.get("/api/worker/voice-models")
    if response.status_code == 404:
        return  # A worker binary may briefly talk to a server predating this contract.
    response.raise_for_status()
    answer = response.json()
    if not isinstance(answer, dict):
        raise ValueError("invalid voice models from server")
    executor.handle(answer, http)


_executor: OrderExecutor | None = None


def default_executor() -> OrderExecutor:  # pragma: no cover - wires the real manager
    global _executor
    if _executor is None:
        # The token file path is read before the manager takes it out of the process environment.
        token_file = voice_download.default_token_file(os.environ)
        _executor = OrderExecutor(voice_models.default_manager(), os.environ, token_file)
    return _executor
