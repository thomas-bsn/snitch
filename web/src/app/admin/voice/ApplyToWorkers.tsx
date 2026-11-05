import { useEffect, useState } from "react";
import type { components } from "../../../api.gen";
import { api, errorCode, mutation } from "../../api/client";
import { buttonClass } from "../../components/buttons";
import { Modal } from "../../components/Modal";
import { useI18n } from "../../i18n";
import { useSession } from "../../session";
import type { Catalogue } from "./VoiceModelCards";

type Row = components["schemas"]["WorkerModelsRow"];
type Kind = components["schemas"]["WorkerModelCell"]["kind"];

type Order = { worker: Row; key: string };

/**
 * "Apply to several Workers": an explicit action. One order is built per ticked Worker and listed before
 * anything is sent; each order is then sent on its own with its own key, and its outcome is shown line by line.
 */
export function ApplyToWorkers({
  open,
  onClose,
  workers,
  catalogue,
  kinds,
}: {
  open: boolean;
  onClose: () => void;
  workers: Row[];
  catalogue: Catalogue;
  kinds: Kind[];
}) {
  const { t, error } = useI18n();
  const { session } = useSession();
  const headers = mutation(session?.csrf_token);
  const [kind, setKind] = useState<Kind>("diarization");
  const [modelId, setModelId] = useState("");
  const [ticked, setTicked] = useState<string[]>([]);
  const [orders, setOrders] = useState<Order[]>();
  const [outcome, setOutcome] = useState<Record<string, string>>({});
  const [busy, setBusy] = useState(false);
  const models = catalogue.models.filter(
    (m) => m.kind === kind && m.enabled && m.supported,
  );
  const eligible = workers.filter((w) =>
    w.models.some((m) => m.kind === kind && m.state === "not_installed"),
  );
  useEffect(() => {
    if (!open) return;
    setOrders(undefined);
    setOutcome({});
    setTicked([]);
  }, [open]);
  useEffect(() => {
    setModelId(models[0]?.id ?? "");
    setTicked([]);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [kind, catalogue]);
  const label = (id: string) => {
    const m = catalogue.models.find((x) => x.id === id);
    return m?.label || m?.repo || id;
  };
  async function send() {
    if (!orders) return;
    setBusy(true);
    for (const order of orders) {
      if (outcome[order.worker.worker_id] === "sent") continue;
      try {
        const result = await api.POST(
          "/api/admin/workers/{id}/voice-model-orders",
          {
            params: {
              path: { id: order.worker.worker_id },
              header: headers,
            },
            body: { model_id: modelId, idempotency_key: order.key },
          },
        );
        setOutcome((o) => ({
          ...o,
          [order.worker.worker_id]: result.error
            ? error(errorCode(result.error) ?? "unknown")
            : "sent",
        }));
      } catch {
        setOutcome((o) => ({
          ...o,
          [order.worker.worker_id]: error("unavailable"),
        }));
      }
    }
    setBusy(false);
  }
  const allSent =
    orders !== undefined &&
    orders.every((o) => outcome[o.worker.worker_id] === "sent");
  return (
    <Modal
      open={open}
      onOpenChange={(next) => !next && onClose()}
      title={t("voiceModels.byWorker.applyTitle")}
      closeLabel={t("voiceModels.byWorker.close")}
    >
      <p className="text-mute">{t("voiceModels.byWorker.applyHint")}</p>
      {orders === undefined ? (
        <>
          <label className="flex flex-col gap-1 text-sm">
            {t("voiceModels.byWorker.applyType")}
            <select
              value={kind}
              onChange={(e) => setKind(e.target.value as Kind)}
              className="h-11 rounded-md border border-line-strong bg-surface px-2 md:h-9"
            >
              {kinds.map((k) => (
                <option key={k} value={k}>
                  {t(`voiceModels.byWorker.type.${k}`)}
                </option>
              ))}
            </select>
          </label>
          <label className="flex flex-col gap-1 text-sm">
            {t("voiceModels.byWorker.applyModel")}
            <select
              value={modelId}
              onChange={(e) => setModelId(e.target.value)}
              className="h-11 rounded-md border border-line-strong bg-surface px-2 md:h-9"
            >
              {models.map((m) => (
                <option key={m.id} value={m.id}>
                  {m.label || m.repo}
                </option>
              ))}
            </select>
          </label>
          <fieldset className="flex flex-col gap-1">
            <legend className="mb-1 text-sm">
              {t("voiceModels.byWorker.applyWorkers")}
            </legend>
            {eligible.length === 0 && (
              <p className="text-mute">{t("voiceModels.byWorker.applyNone")}</p>
            )}
            {eligible.map((w) => (
              <label
                key={w.worker_id}
                className="flex min-h-11 items-center gap-2 md:min-h-8"
              >
                <input
                  type="checkbox"
                  className="size-4"
                  checked={ticked.includes(w.worker_id)}
                  onChange={(e) =>
                    setTicked(
                      e.target.checked
                        ? [...ticked, w.worker_id]
                        : ticked.filter((id) => id !== w.worker_id),
                    )
                  }
                />
                {w.name}
              </label>
            ))}
          </fieldset>
          <div className="flex justify-end">
            <button
              type="button"
              disabled={ticked.length === 0 || modelId === ""}
              onClick={() =>
                setOrders(
                  workers
                    .filter((w) => ticked.includes(w.worker_id))
                    .map((worker) => ({ worker, key: crypto.randomUUID() })),
                )
              }
              className={buttonClass("primary")}
            >
              {t("voiceModels.byWorker.applyPrepare")}
            </button>
          </div>
        </>
      ) : (
        <>
          <h4 className="font-medium">
            {t("voiceModels.byWorker.applyPreview", { count: orders.length })}
          </h4>
          <ul className="flex flex-col divide-y divide-line rounded-lg border border-line">
            {orders.map((o) => {
              const result = outcome[o.worker.worker_id];
              return (
                <li key={o.worker.worker_id} className="px-3 py-2">
                  {t("voiceModels.byWorker.applyLine", {
                    model: label(modelId),
                    worker: o.worker.name,
                  })}
                  {result === "sent" && (
                    <span className="block text-ok-fg">
                      {t("voiceModels.byWorker.applySent")}
                    </span>
                  )}
                  {result && result !== "sent" && (
                    <span role="alert" className="block text-bad-fg">
                      {t("voiceModels.byWorker.applyFailed", {
                        reason: result,
                      })}
                    </span>
                  )}
                </li>
              );
            })}
          </ul>
          <div className="flex justify-end gap-2">
            <button
              type="button"
              disabled={busy}
              onClick={() => {
                setOrders(undefined);
                setOutcome({});
              }}
              className={buttonClass("secondary")}
            >
              {t("voiceModels.byWorker.applyBack")}
            </button>
            {allSent ? (
              <button
                type="button"
                onClick={onClose}
                className={buttonClass("primary")}
              >
                {t("voiceModels.byWorker.close")}
              </button>
            ) : (
              <button
                type="button"
                disabled={busy}
                onClick={() => void send()}
                className={buttonClass("primary")}
              >
                {t("voiceModels.byWorker.applySend", { count: orders.length })}
              </button>
            )}
          </div>
        </>
      )}
    </Modal>
  );
}
