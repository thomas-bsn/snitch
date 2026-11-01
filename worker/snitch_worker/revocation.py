"""Self-cleanup of a Worker deleted in Snitch (ADR 0021, section on #89).

The container cannot reach Docker (no socket, UID 10001), so it cleans what it owns: the content
of its models volume. A 401 is never enough: only the narrow confirmation endpoint answering
``revoked: true`` triggers the erasure. A marker left in the volume keeps the Worker inert on every
later start (restart policy, host reboot) without any network call; the process stays alive and
idle instead of exiting, because ``--restart unless-stopped`` would restart an exited container in
a loop. Messages are fixed strings: the token and library errors are never logged.
"""

from __future__ import annotations

import os
import shutil
import sys
import threading
from typing import Any, Callable, Mapping

from snitch_client.api.default.get_worker_revocation import sync as fetch_revocation
from snitch_client.models.get_worker_revocation_response_200 import GetWorkerRevocationResponse200

MODELS_DIR_ENV = "SNITCH_WORKER_MODELS_DIR"
DEFAULT_MODELS_DIR = "/models"
MARKER_NAME = ".snitch-worker-revoked"
LOG_PREFIX = "[snitch-worker] "
INERT_MESSAGE = (
    "this Worker was deleted in Snitch: its models and caches were erased from the volume and it no longer "
    "contacts Snitch. The container stays idle; remove it, the empty volume and the launcher folder on the "
    "host. / Ce Worker a été supprimé dans Snitch : ses modèles et caches ont été effacés du volume et il ne "
    "contacte plus Snitch. Le conteneur reste inactif ; supprimez-le sur l’hôte, avec le volume vide et le "
    "dossier du lanceur."
)


def _log(message: str) -> None:
    print(LOG_PREFIX + message, file=sys.stderr, flush=True)


def models_dir(environ: Mapping[str, str]) -> str:
    return environ.get(MODELS_DIR_ENV, "").strip() or DEFAULT_MODELS_DIR


def is_revoked(directory: str) -> bool:
    return os.path.isfile(os.path.join(directory, MARKER_NAME))


def confirm_revocation(client: Any) -> bool:
    """True only when Snitch explicitly answers ``revoked: true``. Any other outcome erases nothing."""
    try:
        answer = fetch_revocation(client=client)
    except Exception:  # noqa: BLE001 - an outage or a malformed answer is not a confirmation
        return False
    return isinstance(answer, GetWorkerRevocationResponse200) and answer.revoked is True


def erase_local_data(directory: str, log: Callable[[str], None] = _log) -> None:
    """Write the marker first (so a crash mid-way still leaves the Worker inert), then remove the rest."""
    marker = os.path.join(directory, MARKER_NAME)
    try:
        with open(marker, "w", encoding="utf-8") as handle:
            handle.write("revoked\n")
    except OSError:
        log("could not write the revocation marker in the models volume")
    try:
        entries = os.listdir(directory)
    except OSError:
        log("could not list the models volume: nothing erased")
        return
    for name in entries:
        if name == MARKER_NAME:
            continue
        path = os.path.join(directory, name)
        try:
            if os.path.isdir(path) and not os.path.islink(path):
                shutil.rmtree(path)
            else:
                os.unlink(path)
        except OSError:
            log("could not erase part of the models volume")


def wait_inert(shutdown: threading.Event) -> None:
    """Stay alive and idle until Docker stops the container: no network, no restart loop."""
    _log(INERT_MESSAGE)
    while not shutdown.wait(1.0):
        pass


class RevocationGuard:
    """Shared by the worker threads: one confirmation, one erasure, then no more API calls."""

    def __init__(self, directory: str, log: Callable[[str], None] = _log) -> None:
        self.directory = directory
        self.revoked = threading.Event()
        self._lock = threading.Lock()
        self._log = log

    def on_unauthorized(self, client: Any) -> bool:
        with self._lock:
            if self.revoked.is_set():
                return True
            if not confirm_revocation(client):
                return False
            self.revoked.set()
            # Closing the HTTP client makes any call still in flight in another thread fail locally.
            try:
                client.get_httpx_client().close()
            except Exception:  # noqa: BLE001
                pass
            self._log("Snitch confirmed this Worker was deleted: erasing its models volume")
            erase_local_data(self.directory, self._log)
            return True
