import { useEffect, useState } from "react";
import type { components } from "../../../api.gen";
import { api, errorCode } from "../../api/client";
import { ErrorAlert } from "../../components/Alert";
import { Modal } from "../../components/Modal";
import { StatusPill, type PillTone } from "../../components/StatusPill";
import { useI18n } from "../../i18n";

type Item = components["schemas"]["VoiceModelOrderHistoryItem"];
type Row = components["schemas"]["WorkerModelsRow"];

const tones: Record<Item["status"], PillTone> = {
  pending: "busy",
  acknowledged: "busy",
  done: "ok",
  failed: "bad",
  refused: "bad",
  cancelled: "idle",
};

/** The orders Snitch gave to one Worker, newest first: what was asked, by whom, and how it ended. */
export function OrderHistory({
  open,
  onClose,
  workers,
  initialWorkerId,
}: {
  open: boolean;
  onClose: () => void;
  workers: Row[];
  initialWorkerId?: string;
}) {
  const { t, error } = useI18n();
  const [workerId, setWorkerId] = useState("");
  const [orders, setOrders] = useState<Item[]>();
  const [failure, setFailure] = useState<string>();
  useEffect(() => {
    if (open) setWorkerId(initialWorkerId ?? workers[0]?.worker_id ?? "");
  }, [open, initialWorkerId, workers]);
  useEffect(() => {
    if (!open || workerId === "") return;
    let current = true;
    setOrders(undefined);
    setFailure(undefined);
    api
      .GET("/api/admin/workers/{id}/voice-model-orders", {
        params: { path: { id: workerId } },
      })
      .then((result) => {
        if (!current) return;
        if (result.data) setOrders(result.data.orders);
        else setFailure(errorCode(result.error) ?? "unknown");
      })
      .catch(() => current && setFailure("unavailable"));
    return () => {
      current = false;
    };
  }, [open, workerId]);
  return (
    <Modal
      open={open}
      onOpenChange={(next) => !next && onClose()}
      title={t("voiceModels.byWorker.historyTitle")}
      closeLabel={t("voiceModels.byWorker.close")}
      variant="side"
    >
      <label className="flex flex-col gap-1 text-sm">
        {t("voiceModels.byWorker.historyWorker")}
        <select
          value={workerId}
          onChange={(e) => setWorkerId(e.target.value)}
          className="h-11 rounded-md border border-line-strong bg-surface px-2 md:h-9"
        >
          {workers.map((w) => (
            <option key={w.worker_id} value={w.worker_id}>
              {w.name}
            </option>
          ))}
        </select>
      </label>
      {failure && <ErrorAlert>{error(failure)}</ErrorAlert>}
      {orders && orders.length === 0 && (
        <p className="text-mute">{t("voiceModels.byWorker.historyEmpty")}</p>
      )}
      {orders && orders.length > 0 && (
        <ol className="flex flex-col divide-y divide-line">
          {orders.map((o) => (
            <li key={o.order_id} className="flex flex-col gap-1 py-3">
              <div className="flex flex-wrap items-center justify-between gap-2">
                <span className="font-medium">
                  {t(`voiceModels.byWorker.historyAction.${o.action}`)} ·{" "}
                  {o.label || o.repo}
                </span>
                <StatusPill tone={tones[o.status]}>
                  {t(`voiceModels.byWorker.historyStatus.${o.status}`)}
                </StatusPill>
              </div>
              <span className="text-xs text-mute">
                <time dateTime={o.created_at}>
                  {new Date(o.created_at).toLocaleString()}
                </time>{" "}
                ·{" "}
                {t("voiceModels.byWorker.historyBy", { author: o.author_name })}
              </span>
              {o.cause_code !== "" && (
                <span className="text-bad-fg">
                  {t(`voiceModels.cause.${o.cause_code as "disk_full"}`)}
                </span>
              )}
            </li>
          ))}
        </ol>
      )}
    </Modal>
  );
}
