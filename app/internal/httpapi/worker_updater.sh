#!/usr/bin/env bash
# Snitch Worker local updater (ADR 0036). The launcher installs it next to worker.env and cron runs it.
# It asks Snitch for the target version with the Worker's durable token, pulls that image, restarts the
# container with the launcher's arguments, and rolls back to the previous image if the new one does not stay up.
# It never prints the token and never mounts the Docker socket into the Worker.
set -euo pipefail
umask 077

dir=$(cd "$(dirname "$0")" && pwd)
worker_id=$(basename "$dir")
env_file="$dir/worker.env"
container="snitch-worker-$worker_id"
volume="snitch-models-$worker_id"
lock="$dir/update.lock"
resp="$dir/update.json"
# Seconds the new container must run before it counts as up. Tests shorten it.
settle="${SNITCH_UPDATE_SETTLE:-45}"

[ -f "$env_file" ] || exit 0
api_url=$(sed -n 's/^SNITCH_API_URL=//p' "$env_file" | head -n1)
token=$(sed -n 's/^SNITCH_WORKER_TOKEN=//p' "$env_file" | head -n1)
[ -n "$api_url" ] && [ -n "$token" ] || exit 0
# One run at a time: cron may fire while a long pull is still running.
# A lock older than 30 minutes is left by a killed run (power loss, SIGKILL): take it over.
if [ -d "$lock" ] && [ -n "$(find "$lock" -maxdepth 0 -mmin +30 2>/dev/null)" ]; then rmdir "$lock" 2>/dev/null || true; fi
mkdir "$lock" 2>/dev/null || exit 0
trap 'rmdir "$lock" 2>/dev/null || true; rm -f "$resp"' EXIT

# report STATE VERSION DIGEST CAUSE: no free text, the server refuses anything else.
report() {
 local body
 body=$(printf '{"state":"%s","version":"%s","digest":"%s","cause_code":"%s"}' "$1" "$2" "$3" "$4")
 printf 'header = "Authorization: Bearer %s"\n' "$token" | curl -sS --max-time 15 -K - -H 'Content-Type: application/json' --data-binary "$body" -o /dev/null "$api_url/api/worker/update-status" >/dev/null 2>&1 || true
}

# The Worker must already have been installed by the launcher: no container, nothing to update.
current=$(docker inspect -f '{{.Config.Image}}' "$container" 2>/dev/null || true)
[ -n "$current" ] || exit 0

code=$(printf 'header = "Authorization: Bearer %s"\n' "$token" | curl -sS --max-time 20 -K - -o "$resp" -w '%{http_code}' "$api_url/api/worker/update" 2>/dev/null || true)
# 401 means revoked or unknown: the launcher owns that case, the updater stays silent.
[ "$code" = 200 ] || exit 0
target=$(sed -n 's/.*"target_image":"\([^"]*\)".*/\1/p' "$resp" | head -n1)
version=$(sed -n 's/.*"target_version":"\([^"]*\)".*/\1/p' "$resp" | head -n1)
[ -n "$target" ] && [ -n "$version" ] || exit 0

# Only a released version of the same repository as the installed image is accepted.
if ! [[ "$version" =~ ^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$ ]]; then exit 0; fi
if [ "$target" != "${current%:*}:$version" ]; then report failed "$version" "" target_mismatch; exit 0; fi
[ "$target" != "$current" ] || exit 0

previous_id=$(docker inspect -f '{{.Image}}' "$container" 2>/dev/null || true)
previous_version=${current##*:}
# Rollback only to an image whose digest is known (ADR 0036): remember it before the container is replaced.
previous_digest=$(docker image inspect -f '{{index .RepoDigests 0}}' "$previous_id" 2>/dev/null | sed 's/^[^@]*@//' || true)
if ! [[ "$previous_digest" =~ ^sha256:[a-f0-9]{64}$ ]]; then previous_id=''; fi

report pulling "$version" "" ""
if ! docker pull "$target" >/dev/null 2>&1; then report failed "$version" "" image_pull_failed; exit 0; fi
digest=$(docker image inspect -f '{{index .RepoDigests 0}}' "$target" 2>/dev/null | sed 's/^[^@]*@//' || true)
if ! [[ "$digest" =~ ^sha256:[a-f0-9]{64}$ ]]; then report failed "$version" "" digest_unknown; exit 0; fi

gpu=()
if command -v nvidia-smi >/dev/null 2>&1 && nvidia-smi >/dev/null 2>&1; then gpu=(--gpus all); fi
# Same arguments as the launcher's service start (deploy in worker_install.go): keep them in step.
start() { docker run -d --name "$container" --restart unless-stopped --user 10001:10001 ${gpu[@]+"${gpu[@]}"} --add-host host.docker.internal:host-gateway --env-file "$env_file" -v "$volume:/models" "$1" >/dev/null; }

report restarting "$version" "$digest" ""
docker rm -f "$container" >/dev/null 2>&1 || true
# Start by digest: the tag can be rewritten after the pull, the digest cannot.
if start "${target%:*}@$digest"; then
 sleep "$settle"
 # Up means running and never restarted: a crash loop under "unless-stopped" shows a restart count.
 if [ "$(docker inspect -f '{{.State.Running}} {{.RestartCount}}' "$container" 2>/dev/null || true)" = "true 0" ]; then
  report updated "$version" "$digest" ""
  exit 0
 fi
fi

# The new image did not stay up: restore the previous container image, which is still local.
docker rm -f "$container" >/dev/null 2>&1 || true
if [ -n "$previous_id" ] && start "$previous_id"; then
 report rolled_back "$previous_version" "" unhealthy
else
 report failed "$previous_version" "" rollback_failed
fi
exit 0
