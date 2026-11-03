"""Cancellation of the running job when its lease is refused (ADR 0029).

The job runs on the thread that claimed it. ``scope`` binds that thread to the event the lease
renewal thread sets on a refusal; the work then stops within a fraction of a second, by the means
that fits each kind of work:

* our own loops (media download, VAD chunks, embedding turns) call ``check`` between iterations;
* a child process (ffmpeg) is run by ``run_process``, which kills it;
* a remote call (the transcription server) is run by ``wait_for`` on a helper thread: the job stops
  waiting at once and the connection is closed, so nothing further is sent;
* an opaque in-process inference (pyannote, NeMo, torch) is run by ``interruptible``, which raises
  ``JobCancelled`` inside it: torch goes back to Python between layers, so the exception lands
  there and unwinds the inference, releasing its locks.

``JobCancelled`` derives from ``BaseException`` so that the providers' ``except Exception``
wrappers never turn a cancellation into a failure code.
"""

from __future__ import annotations

import contextlib
import ctypes
import subprocess
import threading
from typing import Any, Callable, Iterator, Sequence, TypeVar

T = TypeVar("T")

# How often a waiting job looks at its cancellation event.
POLL = 0.1

_local = threading.local()


class JobCancelled(BaseException):
    """The job's lease was refused: stop the work, record neither a result nor a failure."""


@contextlib.contextmanager
def scope(event: threading.Event) -> Iterator[None]:
    """Bind the current thread's work to ``event``: once set, the job is cancelled."""
    previous = getattr(_local, "event", None)
    _local.event = event
    try:
        yield
    finally:
        _local.event = previous


def _event() -> threading.Event | None:
    return getattr(_local, "event", None)


def check() -> None:
    """Raise ``JobCancelled`` when the current job was cancelled; a no-op outside a job."""
    event = _event()
    if event is not None and event.is_set():
        raise JobCancelled()


def run_process(command: Sequence[str], timeout: float, **kwargs: Any) -> None:
    """``subprocess.run(command, check=True, timeout=timeout)`` that kills the process on cancellation."""
    event = _event()
    with subprocess.Popen(command, **kwargs) as process:
        waited = 0.0
        while True:
            try:
                code = process.wait(POLL)
                break
            except subprocess.TimeoutExpired:
                waited += POLL
                if event is not None and event.is_set():
                    process.kill()
                    process.wait()
                    raise JobCancelled() from None
                if waited >= timeout:
                    process.kill()
                    process.wait()
                    raise subprocess.TimeoutExpired(command, timeout) from None
    if code != 0:
        raise subprocess.CalledProcessError(code, command)


def wait_for(call: Callable[[], T], on_cancel: Callable[[], None] | None = None) -> T:
    """Run a blocking remote call on a helper thread and stop waiting for it on cancellation.

    ``on_cancel`` closes what the call uses (its HTTP client); the helper thread is abandoned and
    ends with its connection, its outcome is never read."""
    event = _event()
    if event is None:
        return call()
    check()
    outcome: dict[str, Any] = {}
    done = threading.Event()

    def run() -> None:
        try:
            outcome["value"] = call()
        except BaseException as exc:  # noqa: BLE001 - handed back to the job thread
            outcome["error"] = exc
        finally:
            done.set()

    threading.Thread(target=run, daemon=True, name="snitch-remote-call").start()
    while not done.wait(POLL):
        if event.is_set():
            if on_cancel is not None:
                with contextlib.suppress(Exception):
                    on_cancel()
            raise JobCancelled()
    if "error" in outcome:
        raise outcome["error"]
    return outcome["value"]


def _raise_in(thread_id: int) -> None:
    ctypes.pythonapi.PyThreadState_SetAsyncExc(ctypes.c_ulong(thread_id), ctypes.py_object(JobCancelled))


def interruptible(call: Callable[..., T], *args: Any, **kwargs: Any) -> T:
    """Run an in-process inference that offers no hook, and interrupt it on cancellation.

    A watcher raises ``JobCancelled`` in the job thread, only while ``call`` is running: the
    exception surfaces at the next Python instruction of the inference and unwinds it."""
    event = _event()
    if event is None:
        return call(*args, **kwargs)
    check()
    target = threading.get_ident()
    guard = threading.Lock()
    running = [True]
    finished = threading.Event()

    def watch() -> None:
        while not finished.wait(POLL):
            if event.is_set():
                with guard:
                    if running[0]:
                        _raise_in(target)
                return

    watcher = threading.Thread(target=watch, daemon=True, name="snitch-cancel-watch")
    watcher.start()
    try:
        return call(*args, **kwargs)
    finally:
        with guard:
            running[0] = False
        finished.set()
