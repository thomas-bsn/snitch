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
if [[ $# -eq 0 ]]; then set -- ./...; fi
(cd app && go test -race -tags integration -count=1 -timeout 60m -v "$@")
