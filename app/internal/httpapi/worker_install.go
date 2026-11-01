package httpapi

import (
	"bytes"
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
// The invitation is single-use; the private local launcher it downloads resumes later installs.
// It returns an empty command and a reason when Snitch refuses to generate one.
func (s *Server) createWorkerInstallCommand(ctx context.Context, workerID, token, image, apiURL string, allowInsecure bool, transcriptionMode, transcriptionEndpoint, transcriptionModel, transcriptionLanguage string) (command, unavailable string, err error) {
	origin, err := url.Parse(apiURL)
	if err != nil || (origin.Scheme != "http" && origin.Scheme != "https") || origin.Host == "" || origin.User != nil || origin.Path != "" || origin.RawQuery != "" || origin.Fragment != "" {
		return "", "invalid_origin", nil
	}
	if origin.Scheme == "http" && !allowInsecure {
		return "", "insecure_origin", nil
	}
	code := auth.Token()
	sealed, err := auth.Seal(s.Config.MasterKey, "worker-install:"+workerID, token)
	if err != nil {
		return "", "", err
	}
	_, err = s.Pool.Exec(ctx, `INSERT INTO worker_install_invites(code_hash,worker_id,encrypted_token,api_url,image,expires_at,transcription_mode,transcription_endpoint,transcription_model,transcription_language) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`, auth.Hash(code), workerID, sealed, apiURL, image, time.Now().Add(workerInstallInviteLifetime), transcriptionMode, transcriptionEndpoint, transcriptionModel, transcriptionLanguage)
	if err != nil {
		return "", "", err
	}
	endpoint := strings.TrimRight(apiURL, "/") + "/api/worker/install/" + code
	dir := "$HOME/.local/share/snitch-worker/" + workerID
	proto := "=https"
	if origin.Scheme == "http" {
		proto = "=http,https"
	}
	// The launcher is kept only once downloaded: rerunning the command reuses it, a failed download needs a new invitation.
	return fmt.Sprintf(`bash -c 'set -euo pipefail; d="%s"; mkdir -p "$d"; chmod 700 "$d"; if [ ! -s "$d/install.sh" ]; then umask 077; curl -fsS --proto "%s" --tlsv1.2 %q -o "$d/install.sh.tmp"; mv "$d/install.sh.tmp" "$d/install.sh"; fi; bash "$d/install.sh"'`, dir, proto, endpoint), "", nil
}

var workerInstallScript = template.Must(template.New("install").Parse(`#!/usr/bin/env bash
# Snitch Worker launcher. Safe to rerun: identity, service, volume and models are reused.
set -euo pipefail
umask 077
stage=prerequisites
worker_id='{{.WorkerID}}'
dir="$HOME/.local/share/snitch-worker/$worker_id"
mkdir -p "$dir"; chmod 700 "$dir"
env_file="$dir/worker.env"
# Installation events: phase, status, stable cause code and the original timestamp only, never free text or a secret.
# They are queued in events.queue and sent with the durable token; offline, they wait for the next authenticated run.
queue="$dir/events.queue"
events_token=''; events_api=''
flush_events() {
 [ -n "$events_token" ] && [ -s "$queue" ] || return 0
 local body code ts phase status cause sep=''
 body=$(mktemp "$dir/events.XXXXXX")
 { printf '{"events":['
   while read -r ts phase status cause; do
     printf '%s{"phase":"%s","status":"%s","occurred_at":"%s"' "$sep" "$phase" "$status" "$ts"
     [ -z "$cause" ] || printf ',"cause_code":"%s"' "$cause"
     printf '}'; sep=','
   done < "$queue"
   printf ']}'; } > "$body"
 code=$(printf 'header = "Authorization: Bearer %s"\n' "$events_token" | curl -sS --max-time 10 -K - -H 'Content-Type: application/json' --data-binary @"$body" -o /dev/null -w '%{http_code}' "$events_api/api/worker/install-events" 2>/dev/null || true)
 rm -f "$body"
 case "$code" in 2??|400|401) rm -f "$queue" ;; esac
 return 0
}
# emit PHASE STATUS [CAUSE]
emit() {
 printf '%s %s %s %s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$1" "$2" "${3:-}" >> "$queue"
 if [ "$(wc -l < "$queue")" -gt 50 ]; then tail -n 50 "$queue" > "$queue.tmp"; mv "$queue.tmp" "$queue"; fi
 flush_events
}
# fail CODE "English recovery" "Récupération en français": names the step and a stable cause, never a secret.
# A preflight, when set, names the diagnostic that failed (WSL2 checks); it is shown in the terminal only.
preflight=''
fail() { emit "$stage" failed "$1"; printf 'Snitch Worker: step "%s" failed [%s].\n' "$stage" "$1" >&2; [ -z "$preflight" ] || printf '  Preflight / Diagnostic: preflight %s\n' "$preflight" >&2; printf '  %s\n  %s\n' "$2" "$3" >&2; exit 1; }
# WSL2 (P1): a Linux kernel built by Microsoft. WSL1 has no real kernel and cannot run Docker Engine.
wsl=''
kernel_release=$(uname -r 2>/dev/null || true)
proc_version=$(cat "${SNITCH_PROC_VERSION:-/proc/version}" 2>/dev/null || true)
case "$kernel_release $proc_version" in
 *[Ww][Ss][Ll]2*) wsl=2 ;;
 *[Mm]icrosoft*) wsl=1 ;;
esac
[ "$(uname -s)" = Linux ] || fail unsupported_platform 'Only Linux hosts are supported; run this command on a Linux machine.' 'Seuls les hôtes Linux sont pris en charge ; lancez cette commande sur une machine Linux.'
if [ "$wsl" = 1 ]; then fail unsupported_platform 'WSL1 cannot run Docker. Convert this distribution with "wsl --set-version <distribution> 2" in Windows, then rerun.' 'WSL1 ne peut pas exécuter Docker. Convertissez cette distribution avec « wsl --set-version <distribution> 2 » dans Windows, puis relancez.'; fi
if [ "$wsl" = 2 ]; then
 printf 'WSL2 detected: running the WSL2 diagnostics.\n'
 export PATH="$PATH:/usr/lib/wsl/lib"
 # State and models must live on the Linux filesystem: Windows mounts break permissions (UID 10001, mode 600) and are slow.
 preflight=wsl-storage
 case "$dir/" in
  "${SNITCH_WSL_MOUNT_PREFIX:-/mnt/}"[a-z]/*) fail volume_not_writable "The Worker state is on a Windows drive ($dir). Move to your Linux home with \"cd ~\", delete that folder, then rerun the command from the Linux home (not /mnt/c)." "L’état du Worker est sur un disque Windows ($dir). Revenez dans votre dossier personnel Linux avec « cd ~ », supprimez ce dossier, puis relancez la commande depuis ce dossier (pas /mnt/c). Linux home = dossier personnel Linux." ;;
 esac
 preflight=''
fi
command -v docker >/dev/null || fail docker_unavailable "$([ "$wsl" = 2 ] && printf 'Install Docker Engine in this WSL2 distribution, or enable Docker Desktop WSL integration for it, then rerun.' || printf 'Install Docker Engine, then rerun this command.')" "$([ "$wsl" = 2 ] && printf 'Installez Docker Engine dans cette distribution WSL2, ou activez l’intégration WSL de Docker Desktop pour elle, puis relancez.' || printf 'Installez Docker Engine, puis relancez cette commande.')"
command -v curl >/dev/null || fail unknown 'Install curl, then rerun this command.' 'Installez curl, puis relancez cette commande.'
command -v base64 >/dev/null || fail unknown 'Install coreutils, then rerun this command.' 'Installez coreutils, puis relancez cette commande.'
if [ "$wsl" = 2 ]; then preflight=wsl-docker; docker info >/dev/null 2>&1 || fail docker_unavailable 'Docker is not reachable from WSL2. Start Docker Desktop and enable WSL integration for this distribution (or start dockerd here), then rerun.' 'Docker est injoignable depuis WSL2. Démarrez Docker Desktop et activez l’intégration WSL pour cette distribution (ou démarrez dockerd ici), puis relancez.'; preflight=''; fi
docker info >/dev/null 2>&1 || fail docker_unavailable 'Start Docker and grant this user access, then rerun.' 'Démarrez Docker et donnez accès à cet utilisateur, puis relancez.'
api_url=$(printf '%s' '{{.APIURL}}' | base64 -d)
seed_token=$(printf '%s' '{{.Token}}' | base64 -d)
image=$(printf '%s' '{{.Image}}' | base64 -d)
transcription_endpoint=$(printf '%s' '{{.TranscriptionEndpoint}}' | base64 -d)
transcription_model=$(printf '%s' '{{.TranscriptionModel}}' | base64 -d)
transcription_language=$(printf '%s' '{{.TranscriptionLanguage}}' | base64 -d)
# The durable token lives in worker.env (mode 600). A lost token is unrecoverable: ask the administrator for a new invitation.
worker_token=$(sed -n 's/^SNITCH_WORKER_TOKEN=//p' "$env_file" 2>/dev/null | head -n1 || true)
[ -n "$worker_token" ] || worker_token=$seed_token
unset seed_token
events_token=$worker_token; events_api=$api_url
emit prerequisites succeeded
stage=registration
status=$(printf 'header = "Authorization: Bearer %s"\n' "$worker_token" | curl -sS --max-time 20 -K - -o "$dir/config.json" -w '%{http_code}' "$api_url/api/worker/transcription-config" 2>/dev/null || true)
case "$status" in
 200) ;;
 401)
  # A deleted Worker is told so by the narrow confirmation endpoint: the launcher removes the service and nothing else.
  # A running Worker erases its own models volume content on that same confirmation (ADR 0021); the volume itself and this folder stay.
  revoked=$(printf 'header = "Authorization: Bearer %s"\n' "$worker_token" | curl -sS --max-time 20 -K - -o "$dir/revocation.json" -w '%{http_code}' "$api_url/api/worker/revocation" 2>/dev/null || true)
  if [ "$revoked" = 200 ] && grep -q '"revoked":true' "$dir/revocation.json" 2>/dev/null; then
   rm -f "$dir/revocation.json" "$queue"
   docker rm -f "snitch-worker-$worker_id" >/dev/null 2>&1 || true
   printf 'This Worker was deleted in Snitch and its access is revoked for good. The service was stopped.\nThe launcher erased nothing. A running Worker erases its own models and caches when it detects the deletion; the Docker volume and this folder stay: remove them yourself if you no longer need them (docker volume rm snitch-models-%s; rm -r %s).\nCe Worker a été supprimé dans Snitch et son accès est révoqué définitivement. Le service a été arrêté.\nLe lanceur n’a rien effacé. Un Worker en marche efface lui-même ses modèles et caches quand il détecte la suppression ; le volume Docker et ce dossier restent : supprimez-les vous-même si vous n’en avez plus besoin (docker volume rm snitch-models-%s ; rm -r %s).\n' "$worker_id" "$dir" "$worker_id" "$dir"
   exit 0
  fi
  rm -f "$dir/revocation.json"
  fail registration_rejected 'Snitch rejected this Worker credential (revoked or replaced). Generate a new installation command in Admin > Workers.' 'Snitch a refusé ce jeton de Worker (révoqué ou remplacé). Générez une nouvelle commande dans Admin > Workers.' ;;
 *) fail network_unreachable 'Could not reach Snitch to register this Worker. Check the network and the Snitch address, then rerun.' 'Impossible de joindre Snitch pour enregistrer ce Worker. Vérifiez le réseau et l’adresse de Snitch, puis relancez.' ;;
esac
json_field() { sed -n 's/.*"'"$1"'":"\([^"]*\)".*/\1/p' "$dir/config.json" | head -n1 | sed 's/\\u0026/\&/g' | tr -d '\r\n'; }
# Current Snitch configuration wins over what the first download carried.
transcription_endpoint=$(json_field endpoint); transcription_model=$(json_field model); transcription_language=$(json_field language)
rm -f "$dir/config.json"
emit registration succeeded
printf 'SNITCH_API_URL=%s\nSNITCH_WORKER_TOKEN=%s\nTRANSCRIPTION_API_URL=%s\nTRANSCRIPTION_MODEL=%s\nTRANSCRIPTION_LANGUAGE=%s\nSNITCH_VOICE_DOWNLOAD_ON_START=0\nSNITCH_VOICE_CONFIRM_TERMS=0\nHOME=/tmp\nXDG_CACHE_HOME=/models/.cache\nHF_HOME=/models/.cache/huggingface\nHF_XET_CACHE=/models/.cache/huggingface/xet\n' "$api_url" "$worker_token" "$transcription_endpoint" "$transcription_model" "$transcription_language" > "$env_file.tmp"
chmod 600 "$env_file.tmp"; mv "$env_file.tmp" "$env_file"; unset worker_token
stage=image
if ! docker pull "$image"; then
 docker image inspect "$image" >/dev/null 2>&1 || fail image_pull_failed 'Could not download the Worker image. Check the network and registry access, then rerun.' 'Impossible de télécharger l’image du Worker. Vérifiez le réseau et l’accès au registre, puis relancez.'
 printf 'Warning: could not refresh the image; using the local copy.\n' >&2
fi
volume="snitch-models-$worker_id"; container="snitch-worker-$worker_id"
docker volume create "$volume" >/dev/null || fail volume_not_writable 'Check Docker volume permissions, then rerun.' 'Vérifiez les permissions des volumes Docker, puis relancez.'
if [ "$wsl" = 2 ]; then
 # Reachability as the Worker sees it: from inside the Docker runtime, not from the WSL2 shell.
 probe() { docker run --rm --entrypoint python "$image" -c 'import sys, urllib.request, urllib.error
try: urllib.request.urlopen(sys.argv[1], timeout=10)
except urllib.error.HTTPError: pass' "$1"; }
 preflight=snitch-reachability
 probe "$api_url" >/dev/null 2>&1 || fail network_unreachable 'Snitch is not reachable from inside Docker. Use an address the WSL2 Docker runtime can resolve (not localhost), check the Windows firewall and VPN, then rerun.' 'Snitch est injoignable depuis Docker. Utilisez une adresse résolue par le runtime Docker de WSL2 (pas localhost), vérifiez le pare-feu Windows et le VPN, puis relancez.'
 if [ -n "$transcription_endpoint" ]; then
  preflight=transcription-reachability
  probe "$transcription_endpoint" >/dev/null 2>&1 || fail network_unreachable 'The transcription endpoint is not reachable from inside Docker. If it uses localhost or 127.0.0.1, use host.docker.internal or the machine address instead, then rerun.' 'Le point d’accès de transcription est injoignable depuis Docker. S’il utilise localhost ou 127.0.0.1, utilisez host.docker.internal ou l’adresse de la machine, puis relancez.'
 fi
 preflight=''
fi
emit image succeeded
stage=permissions
docker run --rm --user 0 --entrypoint sh -v "$volume:/models" "$image" -c 'mkdir -p /models/.cache/huggingface/xet; chown -R 10001:10001 /models' || fail volume_not_writable 'Could not prepare the model volume for UID 10001. Check Docker storage ownership, then rerun.' 'Impossible de préparer le volume de modèles pour l’UID 10001. Vérifiez les droits du stockage Docker, puis relancez.'
docker run --rm --user 10001:10001 --entrypoint sh -v "$volume:/models" "$image" -c 'touch /models/.snitch-write && rm /models/.snitch-write' || fail volume_not_writable 'The model volume is not writable by UID 10001. Check Docker storage ownership, then rerun.' 'Le volume de modèles n’est pas inscriptible par l’UID 10001. Vérifiez les droits du stockage Docker, puis relancez.'
docker run --rm --user 10001:10001 --entrypoint sh -v "$volume:/models" "$image" -c 'touch /models/.cache/huggingface/xet/.snitch-write && rm /models/.cache/huggingface/xet/.snitch-write' || fail hf_cache_not_writable 'The Hugging Face/Xet cache is not writable. Remove the volume ('"$volume"') only if it holds no models you need, then rerun.' 'Le cache Hugging Face/Xet n’est pas inscriptible. Supprimez le volume ('"$volume"') seulement s’il ne contient aucun modèle utile, puis relancez.'
emit permissions succeeded
gpu=()
if command -v nvidia-smi >/dev/null && nvidia-smi >/dev/null 2>&1; then
 gpu=(--gpus all); stage=gpu
 docker run --rm ${gpu[@]+"${gpu[@]}"} --entrypoint sh "$image" -c true || fail gpu_unavailable 'Install NVIDIA Container Toolkit for Docker and restart Docker, then rerun.' 'Installez NVIDIA Container Toolkit pour Docker et redémarrez Docker, puis relancez.'
 if [ "$wsl" = 2 ]; then
  preflight=docker-cuda
  docker run --rm --gpus all --entrypoint python "$image" -c 'import ctypes; ctypes.CDLL("libcuda.so.1")' >/dev/null 2>&1 || fail gpu_unavailable 'CUDA is not usable from Docker in WSL2. Update the Windows NVIDIA driver (with WSL support), do not install a Linux driver inside WSL, restart with "wsl --shutdown", then rerun.' 'CUDA est inutilisable depuis Docker dans WSL2. Mettez à jour le pilote NVIDIA Windows (avec prise en charge WSL), n’installez pas de pilote Linux dans WSL, redémarrez avec « wsl --shutdown », puis relancez. Windows NVIDIA driver = pilote NVIDIA Windows.'
  preflight=''
 fi
 emit gpu succeeded
else printf 'No NVIDIA GPU: voice processing unavailable (transcription still works).\n'; fi
# Voice models are optional and never block the installation or transcription.
# Voice is ready only after a synthetic inference succeeds, never because models were downloaded.
voice_selftest() {
 stage=voice-test
 if docker run --rm "${gpu[@]}" --user 10001:10001 --env-file "$env_file" -e HF_HUB_OFFLINE=1 -v "$volume:/models" --entrypoint python "$image" -m snitch_worker.voice_selftest; then emit voice-test succeeded
 else emit voice-test failed unknown; printf 'Warning [unknown]: the synthetic voice test failed; voice stays unavailable, transcription still works.\nAvertissement [unknown] : le test vocal synthétique a échoué ; la voix reste indisponible, la transcription fonctionne.\n' >&2; fi
}
voice_warn() {
 emit voice-models failed "$1"
 printf 'Warning [%s]: voice models unavailable; transcription stays available. Fix the cause, then rerun this command.\nAvertissement [%s] : modèles vocaux indisponibles ; la transcription reste disponible. Corrigez la cause, puis relancez cette commande.\n' "$1" "$1" >&2
}
if [ ${#gpu[@]} -gt 0 ]; then
 stage=voice-models
 # Models already downloaded (valid manifest) are reused: no prompt, no token, no download.
 if docker run --rm --user 10001:10001 --env-file "$env_file" -v "$volume:/models" --entrypoint python "$image" -m snitch_worker.bootstrap check >/dev/null 2>&1; then
  emit voice-models succeeded
  voice_selftest
 elif [ ! -e "$dir/.voice-declined" ] && (exec </dev/tty) 2>/dev/null; then
  printf 'Download the voice models (optional)? [y/N] ' > /dev/tty
  IFS= read -r choice < /dev/tty || choice=''
  if [ "$choice" = y ] || [ "$choice" = Y ]; then
   printf 'The models are gated by their authors on Hugging Face (pyannote/speaker-diarization-community-1, pyannote/wespeaker-voxceleb-resnet34-LM).\nType "accept" once you have accepted their terms with your Hugging Face account: ' > /dev/tty
   IFS= read -r terms < /dev/tty || terms=''
   if [ "$terms" = accept ]; then
    printf 'Hugging Face token (hidden): ' > /dev/tty
    trap 'stty echo < /dev/tty 2>/dev/null; rm -f "${bootstrap_env:-}"' EXIT
    stty -echo < /dev/tty
    IFS= read -r hf_token < /dev/tty || hf_token=''
    stty echo < /dev/tty; printf '\n' > /dev/tty
    if [[ "$hf_token" =~ ^hf_[A-Za-z0-9]+$ ]]; then
     # The token reaches the container only through this private (0600) env file, never argv, logs or worker.env.
     bootstrap_env=$(mktemp "$dir/bootstrap.XXXXXX")
     cp "$env_file" "$bootstrap_env"
     printf 'HF_TOKEN=%s\nSNITCH_VOICE_DOWNLOAD_ON_START=1\nSNITCH_VOICE_CONFIRM_TERMS=1\nSNITCH_VOICE_REQUIRE_GPU=1\n' "$hf_token" >> "$bootstrap_env"
     unset hf_token
     rc=0; cause=$(docker run --rm "${gpu[@]}" --user 10001:10001 --env-file "$bootstrap_env" -v "$volume:/models" --entrypoint python "$image" -m snitch_worker.bootstrap provision 2>/dev/null) || rc=$?
     rm -f "$bootstrap_env"; trap - EXIT
     cause=${cause##*cause=}; cause=${cause%%[!a-z_]*}
     case "$cause" in hf_access_denied|network_unreachable|volume_not_writable|hf_cache_not_writable|gpu_unavailable) ;; *) cause=unknown ;; esac
     if [ "$rc" -eq 0 ]; then emit voice-models succeeded; voice_selftest; else voice_warn "$cause"; fi
    else trap - EXIT; unset hf_token; voice_warn hf_access_denied; fi
   else printf 'Terms not confirmed: voice models skipped.\nConditions non confirmées : modèles vocaux ignorés.\n' >&2; fi
  fi
 fi
fi
stage=service
previous_image=''
if docker container inspect "$container" >/dev/null 2>&1; then
 previous_image=$(docker inspect -f '{{"{{.Image}}"}}' "$container" 2>/dev/null || true)
 docker rm -f "$container" >/dev/null || fail unknown 'Could not replace the existing container. Inspect Docker, then rerun.' 'Impossible de remplacer le conteneur existant. Inspectez Docker, puis relancez.'
fi
start_service() { docker run -d --name "$container" --restart unless-stopped --user 10001:10001 ${gpu[@]+"${gpu[@]}"} --add-host host.docker.internal:host-gateway --env-file "$env_file" -v "$volume:/models" "$1" >/dev/null; }
if ! start_service "$image"; then
 if [ -n "$previous_image" ] && docker rm -f "$container" >/dev/null 2>&1 && start_service "$previous_image"; then
  fail service_start_failed 'The new image failed to start; the previous version was restored. Inspect Docker logs, then rerun.' 'La nouvelle image ne démarre pas ; la version précédente a été restaurée. Consultez les journaux Docker, puis relancez.'
 fi
 fail service_start_failed 'Could not start the Worker. Inspect Docker logs, then rerun.' 'Impossible de démarrer le Worker. Consultez les journaux Docker, puis relancez.'
fi
emit service succeeded
printf 'Snitch Worker started. Check Admin > Workers for connection and models.\n'
`))

func (s *Server) serveWorkerInstall(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	if len(code) != 43 || strings.ContainsAny(code, "/?\x00") {
		http.NotFound(w, r)
		return
	}
	tx, err := s.Pool.Begin(r.Context())
	if err != nil {
		s.internal(w, err)
		return
	}
	defer tx.Rollback(r.Context())
	var workerID, apiURL, image, transcriptionMode, transcriptionEndpoint, transcriptionModel, transcriptionLanguage string
	var sealed []byte
	// Consumption is atomic: concurrent or repeated downloads of one invitation cannot both succeed.
	err = tx.QueryRow(r.Context(), `UPDATE worker_install_invites i SET used_at=now() FROM workers w WHERE w.id=i.worker_id AND i.code_hash=$1 AND i.used_at IS NULL AND i.expires_at>now() AND w.revoked_at IS NULL RETURNING i.worker_id,i.encrypted_token,i.api_url,i.image,i.transcription_mode,i.transcription_endpoint,i.transcription_model,i.transcription_language`, auth.Hash(code)).Scan(&workerID, &sealed, &apiURL, &image, &transcriptionMode, &transcriptionEndpoint, &transcriptionModel, &transcriptionLanguage)
	if errors.Is(err, pgx.ErrNoRows) {
		var used bool
		// Same transaction: a second pool connection here deadlocks the pool when concurrent losers hold all of them.
		if e := tx.QueryRow(r.Context(), `SELECT used_at IS NOT NULL FROM worker_install_invites WHERE code_hash=$1`, auth.Hash(code)).Scan(&used); e == nil && used {
			fail(w, http.StatusGone, "install_invite_used")
			return
		}
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
	data := struct {
		WorkerID              string
		APIURL                string
		Token                 string
		Image                 string
		TranscriptionMode     string
		TranscriptionEndpoint string
		TranscriptionModel    string
		TranscriptionLanguage string
	}{
		WorkerID:              workerID,
		APIURL:                base64.StdEncoding.EncodeToString([]byte(apiURL)),
		Token:                 base64.StdEncoding.EncodeToString([]byte(token)),
		Image:                 base64.StdEncoding.EncodeToString([]byte(image)),
		TranscriptionMode:     transcriptionMode,
		TranscriptionEndpoint: base64.StdEncoding.EncodeToString([]byte(transcriptionEndpoint)),
		TranscriptionModel:    base64.StdEncoding.EncodeToString([]byte(transcriptionModel)),
		TranscriptionLanguage: base64.StdEncoding.EncodeToString([]byte(transcriptionLanguage)),
	}
	var script bytes.Buffer
	if err = workerInstallScript.Execute(&script, data); err != nil {
		s.Log.Error("worker install script render failed")
		s.internal(w, err)
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		s.internal(w, err)
		return
	}
	_, _ = w.Write(script.Bytes())
}
