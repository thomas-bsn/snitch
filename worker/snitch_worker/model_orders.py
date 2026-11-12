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
import time
from collections.abc import Mapping
from typing import Any

from snitch_worker import engine_runtime, hf_token_sync, log, voice_download, voice_manifest, voice_models

_ORDER_ID = re.compile(r"^ord_[A-Za-z0-9_-]{16,64}$")
_WORKER_ID = re.compile(r"^wk_[A-Za-z0-9_-]{20,60}$")
_KEY = re.compile(r"^[A-Za-z0-9._:-]{8,100}$")
_MODEL_ID = re.compile(r"^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$")
# Engines an approved adapter exists for, by type. Separation only runs on an on-demand family (see _runs_here).
ENGINES = {"diarization": {"pyannote", "nemo"}, "fingerprint": {"pyannote"} | engine_runtime.SIGNED_ONLY_ADAPTERS, "vad": {"silero-jit"}}
_DIGEST = re.compile(r"^[0-9a-f]{64}$")
# Causes an engine order may report (the same stable set Snitch accepts); anything else folds onto the generic one.
# Intents of a model order: "install" (weights and runtime), or a runtime-only "update" / "rollback" (#352).
RUNTIME_INTENTS = {"install", "update", "rollback"}
ENGINE_CAUSES = {"disk_full", "runtime_incompatible", "hf_network", "cancelled", "runtime_in_use", "platform_incompatible", "cuda_incompatible", "insufficient_ram", "insufficient_vram",
                "package_not_found", "resolution_failed", "build_failed"}
