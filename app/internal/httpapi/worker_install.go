package httpapi

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"text/template"
	"time"

	"github.com/jackc/pgx/v5"
	"snitch/internal/auth"
)

const workerInstallInviteLifetime = 15 * time.Minute

// createWorkerInstallCommand keeps the Worker token out of the copied command.
// A private local script enables later retries after this invitation expires.
func (s *Server) createWorkerInstallCommand(ctx context.Context, workerID, token, image, apiURL string) (string, error) {
	origin, err := url.Parse(apiURL)
	if err != nil || origin.Scheme != "https" || origin.Host == "" || origin.User != nil || origin.Path != "" || origin.RawQuery != "" || origin.Fragment != "" {
		return "", nil
	}
	code := auth.Token()
	sealed, err := auth.Seal(s.Config.MasterKey, "worker-install:"+workerID, token)
	if err != nil {
		return "", err
	}
	_, err = s.Pool.Exec(ctx, `INSERT INTO worker_install_invites(code_hash,worker_id,encrypted_token,api_url,image,expires_at) VALUES($1,$2,$3,$4,$5,$6)`, auth.Hash(code), workerID, sealed, apiURL, image, time.Now().Add(workerInstallInviteLifetime))
	if err != nil {
		return "", err
	}
	endpoint := strings.TrimRight(apiURL, "/") + "/api/worker/install/" + code
	dir := "$HOME/.local/share/snitch-worker/" + workerID
	return fmt.Sprintf(`bash -c 'set -euo pipefail; d="%s"; mkdir -p "$d"; chmod 700 "$d"; if [ ! -s "$d/install.sh" ]; then umask 077; curl -fsS --proto "=https" --tlsv1.2 %q -o "$d/install.sh.tmp"; mv "$d/install.sh.tmp" "$d/install.sh"; fi; bash "$d/install.sh"'`, dir, endpoint), nil
}

