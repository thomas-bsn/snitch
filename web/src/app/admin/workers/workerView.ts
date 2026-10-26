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
}): { tone: PillTone; label: MessageKey } {
  if (worker.revoked_at)
    return { tone: "bad", label: "workers.status.revoked" };
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
