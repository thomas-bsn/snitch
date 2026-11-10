import { useCallback, useEffect, useState } from "react";
import type { components } from "../../../api.gen";
import { api, errorCode, mutation } from "../../api/client";
import { useLoaded, type Result } from "../../api/loaded";
import { ErrorAlert } from "../../components/Alert";
import { Modal } from "../../components/Modal";
import { useI18n } from "../../i18n";
import { useSession } from "../../session";
import { WorkerHardware } from "./WorkerHardware";
import {
  engineTargets,
  type EngineEntry,
  type EngineWorker,
} from "./engineOrders";

type Family = components["schemas"]["EngineFamily"];
type Plan = components["schemas"]["OnDemandEnginePlan"];

type Props = {
  /** A failed on-demand engine being replaced: its family and task are fixed, only the version changes. */
  replacing?: EngineEntry;
  worker: EngineWorker;
  workers: EngineWorker[];
  checked: string[];
  busy: boolean;
  onClose: () => void;
  onOrder: (
    family: string,
    version: string,
    workerIds: string[],
    kind?: string,
  ) => Promise<void>;
};

async function fetchFamilies(): Promise<Result<Family[]>> {
  try {
    const { data, error } = await api.GET("/api/admin/voice-engines/families");
    return data
      ? { ok: true, value: data.families }
      : { ok: false, code: errorCode(error) ?? "unknown" };
  } catch {
    return { ok: false, code: "unavailable" };
  }
}

const gibibytes = (bytes: number) => Math.round((bytes / 1024 ** 3) * 10) / 10;
const minutes = (seconds: number) => Math.max(1, Math.ceil(seconds / 60));

/**
 * Add an engine from the families Snitch supports: the default version is pre-filled and editable, the plan is
 * computed by the server (hardware of each Worker included) and nothing is ordered before the confirmation.
 */
