#!/usr/bin/env python3
"""Local, single-instance supervisor for one named Snitch Worker container."""
import fcntl
import json
import os
import subprocess
import sys
import time
import urllib.error
import urllib.request
from pathlib import Path


API = os.environ["SNITCH_API_URL"].rstrip("/")
TOKEN = os.environ["SNITCH_SUPERVISOR_TOKEN"]
WORKER = os.environ["SNITCH_WORKER_ID"]
CONTAINER = f"snitch-worker-{WORKER}"
POLL_SECONDS = 5
HEALTH_SECONDS = 30
HEALTH_FILE = "/tmp/snitch-worker-health.json"
MAX_RESTARTS = 5
WINDOW_SECONDS = 3600
STATE = Path.home() / ".local" / "share" / "snitch-worker" / WORKER


def request(path: str, payload: dict | None = None) -> dict | None:
    data = None if payload is None else json.dumps(payload).encode()
    request = urllib.request.Request(
        API + path,
        data=data,
        headers={"Authorization": f"Bearer {TOKEN}", "Content-Type": "application/json"},
        method="GET" if payload is None else "PUT",
    )
    try:
        with urllib.request.urlopen(request, timeout=10) as response:
            return json.loads(response.read()) if response.status == 200 else None
    except (OSError, urllib.error.URLError, json.JSONDecodeError):
        return None


def docker(*args: str, timeout: int = 30) -> subprocess.CompletedProcess[str]:
    return subprocess.run(["docker", *args], capture_output=True, text=True, timeout=timeout, check=False)


def report(command: dict, status: str, result: str = "") -> None:
    request(f"/api/worker/supervisor/restarts/{command['id']}", {"status": status, "result": result})


def handle_restart(command: dict) -> None:
    report(command, "stopping")
    # The Worker stops claiming on SIGTERM and drains active leases before Docker stops it.
    # The bounded grace period is below the lease TTL; unfinished work can safely be reclaimed.
    stopped = docker("stop", "--time", "110", CONTAINER, timeout=120)
    if stopped.returncode != 0:
        report(command, "failed", "container_stop_failed")
        return
    report(command, "starting")
    started = docker("start", CONTAINER)
    if started.returncode != 0:
        report(command, "failed", "container_start_failed")
        return
    # Completion requires a fresh authenticated Worker pulse; the API, not this host, decides online.


def supervise() -> None:
    STATE.mkdir(mode=0o700, parents=True, exist_ok=True)
    lock = open(STATE / "supervisor.lock", "w", encoding="utf-8")
    try:
        fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
    except BlockingIOError:
        return
    restarts: list[float] = []
    consecutive = 0
    while True:
        command_response = request("/api/worker/supervisor/restarts")
        command = command_response.get("command") if command_response else None
        if command:
            handle_restart(command)
            time.sleep(POLL_SECONDS)
            continue
        # Do not auto-restart when Snitch cannot be reached; WSL2/Docker outages and partitions
        # must stay diagnosable and never cause a destructive local action.
        if command_response is not None:
            state = docker("inspect", "-f", "{{.State.Running}}", CONTAINER)
            stalled = False
            if state.returncode == 0 and state.stdout.strip() == "true":
                health = docker("exec", CONTAINER, "cat", HEALTH_FILE)
                if health.returncode == 0:
                    try:
                        loop_at = float(json.loads(health.stdout).get("loop_at", 0))
                        # A healthy retry loop may be unable to reach Snitch; do not
                        # restart it based on an old successful network pulse.
                        stalled = loop_at > 0 and time.time() - loop_at > HEALTH_SECONDS
                    except (ValueError, json.JSONDecodeError, TypeError):
                        stalled = False
            if state.returncode == 0 and (state.stdout.strip() == "false" or stalled):
                now = time.time()
                restarts = [stamp for stamp in restarts if now - stamp < WINDOW_SECONDS]
                if len(restarts) < MAX_RESTARTS:
                    consecutive += 1
                    time.sleep(min(30 * (2 ** min(consecutive - 1, 6)), 1800))
                    # Reachability may have changed during backoff. Require a fresh
                    # authenticated Snitch response immediately before local repair.
                    fresh_response = request("/api/worker/supervisor/restarts")
                    if fresh_response is None:
                        consecutive = 0
                        time.sleep(POLL_SECONDS)
                        continue
                    if fresh_response.get("command"):
                        handle_restart(fresh_response["command"])
                        time.sleep(POLL_SECONDS)
                        continue
                    if stalled:
                        docker("stop", "--time", "110", CONTAINER, timeout=120)
                    started = docker("start", CONTAINER)
                    if started.returncode == 0:
                        restarts.append(now)
                    else:
                        consecutive = 0
                else:
                    time.sleep(POLL_SECONDS)
            else:
                consecutive = 0
        time.sleep(POLL_SECONDS)


if __name__ == "__main__":
    try:
        supervise()
    except KeyboardInterrupt:
        sys.exit(0)
