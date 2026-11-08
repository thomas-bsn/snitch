#!/usr/bin/env bash
# Go tests with the race detector and the integration tag. Arguments are passed to `go test` (packages, -run...);
# without any, every package runs. CI shards the browser suites this way (SNITCH_E2E_SHARD, see verify.yml).
set -euo pipefail
cd "$(dirname "$0")/.."
if [[ -z "${DOCKER_HOST:-}" ]]; then
  export DOCKER_HOST="$(docker context inspect --format '{{.Endpoints.docker.Host}}')"
fi
# Docker socket inside Colima is /var/run/docker.sock; the host endpoint differs.
export TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE="${TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE:-/var/run/docker.sock}"
# A full local browser run uses the same partition as GitHub Actions. Explicit
# package/run arguments and CI shard invocations retain their existing behavior.
if [[ "${SNITCH_E2E:-}" == 1 && -z "${SNITCH_E2E_SHARD:-}" && $# -eq 0 ]]; then
  shards="${SNITCH_E2E_JOBS:-5}"
  if [[ ! "$shards" =~ ^[1-9][0-9]*$ ]]; then
    echo 'SNITCH_E2E_JOBS must be a positive integer' >&2
    exit 2
  fi
  pids=()
  stop_tree() {
    local child
    for child in $(pgrep -P "$1" || true); do stop_tree "$child"; done
    kill -TERM "$1" 2>/dev/null || true
  }
  stop_tests() {
    local pid
    for pid in "${pids[@]}"; do stop_tree "$pid"; done
  }
  trap 'stop_tests; exit 130' INT
  trap 'stop_tests; exit 143' TERM
  run_lot() {
    local label="$1"
    shift
    "$@" 2>&1 | sed -u "s|^|[$label] |"
  }
  run_lot go env SNITCH_E2E=0 bash scripts/test.sh &
  pids+=("$!")
  for ((shard=1; shard<=shards; shard++)); do
    run_lot "e2e $shard/$shards" env SNITCH_E2E_SHARD="$shard/$shards" \
      bash scripts/test.sh ./internal/httpapi -run 'TestFoundation/browser' &
    pids+=("$!")
  done
  status=0
  for pid in "${pids[@]}"; do
    if wait "$pid"; then :; else status=1; fi
  done
  exit "$status"
fi
if [[ $# -eq 0 ]]; then set -- ./...; fi
(cd app && go test -race -tags integration -count=1 -timeout 60m -v "$@")