export function WorkerEngineAdd({
  replacing,
  worker,
  workers,
  checked,
  busy,
  onClose,
  onOrder,
}: Props) {
  const { t, error } = useI18n();
  const { session } = useSession();
  const families = useLoaded(useCallback(fetchFamilies, []));
  const [familyKey, setFamilyKey] = useState("");
  const [typed, setTyped] = useState<{ family: string; value: string }>();
  const [scope, setScope] = useState<"this" | "checked">("this");
  const [plan, setPlan] = useState<{
    key: string;
    plan?: Plan;
    code?: string;
  }>();
  const list = families.state.status === "ready" ? families.state.value : [];
  const family = replacing
    ? list.find((f) => f.engine === replacing.engine)
    : (list.find((f) => f.family === familyKey) ?? list[0]);
  const kind = replacing?.kind;
  // The version follows the family until the administrator edits it.
  const version =
    typed && typed.family === family?.family
      ? typed.value
      : (family?.default_version ?? "");
  const targets = engineTargets(workers, worker, checked, scope);
  const ids = targets.map((w) => w.worker_id);
  const key = `${family?.family}|${kind ?? ""}|${version.trim()}|${ids.join(",")}`;
  const csrf = session?.csrf_token;

  useEffect(() => {
    if (!family || ids.length === 0 || !version.trim()) return;
    let current = true;
    const timer = setTimeout(async () => {
      try {
        const { data, error: failure } = await api.POST(
          "/api/admin/voice-engines/on-demand-plan",
          {
            params: { header: mutation(csrf) },
            body: {
              family: family.family,
              version: version.trim(),
              worker_ids: ids,
              ...(kind ? { kind } : {}),
            },
          },
        );
        if (current)
          setPlan(
            data
              ? { key, plan: data }
              : { key, code: errorCode(failure) ?? "unknown" },
          );
      } catch {
        if (current) setPlan({ key, code: "unavailable" });
      }
    }, 250);
    return () => {
      current = false;
      clearTimeout(timer);
    };
    // `key` summarises family, version and Workers.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [key, csrf]);

  const fresh = plan?.key === key ? plan : undefined;
  const result = fresh?.plan;
  const nameOf = (id: string) =>
    workers.find((w) => w.worker_id === id)?.name ?? id;
  const accepted = result?.workers.filter((w) => w.status === "ok") ?? [];
  const refusedList = result?.workers.filter((w) => w.status !== "ok") ?? [];

  return (
    <Modal
      open
      onOpenChange={(open) => !open && !busy && onClose()}
      title={t("workerEngines.addTitle", { worker: worker.name })}
      closeLabel={t("voiceModels.byWorker.close")}
      variant="wide"
    >
      <p className="text-mute">
        {replacing
          ? t("workerEngines.action.replaceLead", { engine: replacing.engine })
          : t("workerEngines.family.lead")}
      </p>
      <WorkerHardware worker={worker} />
      {families.state.status === "loading" && (
        <p role="status">{t("workerEngines.family.loading")}</p>
      )}
      {families.state.status === "failed" && (
        <ErrorAlert>{error(families.state.code)}</ErrorAlert>
      )}
      {families.state.status === "ready" && list.length === 0 && (
        <p className="engine-empty">{t("workerEngines.family.none")}</p>
      )}
      {family && (
        <div className="engine-registry">
          <label>
            {t("workerEngines.family.choose")}
            <select
              className="snitch-select"
              value={family.family}
              disabled={!!replacing}
              onChange={(e) => setFamilyKey(e.target.value)}
            >
              {list.map((f) => (
                <option key={f.family} value={f.family}>
                  {f.engine}
                </option>
              ))}
            </select>
          </label>
          <span className="engine-sub">
            {t("workerEngines.family.meta", {
              kinds: family.kinds.join(", "),
              disk: gibibytes(family.disk_bytes),
              minutes: minutes(family.estimated_seconds),
            })}
          </span>
          <label>
            {t("workerEngines.family.version")}
            <input
              type="text"
              className="h-11 rounded-lg border border-line-strong bg-surface px-3 text-ink md:h-10"
              value={version}
              maxLength={64}
              spellCheck={false}
              autoComplete="off"
              aria-describedby="engine-version-hint"
              onChange={(e) =>
                setTyped({ family: family.family, value: e.target.value })
              }
            />
          </label>
          <span id="engine-version-hint" className="engine-sub">
            {t("workerEngines.family.versionHint", {
              version: family.default_version,
            })}
          </span>
          {family.recent_versions.length > 0 && (
            <div
              className="engine-versions"
              role="group"
              aria-label={t("workerEngines.family.recent")}
            >
              <span className="engine-sub">
                {t("workerEngines.family.recent")}
              </span>
              {family.recent_versions.map((v) => (
                <button
                  key={v}
                  type="button"
                  className="engine-version-chip"
                  aria-pressed={version.trim() === v}
                  onClick={() => setTyped({ family: family.family, value: v })}
                >
                  {v}
                </button>
              ))}
            </div>
          )}
        </div>
      )}
      <div className="engine-bar">
        <label>
          {t("workerEngines.scopeOn")}
          <select
            value={scope}
            onChange={(e) => setScope(e.target.value as "this" | "checked")}
          >
            <option value="this">{t("workerEngines.scopeThis")}</option>
            <option value="checked">
              {t("workerEngines.scopeChecked", { count: checked.length })}
            </option>
          </select>
        </label>
        <div className="engine-plan" aria-live="polite">
          {ids.length === 0 && (
            <span>{t("workerEngines.family.noWorker")}</span>
          )}
          {ids.length > 0 && !fresh && family && (
            <span>{t("workerEngines.family.planLoading")}</span>
          )}
          {fresh?.code && (
            <span className="engine-refused">
              {fresh.code === "invalid_engine_order"
                ? t("workerEngines.family.invalidVersion")
                : error(fresh.code)}
            </span>
          )}
          {result && (
            <span>
              {t("workerEngines.family.plan", {
                count: result.accepted,
                disk: gibibytes(result.disk_bytes),
                minutes: minutes(result.estimated_seconds),
              })}
            </span>
          )}
          {refusedList.map((w) => (
            <span key={w.worker_id} className="engine-refused">
              {t("workerEngines.family.refusedWorker", {
                worker: nameOf(w.worker_id),
                reason: error(w.cause_code ?? "unknown"),
              })}
              {w.required_bytes !== undefined &&
                ` (${t("workerEngines.family.refusedDetail", {
                  required: gibibytes(w.required_bytes),
                  available: gibibytes(w.available_bytes ?? 0),
                })})`}
            </span>
          ))}
          {targets.some((w) => !w.connected) && (
            <span className="engine-note">
              {t("workerEngines.offlineNote")}
            </span>
          )}
        </div>
        <div className="engine-actions">
          <button
            type="button"
            className="voice-btn"
            disabled={busy}
            onClick={onClose}
          >
            {t("workerEngines.cancel")}
          </button>
          <button
            type="button"
            className="voice-btn primary"
            disabled={busy || !family || accepted.length === 0}
            onClick={() =>
              family &&
              void onOrder(
                family.family,
                version.trim(),
                accepted.map((w) => w.worker_id),
                kind,
              )
            }
          >
            {t("workerEngines.install", { count: accepted.length })}
          </button>
        </div>
      </div>
    </Modal>
  );
}