# The inventory is observed on every pull but only sent when it changed or this long after the last send.
INVENTORY_REFRESH_SECONDS = 60.0
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

    @staticmethod
    def _install_runtime(runtime: Any, environ: Mapping[str, str], report: Any, detailed: bool = False) -> Any:
        from snitch_worker import engine_runtime
        return engine_runtime.install(runtime, environ, report, detailed)

    def __init__(self, manager: voice_models.ModelManager, environ: Mapping[str, str], token_file: str) -> None:
        self._manager = manager
        self._env = environ
        self._token_file = token_file
        self._path = state_path(environ)
        self._lock = threading.Lock()
        self._state_lock = threading.RLock()
        self._queue: list[tuple[dict[str, Any], Any]] = []
        self._running: set[str] = set()
        # Engine orders Snitch asked to cancel (#351); the install thread checks this at each safe point.
        self._cancelled: set[str] = set()
        self._thread: threading.Thread | None = None
        self._inventory_sent: tuple[list[dict[str, str]], float] | None = None
        self._hardware_sent: tuple[dict[str, Any], float] | None = None

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
    def _valid_runtime(order: Mapping[str, Any]) -> bool:
        from snitch_worker import engine_runtime
        runtime = order.get("runtime")
        if runtime is None:
            return order.get("engine") not in engine_runtime.SPEAKER_ENGINES
        try:
            if engine_runtime.is_on_demand(runtime):
                # The engine of a model installs by family and version; its kind must be the model's (ADR 0056).
                engine_runtime.validate_on_demand(runtime)
                return runtime["kind"] == order.get("kind")
            engine_runtime.validate(runtime, order.get("engine", ""), order.get("kind", ""))
            return True
        except engine_runtime.RuntimeFailed:
            return False

    @staticmethod
    def _runs_here(order: Mapping[str, Any], kind: Any, engine: str) -> bool:
        """An adapter of this Worker runs the model: the signed and embedded ones by name, a family model by the
        family and kind of its on-demand runtime, whatever adapter name the catalogue gives it (ADR 0056)."""
        from snitch_worker import engine_runtime
        runtime = order.get("runtime")
        if engine_runtime.is_on_demand(runtime):
            return engine_runtime.runs_kind(runtime) and runtime.get("kind") == kind
        return kind in ENGINES and engine in ENGINES[kind]

    @staticmethod
    def _valid_intent(order: Mapping[str, Any]) -> bool:
        """An update or a rollback only changes the runtime of a model already here: it needs a runtime to bind."""
        intent = order.get("runtime_intent") or "install"
        if intent == "install":
            return True
        return intent in RUNTIME_INTENTS and order.get("action") == "install" and isinstance(order.get("runtime"), dict)

    @staticmethod
    def _well_formed(order: Mapping[str, Any]) -> bool:
        kind, engine = order.get("kind"), order.get("engine")
        repo, revision = order.get("repo"), order.get("revision")
        return (
            order.get("action") in ("install", "remove")
            and (order.get("action") == "remove" or OrderExecutor._valid_runtime(order))
            and OrderExecutor._valid_intent(order)
            and isinstance(kind, str)
            and isinstance(engine, str)
            and OrderExecutor._runs_here(order, kind, engine)
            and isinstance(order.get("model_id"), str)
            and bool(_MODEL_ID.fullmatch(order["model_id"]))
            and isinstance(repo, str)
            and isinstance(revision, str)
            and voice_models.valid_model(repo, revision)
            and isinstance(order.get("idempotency_key"), str)
            and bool(_KEY.fullmatch(order["idempotency_key"]))
        )

    def handle(self, answer: Mapping[str, Any], http: Any) -> None:
        """Check, acknowledge and queue the orders of one pull; execution continues in the background."""
        orders = answer.get("orders")
        engine_orders = answer.get("engine_orders")
        cleanups = answer.get("engine_cleanups")
        worker_id = answer.get("worker_id")
        if not isinstance(orders, list) or not isinstance(worker_id, str) or not _WORKER_ID.fullmatch(worker_id):
            return
        if not isinstance(engine_orders, list):
            engine_orders = []
        if not isinstance(cleanups, list):
            cleanups = []
        if not orders and not engine_orders and not cleanups:
            return
        with self._state_lock:
            self._handle_orders(orders, worker_id, http, engine_orders, cleanups)

    def _handle_orders(self, orders: list[Any], worker_id: str, http: Any, engine_orders: list[Any] | None = None, cleanups: list[Any] | None = None) -> None:
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
        for order in engine_orders or []:
            if isinstance(order, dict):
                self._receive_engine(order, worker_id, state, http)
        for cleanup in cleanups or []:
            if isinstance(cleanup, dict):
                self._run_cleanup(cleanup, http)

    # -- engine orders (#349): the only thing that installs a runtime ----------------------------------------

    @staticmethod
    def _well_formed_engine(order: Mapping[str, Any]) -> bool:
        from snitch_worker import engine_runtime
        runtime, engine = order.get("runtime"), order.get("engine")
        generation = order.get("generation")
        if not (
            order.get("intent") in ("install", "uninstall")
            and isinstance(engine, str)
            and isinstance(order.get("digest"), str)
            and bool(_DIGEST.fullmatch(order["digest"]))
            and isinstance(order.get("version"), str)
            and type(generation) is int
            and generation > 0
            and isinstance(order.get("idempotency_key"), str)
            and bool(_KEY.fullmatch(order["idempotency_key"]))
            and isinstance(runtime, dict)
        ):
            return False
        try:
            # A signed manifest or an on-demand description: each is validated against its own closed schema.
            engine_runtime.validate_runtime(runtime)
        except engine_runtime.RuntimeFailed:
            return False
        return runtime.get("engine") == engine and runtime.get("version") == order["version"]

    def _receive_engine(self, order: dict[str, Any], worker_id: str, state: dict[str, Any], http: Any) -> None:
        order_id = order.get("order_id")
        if not isinstance(order_id, str) or not _ORDER_ID.fullmatch(order_id):
            log.emit(logging.WARNING, "engine_order_ignored", reason="identity")
            return
        pending = order.get("status", "pending") == "pending"
        key = order.get("idempotency_key")
        content = {name: order.get(name) for name in ("digest", "engine", "version", "intent", "generation", "runtime")}
        # Engine keys live in their own space in the persisted state: a model key can never collide with them.
        state_key = "engine:" + key if isinstance(key, str) else None
        seen = state["keys"].get(state_key) if state_key else None
        valid = (
            order.get("worker_id") == worker_id
            and self._well_formed_engine(order)
            and (seen is None or (seen.get("order_id") == order_id and seen.get("content") == content))
        )
        if not valid:
            log.emit(logging.WARNING, "engine_order_refused", order_id=order_id)
            if pending:
                self._post_ack(http, order_id, False, "voice-engine-orders")
            return
        if pending:
            self._post_ack(http, order_id, True, "voice-engine-orders")
        if seen is None:
            state["keys"][state_key] = {"order_id": order_id, "content": content}
            self._save(state)
        outcome = state["orders"].get(order_id)
        if outcome is not None:
            self._post_engine(http, order_id, content["digest"], outcome["state"], outcome["cause_code"], outcome.get("local_identity", ""))
            return
        if order.get("cancel_requested") is True:
            self._cancel_engine(order_id, content["digest"], http)
            return
        self._enqueue(dict(order, target="engine"), http)

    def _cancel_engine(self, order_id: str, digest: str, http: Any) -> None:
        """Stop an engine order Snitch asked to cancel. A running install stops at its next safe point; one that
        has not started is dropped. Either way the outcome is recorded before it is told."""
        with self._lock:
            if order_id in self._running:
                self._cancelled.add(order_id)
                return
            self._queue = [(o, h) for o, h in self._queue if o["order_id"] != order_id]
        with self._state_lock:
            if order_id in self._load()["orders"]:
                return  # it ended on its own first: that outcome stands
        self._record_engine_outcome(http, order_id, digest, "failed", "cancelled", "")

    def _record_engine_outcome(self, http: Any, order_id: str, digest: str, final: str, cause: str, identity: str) -> None:
        with self._state_lock:
            state = self._load()
            state["orders"][order_id] = {"state": final, "cause_code": cause, "local_identity": identity}
            self._save(state)
        self._post_engine(http, order_id, digest, final, cause, identity)

    def _run_cleanup(self, cleanup: dict[str, Any], http: Any) -> None:
        """Remove the incomplete files of a failed or cancelled install, on Snitch's explicit request only. Skipped
        while any order is queued or running; a validated runtime is never removed."""
        from snitch_worker import engine_runtime
        order_id, digest, runtime = cleanup.get("order_id"), cleanup.get("digest"), cleanup.get("runtime")
        if not (isinstance(order_id, str) and _ORDER_ID.fullmatch(order_id) and isinstance(digest, str) and _DIGEST.fullmatch(digest) and isinstance(runtime, dict)):
            log.emit(logging.WARNING, "engine_cleanup_ignored", reason="malformed")
            return
        try:
            engine_runtime.validate_runtime(runtime)
        except engine_runtime.RuntimeFailed:
            log.emit(logging.WARNING, "engine_cleanup_ignored", reason="runtime")
            return
        with self._lock:
            if self._running or self._queue:
                return  # asked again at the next pull, once no order runs: never under an install in progress
            try:
                removed = engine_runtime.cleanup(runtime, self._env)
            except (engine_runtime.RuntimeFailed, OSError):
                log.emit(logging.WARNING, "engine_cleanup_failed", order_id=order_id)
                return
        log.emit(logging.INFO, "engine_cleanup_done", order_id=order_id, removed=removed)
        http.post("/api/worker/voice-engines/cleanup", json={"order_id": order_id, "digest": digest, "removed": removed}).raise_for_status()

    def _enqueue(self, order: dict[str, Any], http: Any) -> None:
        with self._lock:
            if order["order_id"] in self._running or any(o["order_id"] == order["order_id"] for o, _ in self._queue):
                return
            self._queue.append((order, http))
            if self._thread is None or not self._thread.is_alive():
                self._thread = threading.Thread(target=self._drain, daemon=True)
                self._thread.start()

    @staticmethod
    def _post_engine(http: Any, order_id: str, digest: str, state: str, cause: str = "", identity: str = "", received: int = 0, total: int = 0) -> None:
        body = {"order_id": order_id, "digest": digest, "state": state, "cause_code": cause, "bytes_received": received, "bytes_total": total}
        if identity:
            body["local_identity"] = identity
        http.post("/api/worker/voice-engines/status", json=body).raise_for_status()

    def _run_engine(self, order: dict[str, Any], http: Any) -> None:
        from snitch_worker import engine_runtime
        order_id, digest, runtime = order["order_id"], order["digest"], order["runtime"]
        steps = {"downloading": "downloading_runtime", "installing": "installing", "verifying": "verifying"}

        def report(stage: str, *_ignored: Any) -> None:
            if order_id in self._cancelled:
                raise engine_runtime.Cancelled()
            if stage in steps:
                self._post_engine(http, order_id, digest, steps[stage])

        log.emit(logging.INFO, "engine_order_started", order_id=order_id, engine=order["engine"], version=order["version"], intent=order["intent"])
        identity, cause, final = "", "", "ready"
        try:
            if order["intent"] == "uninstall":
                # Only the explicitly designated release goes, and never one a model still points to. A removal
                # is never cancelled: it does not watch the cancellation flag.
                final = "removed"
                engine_runtime.uninstall(runtime, self._env)
            else:
                with engine_runtime.stop_when(lambda: order_id in self._cancelled):
                    self._install_runtime(runtime, self._env, report, True)
                identity = engine_runtime.local_identity(runtime, self._env)
        except engine_runtime.RuntimeFailed as error:
            final, cause = "failed", error.cause if error.cause in ENGINE_CAUSES else "runtime_incompatible"
            log.emit(logging.WARNING, "engine_order_failed", order_id=order_id, cause_code=cause)
        except Exception as error:  # noqa: BLE001 - an order that cannot run must end as failed, not stay open
            final, cause = "failed", "runtime_incompatible"
            log.emit(logging.WARNING, "engine_order_failed", order_id=order_id, cause_code=cause, error_class=type(error).__name__)
        # The outcome is recorded before it is told: a lost report is repeated, never recomputed.
        self._cancelled.discard(order_id)
        self._record_engine_outcome(http, order_id, digest, final, cause, identity)

    def report_hardware(self, http: Any) -> None:
        """Tell Snitch what this machine offers, so a release is checked before it is ordered. Observation only."""
        from snitch_worker import engine_runtime
        hw = engine_runtime.hardware(self._env)
        if hw is None:
            return  # the GPU cannot be assessed here: Snitch does not constrain this Worker
        now = time.monotonic()
        # Free disk changes constantly: compare what is stable, and refresh the rest on the inventory cadence.
        stable = {k: v for k, v in hw.items() if k != "disk_free_bytes"}
        if self._hardware_sent is not None and self._hardware_sent[0] == stable and now - self._hardware_sent[1] < INVENTORY_REFRESH_SECONDS:
            return
        response = http.post("/api/worker/voice-engines/hardware", json=hw)
        if response.status_code == 404:
            return  # a server predating resource checks
        response.raise_for_status()
        self._hardware_sent = (stable, now)

    def report_inventory(self, http: Any) -> None:
        """Tell Snitch which runtimes this volume proves it holds. Observation only: no order, no download."""
        from snitch_worker import engine_runtime
        runtimes = engine_runtime.inventory(self._env)
        now = time.monotonic()
        if self._inventory_sent is not None and self._inventory_sent[0] == runtimes and now - self._inventory_sent[1] < INVENTORY_REFRESH_SECONDS:
            return
        response = http.post("/api/worker/voice-engines/inventory", json={"runtimes": runtimes})
        if response.status_code == 404:
            return  # a server predating engine orders
        response.raise_for_status()
        self._inventory_sent = (runtimes, now)

    def _receive(self, order: dict[str, Any], worker_id: str, state: dict[str, Any], http: Any) -> None:
        order_id = order.get("order_id")
        if not isinstance(order_id, str) or not _ORDER_ID.fullmatch(order_id):
            log.emit(logging.WARNING, "voice_order_ignored", reason="identity")
            return
        token = order.get("hf_token") if order.get("action") == "install" else None
        key = order.get("idempotency_key")
        seen = state["keys"].get(key) if isinstance(key, str) else None
        content = {name: order.get(name) for name in ("model_id", "repo", "revision", "action", "kind", "engine", "runtime")}
        content["runtime_intent"] = order.get("runtime_intent") or "install"
        # Older volumes recorded only the pinned identity; preserve their restart semantics.
        recorded = seen.get("content", {}) if isinstance(seen, dict) else {}
        same_content = all(content.get(name) == value for name, value in recorded.items()) and ("runtime" in recorded or content["runtime"] is None)
        valid = (
            order.get("worker_id") == worker_id
            and self._well_formed(order)
            and (seen is None or (seen.get("order_id") == order_id and same_content))
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
        self._enqueue(order, http)

    @staticmethod
    def _post_ack(http: Any, order_id: str, accepted: bool, family: str = "voice-model-orders") -> None:
        body = {"accepted": accepted, "cause_code": "" if accepted else "invalid_order"}
        http.post(f"/api/worker/{family}/{order_id}/ack", json=body).raise_for_status()

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
                    self._thread = None
                    return
                order, http = self._queue.pop(0)
                self._running.add(order["order_id"])
            try:
                if order.get("target") == "engine":
                    self._run_engine(order, http)
                else:
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
                "occupied_bytes": _directory_size(voice_models.ready_path(self._env, order["repo"], revision, order["kind"]) or voice_models.model_path(self._env, order["repo"], revision)) if state in ("verifying", "ready") else payload.get("bytes_received", 0),
            }
            http.post("/api/worker/voice-models/status", json=body).raise_for_status()
            last.update(body)

        log.emit(logging.INFO, "voice_order_started", order_id=order_id, model_id=model_id, revision=revision, action=order["action"])
        if order["action"] == "remove":
            # Only an order from Snitch removes a model, after it decided the switch and that no job uses it.
            self._manager.remove([{"model_id": model_id, "repo": order["repo"], "revision": revision, "kind": order["kind"]}], post)
        elif not self._fits(order, post):
            pass
        else:
            self._install(order, post)
        if last.get("state") in ("ready", "failed", "removed"):
            with self._state_lock:
                state = self._load()
                state["orders"][order_id] = {"state": last["state"], "cause_code": last["cause_code"], "occupied_bytes": last["occupied_bytes"]}
                self._save(state)

    def _fits(self, order: dict[str, Any], post: Any) -> bool:
        """Refuse before downloading what the disk cannot hold: the old model is kept, nothing is erased."""
        if voice_models.is_ready(self._env, order["repo"], order["revision"]):
            return True
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
        self._manager.prepare({"model_id": model_id, "kind": order["kind"], "repo": order["repo"], "revision": revision, "role": "ordered", "retry": False, "engine": order["engine"], "runtime": order.get("runtime"), "runtime_intent": order.get("runtime_intent") or "install"}, post)

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
    try:
        executor.report_inventory(http)
    except Exception as error:  # noqa: BLE001 - reconciliation is best effort and never blocks the orders
        log.emit(logging.WARNING, "engine_inventory_failed", error_type=type(error).__name__)
    try:
        executor.report_hardware(http)
    except Exception as error:  # noqa: BLE001 - an observation never blocks the orders either
        log.emit(logging.WARNING, "engine_hardware_failed", error_type=type(error).__name__)


_executor: OrderExecutor | None = None


def default_executor() -> OrderExecutor:  # pragma: no cover - wires the real manager
    global _executor
    if _executor is None:
        # The token file path is read before the manager takes it out of the process environment.
        token_file = voice_download.default_token_file(os.environ)
        _executor = OrderExecutor(voice_models.default_manager(), os.environ, token_file)
    return _executor
