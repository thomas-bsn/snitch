"""Pause of this Worker by an administrator (ADR 0063).

The Worker is the only one to call: it reads a state on its next poll, it is never reached. The state is "paused or
not", so an instruction given while the Worker was offline is applied once, when it is back, and a paused Worker that
restarts stays paused.

- Pause: no new job is claimed. Jobs already running finish, and presence, orders and logs go on.
- Restarting or stopping the container is the managed restart (host supervisor, docs/transcription-worker.md), not
  something done from here: a process that is gone cannot ask for anything.
"""
from __future__ import annotations

import logging
import threading
from typing import Any

from snitch_client.api.default.ack_worker_control import sync as ack_control
from snitch_client.api.default.get_worker_control import sync as get_control
from snitch_client.client import AuthenticatedClient
from snitch_client.models.error import Error
from snitch_client.models.worker_control_ack import WorkerControlAck

from snitch_worker import log


class RemoteControl:
    def __init__(self) -> None:
        self.paused = threading.Event()
        self._applied: bool | None = None

    def sync_once(self, client: AuthenticatedClient) -> None:
        """Read the wanted state, apply it, and tell Snitch what is applied. Raises when Snitch cannot be read."""
        wanted = get_control(client=client)
        if wanted is None or isinstance(wanted, Error):
            raise RuntimeError("worker control not readable")
        if wanted.paused:
            self.paused.set()
        else:
            self.paused.clear()
        if self._applied == wanted.paused:
            return
        answer: Any = ack_control(client=client, body=WorkerControlAck(paused=wanted.paused))
        if isinstance(answer, Error):
            raise RuntimeError("worker control acknowledgement refused")
        # The first poll of a process tells Snitch what runs, so the interface never shows a state nobody confirmed.
        log.emit(logging.INFO, "worker_pause_applied" if wanted.paused else "worker_resume_applied")
        self._applied = wanted.paused


def control_loop(client: AuthenticatedClient, control: RemoteControl, stop: threading.Event, interval: float) -> None:
    """Poll the wanted state until the Worker stops. No recoverable error ends it (like the order loop).

    A Snitch out of reach is logged once when it starts and once when it ends, not at each poll."""
    failing = False
    while not stop.is_set():
        try:
            control.sync_once(client)
        except Exception as error:  # noqa: BLE001 - the control loop must outlive every recoverable error
            if not failing:
                log.emit(logging.WARNING, "worker_control_unreachable", error_type=type(error).__name__)
            failing = True
        else:
            if failing:
                log.emit(logging.INFO, "worker_control_reconnected")
            failing = False
        stop.wait(interval)