var workerInstallScript = template.Must(template.New("install").Parse(`#!/usr/bin/env bash
set -euo pipefail
umask 077
stage=prerequisites
fail() { printf 'Snitch Worker: %s failed. %s\n' "$stage" "$1" >&2; exit 1; }
command -v docker >/dev/null || fail 'Install Docker Engine, then rerun this command.'
command -v base64 >/dev/null || fail 'Install coreutils, then rerun this command.'
docker info >/dev/null 2>&1 || fail 'Start Docker and grant this user access, then rerun.'
api_url=$(printf '%s' '{{.APIURL}}' | base64 -d)
worker_token=$(printf '%s' '{{.Token}}' | base64 -d)
image=$(printf '%s' '{{.Image}}' | base64 -d)
worker_id='{{.WorkerID}}'
dir="$HOME/.local/share/snitch-worker/$worker_id"
mkdir -p "$dir"; chmod 700 "$dir"
env_file="$dir/worker.env"
printf 'SNITCH_API_URL=%s\nSNITCH_WORKER_TOKEN=%s\nTRANSCRIPTION_API_URL=\nSNITCH_VOICE_DOWNLOAD_ON_START=0\nSNITCH_VOICE_CONFIRM_TERMS=0\nHOME=/tmp\nXDG_CACHE_HOME=/models/.cache\nHF_HOME=/models/.cache/huggingface\nHF_XET_CACHE=/models/.cache/huggingface/xet\n' "$api_url" "$worker_token" > "$env_file"
chmod 600 "$env_file"; unset worker_token
stage=image
docker pull "$image" || fail 'Check network and registry credentials.'
volume="snitch-models-$worker_id"; container="snitch-worker-$worker_id"
docker volume create "$volume" >/dev/null || fail 'Check Docker volume permissions.'
stage=permissions
docker run --rm --user 0 --entrypoint sh -v "$volume:/models" "$image" -c 'mkdir -p /models/.cache/huggingface/xet; chown -R 10001:10001 /models' || fail 'Could not prepare model volume for UID 10001.'
docker run --rm --user 10001:10001 --entrypoint sh -v "$volume:/models" "$image" -c 'touch /models/.snitch-write && rm /models/.snitch-write' || fail 'Model volume is not writable by UID 10001.'
gpu=()
if command -v nvidia-smi >/dev/null && nvidia-smi >/dev/null 2>&1; then
 gpu=(--gpus all); stage=gpu
 docker run --rm "${gpu[@]}" --entrypoint sh "$image" -c true || fail 'Install NVIDIA Container Toolkit for Docker.'
else printf 'No NVIDIA GPU: voice processing unavailable.\n'; fi
if [ -r /dev/tty ]; then
 printf 'Download voice models? Accept pyannote Hugging Face terms first. [y/N] ' > /dev/tty
 IFS= read -r choice < /dev/tty || choice=''
else choice=''; fi
if [ "$choice" = y ] || [ "$choice" = Y ]; then
 [ -r /dev/tty ] || fail 'Use an interactive terminal for the Hugging Face token.'
 printf 'Hugging Face token (hidden): ' > /dev/tty
 stty -echo < /dev/tty
 IFS= read -r hf_token < /dev/tty || hf_token=''
 stty echo < /dev/tty; printf '\n' > /dev/tty
 [[ "$hf_token" =~ ^hf_[A-Za-z0-9]+$ ]] || fail 'Enter a valid Hugging Face token after accepting terms.'
 bootstrap_env=$(mktemp "$dir/bootstrap.XXXXXX")
 trap 'rm -f "$bootstrap_env"' EXIT
 cp "$env_file" "$bootstrap_env"
 printf 'HF_TOKEN=%s\nSNITCH_VOICE_DOWNLOAD_ON_START=1\nSNITCH_VOICE_CONFIRM_TERMS=1\n' "$hf_token" >> "$bootstrap_env"
 unset hf_token
 stage=voice-models
 docker run --rm "${gpu[@]}" --user 10001:10001 --env-file "$bootstrap_env" -v "$volume:/models" --entrypoint python "$image" -c 'import os; from snitch_worker.bootstrap import run_bootstrap; raise SystemExit(run_bootstrap(os.environ) == "failed")' || fail 'Check Hugging Face access and model terms, then rerun.'
 rm -f "$bootstrap_env"; trap - EXIT
fi
stage=service
if docker container inspect "$container" >/dev/null 2>&1; then docker rm -f "$container" >/dev/null || fail 'Could not replace existing container.'; fi
docker run -d --pull always --name "$container" --restart unless-stopped --user 10001:10001 "${gpu[@]}" --add-host host.docker.internal:host-gateway --env-file "$env_file" -v "$volume:/models" "$image" >/dev/null || fail 'Could not start Worker; inspect Docker logs.'
printf 'Snitch Worker started. Check Admin > Workers for connection and models.\n'
`))

func (s *Server) serveWorkerInstall(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	if len(code) != 43 || strings.ContainsAny(code, "/?\x00") {
		http.NotFound(w, r)
		return
	}
	var workerID, apiURL, image string
	var sealed []byte
	err := s.Pool.QueryRow(r.Context(), `SELECT i.worker_id,i.encrypted_token,i.api_url,i.image FROM worker_install_invites i JOIN workers w ON w.id=i.worker_id WHERE i.code_hash=$1 AND i.expires_at>now() AND w.revoked_at IS NULL`, auth.Hash(code)).Scan(&workerID, &sealed, &apiURL, &image)
	if errors.Is(err, pgx.ErrNoRows) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		s.internal(w, err)
		return
	}
	token, err := auth.Open(s.Config.MasterKey, "worker-install:"+workerID, sealed)
	if err != nil {
		s.internal(w, err)
		return
	}
	w.Header().Set("Content-Type", "text/x-shellscript; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	data := struct{ WorkerID, APIURL, Token, Image string }{workerID, base64.StdEncoding.EncodeToString([]byte(apiURL)), base64.StdEncoding.EncodeToString([]byte(token)), base64.StdEncoding.EncodeToString([]byte(image))}
	if err = workerInstallScript.Execute(w, data); err != nil {
		s.Log.Error("worker install script render failed")
	}
}
