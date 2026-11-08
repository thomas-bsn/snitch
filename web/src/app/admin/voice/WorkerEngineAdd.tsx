import { useState } from "react";
import { ErrorAlert } from "../../components/Alert";
import { Modal } from "../../components/Modal";
import { useI18n } from "../../i18n";
import {
  planOrders,
  refusalFor,
  type EngineEntry,
  type EngineWorker,
  type Release,
} from "./engineOrders";

type Props = {
  /** A failed installation being replaced: only the other releases of the same engine are offered. */
  replacing?: EngineEntry;
  worker: EngineWorker;
  workers: EngineWorker[];
  checked: string[];
  releases: Release[];
  registryError?: string;
  busy: boolean;
  onClose: () => void;
  onOrder: (
    items: { worker: EngineWorker; release: Release }[],
  ) => Promise<void>;
};

const mebibytes = (bytes: number) => Math.ceil(bytes / 1024 ** 2);

/** The registry of approved releases, seen from one Worker. Nothing starts before the confirmation. */
export function WorkerEngineAdd({
  replacing,
  worker,
  workers,
  checked,
  releases,
  registryError,
  busy,
  onClose,
  onOrder,
}: Props) {
  const { t, error } = useI18n();
  const [query, setQuery] = useState("");
  const [picked, setPicked] = useState<string[]>([]);
  const [scope, setScope] = useState<"this" | "checked">("this");
  const shown = releases
    .filter(
      (r) =>
        !replacing ||
        (r.manifest.engine === replacing.engine &&
          r.digest !== replacing.digest),
    )
    .filter((r) =>
      `${r.manifest.engine} ${r.manifest.version}`
        .toLowerCase()
        .includes(query.toLowerCase()),
    )
    .sort(
      (a, b) =>
        Number(a.revoked) - Number(b.revoked) ||
        a.manifest.engine.localeCompare(b.manifest.engine),
    );
  const targets =
    scope === "checked" && checked.length
      ? workers.filter((w) => checked.includes(w.worker_id))
      : [worker];
  const plan = planOrders(
    targets,
    releases.filter((r) => picked.includes(r.digest)),
  );
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
          : t("workerEngines.addLead")}
      </p>
      {registryError && <ErrorAlert>{error(registryError)}</ErrorAlert>}
      <input
        type="search"
        className="engine-search"
        value={query}
        placeholder={t("workerEngines.addSearch")}
        aria-label={t("workerEngines.addSearch")}
        onChange={(e) => setQuery(e.target.value)}
      />
      <div className="engine-registry">
        {shown.length === 0 && (
          <p className="engine-empty">{t("workerEngines.registryEmpty")}</p>
        )}
        {shown.map((r) => {
          const refusal = refusalFor(r, worker);
          const m = r.manifest;
          return (
            <label
              key={r.digest}
              className={`engine-row ${refusal ? "disabled" : ""}`}
            >
              <input
                type="checkbox"
                disabled={!!refusal}
                checked={picked.includes(r.digest)}
                onChange={(e) =>
                  setPicked((p) =>
                    e.target.checked
                      ? [...p, r.digest]
                      : p.filter((d) => d !== r.digest),
                  )
                }
              />
              <div>
                <strong>
                  {m.engine} · {m.version}
                </strong>
                <span className="engine-sub">
                  {t("workerEngines.constraints", {
                    kind: m.kind,
                    disk: mebibytes(m.disk_bytes),
                    vram: mebibytes(m.vram_bytes),
                    cuda: m.cuda,
                  })}
                </span>
              </div>
              <div>
                {refusal && (
                  <span className="engine-refused">
                    {t(`workerEngines.refusal.${refusal}`)}
                  </span>
                )}
              </div>
            </label>
          );
        })}
      </div>
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
          <span>
            {t("workerEngines.plan", {
              count: plan.accepted.length,
              disk: mebibytes(plan.disk),
            })}
          </span>
          {plan.refusals.map(([reason, n]) => (
            <span key={reason} className="engine-refused">
              {t("workerEngines.refused", {
                count: n,
                reason: t(`workerEngines.refusal.${reason}`),
              })}
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
            disabled={busy || plan.accepted.length === 0}
            onClick={() => void onOrder(plan.accepted)}
          >
            {t("workerEngines.install", { count: plan.accepted.length })}
          </button>
        </div>
      </div>
    </Modal>
  );
}
