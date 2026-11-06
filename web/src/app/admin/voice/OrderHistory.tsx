import { useEffect, useState } from "react";
import { Download, Trash2 } from "lucide-react";
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

/** History is fetched per Worker by the existing contract, in parallel, with partial failures visible. */
export function OrderHistoryContent({
  workers,
  workerId,
  onWorkerChange,
  onCount,
}: {
  workers: Row[];
  workerId: string;
  onWorkerChange: (id: string) => void;
  onCount?: (count: number) => void;
}) {
  const { t, error } = useI18n();
  const [state, setState] = useState<{
    orders: (Item & { workerId: string; workerName: string })[];
    failures: string[];
  }>();
  const [version, setVersion] = useState(0);
  const revision = workers
    .map(
      (w) =>
        `${w.worker_id}:${w.name}:${w.models.map((m) => `${m.order_id}:${m.state}`).join(",")}`,
    )
    .join("|");
  useEffect(() => {
    let current = true;
    void Promise.all(
      workers.map(async (worker) => {
        try {
          const result = await api.GET(
            "/api/admin/workers/{id}/voice-model-orders",
            { params: { path: { id: worker.worker_id } } },
          );
          return result.data
            ? {
                orders: result.data.orders.map((o) => ({
                  ...o,
                  workerId: worker.worker_id,
                  workerName: worker.name,
                })),
                failure: "",
              }
            : {
                orders: [],
                failure: `${worker.name}: ${error(errorCode(result.error) ?? "unknown")}`,
              };
        } catch {
          return {
            orders: [],
            failure: `${worker.name}: ${error("unavailable")}`,
          };
        }
      }),
    ).then((results) => {
      if (!current) return;
      const orders = results
        .flatMap((r) => r.orders)
        .sort((a, b) => b.created_at.localeCompare(a.created_at));
      setState({
        orders,
        failures: results.filter((r) => r.failure).map((r) => r.failure),
      });
      if (results.every((r) => !r.failure)) onCount?.(orders.length);
    });
    return () => {
      current = false;
    };
    // Reload on membership changes; polling keeps pending orders current without resetting filters.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [revision, version, error, onCount]);
  const active = state?.orders.some(
    (o) => o.status === "pending" || o.status === "acknowledged",
  );
  useEffect(() => {
    if (!active) return;
    const timer = window.setInterval(() => setVersion((n) => n + 1), 3000);
    return () => window.clearInterval(timer);
  }, [active]);
  const visible = state?.orders.filter(
    (o) => !workerId || o.workerId === workerId,
  );
  return (
    <>
      <div className="voice-toolbar">
        <label className="flex flex-col gap-1 text-sm">
          {t("voiceModels.byWorker.historyWorker")}
          <select
            value={workerId}
            onChange={(e) => onWorkerChange(e.target.value)}
            className="h-11 max-w-full rounded-md border border-line-strong bg-surface px-2 md:h-9"
          >
            <option value="">{t("voiceWorkspace.allWorkers")}</option>
            {workers.map((w) => (
              <option key={w.worker_id} value={w.worker_id}>
                {w.name}
              </option>
            ))}
          </select>
        </label>
        <button
          type="button"
          className="voice-btn"
          onClick={() => setVersion((n) => n + 1)}
        >
          {t("voiceWorkspace.refresh")}
        </button>
      </div>
      {!state && <p role="status">{t("app.loading")}</p>}
      {state?.failures.map((failure) => (
        <ErrorAlert key={failure}>{failure}</ErrorAlert>
      ))}
      {visible?.length === 0 && (
        <div className="voice-empty">{t("voiceWorkspace.noOrders")}</div>
      )}
      <ol>
        {visible?.map((o) => (
          <li key={o.order_id} className="voice-order">
            <span className="voice-order-icon">
              {o.action === "remove" ? (
                <Trash2 size={17} aria-hidden />
              ) : (
                <Download size={17} aria-hidden />
              )}
            </span>
            <div>
              <strong>
                {t(`voiceModels.byWorker.historyAction.${o.action}`)} ·{" "}
                {o.label || o.repo}
              </strong>
              <small>
                {o.revision.slice(0, 7)} ·{" "}
                {t("voiceWorkspace.orderId", { id: o.order_id.slice(0, 8) })}
              </small>
              <p>
                {o.workerName} ·{" "}
                {t("voiceModels.byWorker.historyBy", { author: o.author_name })}{" "}
                ·{" "}
                <time dateTime={o.created_at}>
                  {new Date(o.created_at).toLocaleString()}
                </time>
              </p>
              {o.cause_code && (
                <p className="text-bad-fg">
                  {t(`voiceModels.cause.${o.cause_code as "disk_full"}`)}
                </p>
              )}
            </div>
            <StatusPill tone={tones[o.status]}>
              {t(`voiceModels.byWorker.historyStatus.${o.status}`)}
            </StatusPill>
          </li>
        ))}
      </ol>
    </>
  );
}
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
  const { t } = useI18n();
  const [workerId, setWorkerId] = useState(initialWorkerId ?? "");
  return (
    <Modal
      open={open}
      onOpenChange={(next) => !next && onClose()}
      title={t("voiceModels.byWorker.historyTitle")}
      closeLabel={t("voiceModels.byWorker.close")}
      variant="side"
    >
      {open && (
        <OrderHistoryContent
          workers={workers}
          workerId={workerId}
          onWorkerChange={setWorkerId}
        />
      )}
    </Modal>
  );
}
