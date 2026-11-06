import type { components } from "../../../api.gen";
import type { MessageKey } from "../../i18n/messages";
import type { PillTone } from "../../components/StatusPill";

type Models = components["schemas"]["WorkerModels"];
type Role = "transcription" | "diarization" | "fingerprint";

export const ROLES: Role[] = ["transcription", "diarization", "fingerprint"];

/** State of a Worker as a tone and a text key (a label always goes with the color). */
export function workerStatus(worker: {
  connected: boolean;
  revoked_at: string | null;
  claim_stalled?: boolean;
}): { tone: PillTone; label: MessageKey } {
  if (worker.revoked_at)
    return { tone: "bad", label: "workers.status.revoked" };
  // Presence alone does not say the Worker can take jobs (#321).
  if (worker.connected && worker.claim_stalled)
    return { tone: "bad", label: "workers.status.notClaiming" };
  return worker.connected
    ? { tone: "ok", label: "workers.status.online" }
    : { tone: "idle", label: "workers.status.offline" };
}

/** "name version" of a declared model, or nothing when the role is not declared. */
export function declaredModelText(models: Models | undefined, role: Role) {
  const model = models?.[role];
  if (!model) return undefined;
  return model.version ? `${model.model} ${model.version}` : model.model;
}

/** Declared model names of a Worker, in role order. */
export function workerModelSummary(models: Models | undefined): string[] {
  return ROLES.flatMap((role) => declaredModelText(models, role) ?? []);
}

type Install = components["schemas"]["WorkerInstall"];

const CAUSES = [
  "unsupported_platform",
  "docker_unavailable",
  "registration_rejected",
  "network_unreachable",
  "image_pull_failed",
  "volume_not_writable",
  "hf_cache_not_writable",
  "gpu_unavailable",
  "service_start_failed",
  "hf_access_denied",
  "unknown",
] as const;
export type InstallCause = (typeof CAUSES)[number];

/** A stable cause code from Snitch; anything unexpected reads as "unknown". */
export function installCause(code: string): InstallCause {
  return (CAUSES as readonly string[]).includes(code)
    ? (code as InstallCause)
    : "unknown";
}

const INSTALL_TONES: Record<Install["state"], PillTone> = {
  installing: "idle",
  connected: "ok",
  voice_ready: "ok",
  failed: "bad",
  never_connected: "idle",
};

/** Installation state as a tone and a text key (a label always goes with the color). */
export function installStatus(install: Install): {
  tone: PillTone;
  label: MessageKey;
} {
  return {
    tone: INSTALL_TONES[install.state],
    label: `workers.install.state.${install.state}`,
  };
}

/** Transcription status, independent of installation: an unconfigured service is only "waiting". */
export function transcriptionState(
  config: components["schemas"]["WorkerTranscriptionConfig"] | undefined,
): "active" | "error" | "waiting" {
  if (config?.error_code) return "error";
  return config?.active_version ? "active" : "waiting";
}

export const RECOVERY_HINTS = [
  "docker_daemon",
  "wsl_stopped",
  "container_stopped",
  "presence_thread",
  "token_rejected",
  "unreachable",
  "installer_missing",
  "no_pulse_after_restart",
] as const;
export type RecoveryHint = (typeof RECOVERY_HINTS)[number];

/** What the administrator can try, most likely first for the known installation state. Snitch only
 * knows the last pulse, not the cause: these are leads to check, never a diagnosis. */
export function recoveryHints(
  install: Pick<Install, "state"> | undefined,
): RecoveryHint[] {
  const never =
    install?.state === "never_connected" || install?.state === "installing";
  if (never)
    return [
      "installer_missing",
      "docker_daemon",
      "wsl_stopped",
      "unreachable",
      "token_rejected",
    ];
  // A Worker that already pulsed was installed: a missing launcher is not a lead for it.
  return RECOVERY_HINTS.filter((hint) => hint !== "installer_missing");
}

/** Local commands for one Worker, run on its own machine. They carry only its identifier: never a
 * token or an invitation link. */
export function restartCommands(workerId: string | number) {
  return {
    restart: `docker restart snitch-worker-${workerId}`,
    logs: `docker logs --tail 50 snitch-worker-${workerId}`,
    launcher: `bash ~/.local/share/snitch-worker/${workerId}/install.sh`,
  };
}

/** True only when a pulse newer than the one seen when the panel opened has arrived and the Worker
 * is connected again: no reconnection is ever announced from an action alone. */
export function hasReturned(
  worker: { connected: boolean; last_seen_at: string | null },
  baseline: string | null,
): boolean {
  if (!worker.connected || !worker.last_seen_at) return false;
  if (baseline === null) return true;
  return Date.parse(worker.last_seen_at) > Date.parse(baseline);
}
