import { useState } from "react";
import { History, Plus, Clock3, Server } from "lucide-react";
import type { components } from "../../../api.gen";
import { api, errorCode, mutation } from "../../api/client";
import { ErrorAlert } from "../../components/Alert";
import { buttonClass } from "../../components/buttons";
import { ConfirmDialog } from "../../components/ConfirmDialog";
import { useI18n } from "../../i18n";
import { useSession } from "../../session";
import type { Catalogue } from "./VoiceModelCards";
import { KindIcon, VoiceBadge, modelKinds } from "./VoiceModelUI";

type Cell = components["schemas"]["WorkerModelCell"];
export type Row = components["schemas"]["WorkerModelsRow"];
export type Kind = Cell["kind"];

const megabytes = (bytes: number) => Math.round(bytes / 1_000_000);

type Pending =
  | {
      type: "install";
      worker: Row;
      kind: Kind;
      modelId: string;
      /** The model this one replaces on the Worker: a replacement, not a plain installation. */
      replaces?: string;
    }
  | { type: "retry"; worker: Row; kind: Kind; modelId: string }
  | { type: "uninstall"; worker: Row; kind: Kind; modelId: string }
  | {
      type: "removeOld";
      worker: Row;
      kind: Kind;
      modelId: string;
      /** The new model that failed for lack of space. */
      successor: string;
    };

function progress(cell: Cell) {
  if (cell.bytes_total > 0)
    return Math.min(100, (cell.bytes_received / cell.bytes_total) * 100);
  return cell.state === "verifying" ? 100 : 4;
}

/**
 * "Models per Worker" (#265): a dense table, one row per Worker and one column per type. Snitch orders; the
 * Worker reports what it really holds. Every status has a symbol and a word, never the colour alone.
 */
export function ModelsByWorker({
  catalogue,
  workers,
  selected,
  onSelected,
  onChanged,
  onHistory,
  onInspect,
  onTransition,
}: {
  catalogue: Catalogue;
  workers: Row[];
  selected: string[];
  onSelected: (ids: string[]) => void;
  onChanged: () => void;
  onHistory: (id: string) => void;
  onInspect: (id: string) => void;
  onTransition: (id: string) => void;
}) {
  const { t, error } = useI18n();
  const { session } = useSession();
  const [orderKey, setOrderKey] = useState("");
  const [choice, setChoice] = useState<string>("");
  const [pending, setPending] = useState<Pending>();
  const [busy, setBusy] = useState(false);
  const [failure, setFailure] = useState<string>();
  const headers = mutation(session?.csrf_token);
  const reload = onChanged;
  const labelOf = (id: string) => {
    const model = catalogue.models.find((m) => m.id === id);
    return model?.label || model?.repo || id;
  };
  const options = (kind: Kind, except = "") =>
    catalogue.models.filter(
      (m) => m.kind === kind && m.enabled && m.supported && m.id !== except,
    );
  const kindLabel = (kind: Kind) => t(`voiceModels.byWorker.type.${kind}`);

  function open(next: Pending) {
    setOrderKey(crypto.randomUUID());
    setFailure(undefined);
    setChoice(next.type === "install" ? next.modelId : "");
    setPending(next);
  }

  async function run(item: Pending) {
    setBusy(true);
    setFailure(undefined);
    try {
      const id = { path: { id: item.worker.worker_id }, header: headers };
      const key = orderKey;
      const modelId =
        item.type === "install" ? choice || item.modelId : item.modelId;
      let result;
      if (item.type === "retry")
        result = await api.POST("/api/admin/workers/{id}/voice-models/retry", {
          params: id,
          body: { model_id: modelId, idempotency_key: key },
        });
      else if (item.type === "install")
        result = await api.POST("/api/admin/workers/{id}/voice-model-orders", {
          params: id,
          body: {
            model_id: modelId,
            idempotency_key: key,
            ...(item.replaces ? { replaces_model_id: item.replaces } : {}),
          },
        });
      else
        result = await api.POST(
          "/api/admin/workers/{id}/voice-model-removals",
          {
            params: id,
            body: { model_id: modelId, idempotency_key: key },
          },
        );
      if (result.error) setFailure(errorCode(result.error) ?? "unknown");
      else {
        setPending(undefined);
        reload();
      }
    } catch {
      setFailure("unavailable");
    } finally {
      setBusy(false);
    }
  }

  async function cancelReplacement(worker: Row, kind: Kind) {
    setBusy(true);
    setFailure(undefined);
    try {
      const result = await api.DELETE(
        "/api/admin/workers/{id}/voice-model-replacements/{kind}",
        {
          params: {
            path: { id: worker.worker_id, kind },
            header: headers,
          },
        },
      );
      if (result.error) setFailure(errorCode(result.error) ?? "unknown");
      else reload();
    } catch {
      setFailure("unavailable");
    } finally {
      setBusy(false);
    }
  }

  const modelName = (id: string) => labelOf(id);
  const dialog = pending
    ? (() => {
        const worker = pending.worker.name;
        const model = modelName(
          pending.type === "install"
            ? choice || pending.modelId
            : pending.modelId,
        );
        const gated =
          pending.type === "install" &&
          catalogue.models.find((m) => m.id === (choice || pending.modelId))
            ?.gated;
        switch (pending.type) {
          case "install":
            return {
              danger: false,
              confirm: t("voiceModels.byWorker.confirmAction"),
              title: pending.replaces
                ? t("voiceModels.byWorker.confirmTitle", { model, worker })
                : t("voiceModels.byWorker.confirmTitle", { model, worker }),
              body: (
                <>
                  <p>
                    {t(
                      pending.replaces
                        ? "voiceModels.byWorker.confirmReplaceText"
                        : "voiceModels.byWorker.confirmText",
                      {
                        model,
                        worker,
                        old: pending.replaces
                          ? modelName(pending.replaces)
                          : "",
                      },
                    )}
                  </p>
                  {gated && <p>{t("voiceModels.byWorker.confirmGated")}</p>}
                </>
              ),
            };
          case "retry":
            return {
              danger: false,
              confirm: t("voiceModels.byWorker.confirmAction"),
              title: t("voiceModels.byWorker.confirmTitle", { model, worker }),
              body: (
                <p>
                  {t("voiceModels.byWorker.confirmText", { model, worker })}
                </p>
              ),
            };
          case "uninstall":
            return {
              danger: true,
              confirm: t("voiceModels.byWorker.uninstallAction"),
              title: t("voiceModels.byWorker.uninstallTitle", {
                model,
                worker,
              }),
              body: (
                <>
                  <p>
                    {t("voiceModels.byWorker.uninstallText", { model, worker })}
                  </p>
                  <p className="font-medium text-ink">
                    {t("voiceModels.byWorker.uninstallNoActive", { worker })}
                  </p>
                </>
              ),
            };
          case "removeOld":
            return {
              danger: true,
              confirm: t("voiceModels.byWorker.removeOldAction", {
                old: model,
              }),
              title: t("voiceModels.byWorker.removeOldTitle", {
                old: model,
                worker,
              }),
              body: (
                <>
                  <p>
                    {t("voiceModels.byWorker.removeOldText", {
                      old: model,
                      model: modelName(pending.successor),
                      worker,
                    })}
                  </p>
                  <p className="font-medium text-ink">
                    {t("voiceModels.byWorker.removeOldNoActive", {
                      old: model,
                      model: modelName(pending.successor),
                      worker,
                    })}
                  </p>
                </>
              ),
            };
        }
      })()
    : undefined;
  const installPick =
    pending?.type === "install" ? options(pending.kind, pending.replaces) : [];

  return (
    <section aria-label={t("voiceModels.byWorker.title")}>
      {failure && !pending && <ErrorAlert>{error(failure)}</ErrorAlert>}
      {workers.length === 0 ? (
        <div className="voice-empty">{t("voiceWorkspace.noWorkers")}</div>
      ) : (
        <div
          role="region"
          tabIndex={0}
          aria-label={t("voiceModels.byWorker.tableLabel")}
          className="voice-table-scroll"
        >
          <table className="voice-matrix">
            <thead>
              <tr>
                <th scope="col">
                  <span className="sr-only">
                    {t("voiceModels.byWorker.applyWorkers")}
                  </span>
                </th>
                <th scope="col">{t("voiceModels.byWorker.worker")}</th>
                {modelKinds.map((kind) => (
                  <th scope="col" key={kind}>
                    <span>
                      <KindIcon kind={kind} />
                      {kindLabel(kind)}
                    </span>
                  </th>
                ))}
                <th scope="col">
                  <span className="sr-only">{t("voiceWorkspace.history")}</span>
                </th>
              </tr>
            </thead>
            <tbody>
              {workers.map((worker) => (
                <tr key={worker.worker_id}>
                  <td>
                    <label className="voice-select">
                      <input
                        type="checkbox"
                        aria-label={t("voiceWorkspace.selectWorker", {
                          worker: worker.name,
                        })}
                        checked={selected.includes(worker.worker_id)}
                        onChange={(e) =>
                          onSelected(
                            e.target.checked
                              ? [...selected, worker.worker_id]
                              : selected.filter(
                                  (id) => id !== worker.worker_id,
                                ),
                          )
                        }
                      />
                    </label>
                  </td>
                  <th scope="row">
                    <strong>
                      <Server size={15} aria-hidden />
                      {worker.name}
                    </strong>
                    <span className="voice-connection">
                      <span className={worker.connected ? "online" : ""} />
                      {t(
                        worker.connected
                          ? "voiceWorkspace.online"
                          : "voiceWorkspace.offline",
                      )}
                    </span>
                  </th>
                  {modelKinds.map((kind) => {
                    const cell = worker.models.find((m) => m.kind === kind);
                    if (kind === "separation")
                      return (
                        <td key={kind}>
                          <div className="voice-worker-cell">
                            <span className="text-mute">
                              {t("voiceWorkspace.adapterPending")}
                            </span>
                            <VoiceBadge>
                              {t("voiceWorkspace.coming")}
                            </VoiceBadge>
                          </div>
                        </td>
                      );
                    const absent = !cell || cell.state === "not_installed";
                    return (
                      <td key={kind}>
                        <div className="voice-worker-cell">
                          {absent ? (
                            <>
                              <span className="text-mute">
                                {t("voiceModels.byWorker.absent")}
                              </span>
                              <p>{t("voiceWorkspace.noStartup")}</p>
                              <span className="sr-only">
                                {t("voiceModels.state.not_installed")}
                              </span>
                              {options(kind).length > 0 && (
                                <button
                                  type="button"
                                  className="voice-btn quiet"
                                  disabled={busy}
                                  aria-label={t(
                                    "voiceModels.byWorker.chooseFor",
                                    {
                                      worker: worker.name,
                                      type: kindLabel(kind),
                                    },
                                  )}
                                  onClick={() =>
                                    open({
                                      type: "install",
                                      worker,
                                      kind,
                                      modelId: options(kind)[0].id,
                                    })
                                  }
                                >
                                  {t("voiceModels.byWorker.choosePrompt")}
                                  <Plus size={14} aria-hidden />
                                </button>
                              )}
                            </>
                          ) : (
                            <>
                              <button
                                type="button"
                                className="voice-model-link"
                                onClick={() => onInspect(cell.model_id)}
                              >
                                {cell.label || cell.repo}
                              </button>
                              <div className="voice-inline">
                                <VoiceBadge
                                  tone={
                                    cell.state === "ready"
                                      ? "ok"
                                      : cell.state === "failed"
                                        ? "bad"
                                        : "accent"
                                  }
                                >
                                  {cell.state !== "ready" &&
                                    cell.state !== "failed" && (
                                      <Clock3 size={12} aria-hidden />
                                    )}
                                  {t(`voiceModels.state.${cell.state}`)}
                                </VoiceBadge>
                                {cell.state === "ready" && (
                                  <span className="text-mute">
                                    {t(
                                      catalogue.models.some(
                                        (m) =>
                                          m.id === cell.model_id && m.active,
                                      )
                                        ? "voiceWorkspace.production"
                                        : "voiceWorkspace.reserve",
                                    )}
                                  </span>
                                )}
                              </div>
                              <span className="voice-cell-meta">
                                {cell.revision
                                  ? cell.revision.slice(0, 7)
                                  : t("voiceWorkspace.unknown")}{" "}
                                ·{" "}
                                {t("voiceModels.byWorker.volume", {
                                  size: megabytes(
                                    cell.occupied_bytes || cell.bytes_total,
                                  ),
                                })}
                              </span>
                              {["downloading", "verifying"].includes(
                                cell.state,
                              ) && (
                                <progress
                                  className="voice-progress"
                                  value={progress(cell)}
                                  max={100}
                                  aria-label={t(
                                    `voiceModels.state.${cell.state}`,
                                  )}
                                />
                              )}
                              {cell.state === "downloading" &&
                                cell.bytes_total > 0 && (
                                  <small>
                                    {t("voiceModels.progress", {
                                      received: megabytes(cell.bytes_received),
                                      total: megabytes(cell.bytes_total),
                                    })}
                                  </small>
                                )}
                              {cell.replaces_model_id &&
                                cell.state !== "removing" && (
                                  <p>
                                    {t("voiceModels.byWorker.replacing", {
                                      model: modelName(cell.replaces_model_id),
                                    })}
                                  </p>
                                )}
                              {cell.state === "removing" && (
                                <p>
                                  {cell.replaces_model_id
                                    ? t("voiceModels.byWorker.removingOld", {
                                        model: modelName(
                                          cell.replaces_model_id,
                                        ),
                                      })
                                    : t(
                                        "voiceModels.byWorker.removingStandalone",
                                      )}
                                </p>
                              )}
                              {cell.waiting_for_jobs && (
                                <p>{t("voiceModels.byWorker.waitingJobs")}</p>
                              )}
                              {cell.blocked && (
                                <p role="status" className="voice-error">
                                  {t(`voiceModels.blocked.${cell.blocked}`)}
                                </p>
                              )}
                              {cell.state === "failed" && cell.cause_code && (
                                <p role="alert" className="voice-error">
                                  {t(`voiceModels.cause.${cell.cause_code}`)}
                                </p>
                              )}
                              <div className="voice-cell-actions">
                                {cell.state === "failed" && (
                                  <button
                                    type="button"
                                    className="voice-btn"
                                    disabled={busy}
                                    onClick={() =>
                                      open({
                                        type: "retry",
                                        worker,
                                        kind,
                                        modelId: cell.model_id,
                                      })
                                    }
                                  >
                                    {t("voiceModels.byWorker.retry")}
                                  </button>
                                )}
                                {cell.replaces_model_id &&
                                  ["requested", "failed"].includes(
                                    cell.state,
                                  ) && (
                                    <button
                                      type="button"
                                      className="voice-btn quiet"
                                      disabled={busy}
                                      onClick={() =>
                                        void cancelReplacement(worker, kind)
                                      }
                                    >
                                      {t(
                                        "voiceModels.byWorker.cancelReplacement",
                                      )}
                                    </button>
                                  )}
                                {cell.replaces_model_id &&
                                  cell.state === "failed" &&
                                  cell.cause_code === "disk_full" && (
                                    <button
                                      type="button"
                                      className="voice-btn quiet"
                                      disabled={busy}
                                      onClick={() =>
                                        open({
                                          type: "removeOld",
                                          worker,
                                          kind,
                                          modelId: cell.replaces_model_id,
                                          successor: cell.model_id,
                                        })
                                      }
                                    >
                                      {t("voiceModels.byWorker.removeOld")}
                                    </button>
                                  )}
                                {cell.state === "ready" &&
                                  !cell.replaces_model_id && (
                                    <>
                                      {options(kind, cell.model_id).length >
                                        0 && (
                                        <button
                                          type="button"
                                          className="voice-btn quiet"
                                          disabled={busy}
                                          onClick={() =>
                                            kind === "fingerprint"
                                              ? onTransition(
                                                  options(
                                                    kind,
                                                    cell.model_id,
                                                  )[0].id,
                                                )
                                              : open({
                                                  type: "install",
                                                  worker,
                                                  kind,
                                                  replaces: cell.model_id,
                                                  modelId: options(
                                                    kind,
                                                    cell.model_id,
                                                  )[0].id,
                                                })
                                          }
                                        >
                                          {t("voiceModels.byWorker.replace")}
                                        </button>
                                      )}
                                      <button
                                        type="button"
                                        className="voice-btn quiet"
                                        disabled={busy || cell.waiting_for_jobs}
                                        onClick={() =>
                                          open({
                                            type: "uninstall",
                                            worker,
                                            kind,
                                            modelId: cell.model_id,
                                          })
                                        }
                                      >
                                        {t("voiceModels.byWorker.uninstall")}
                                      </button>
                                    </>
                                  )}
                                {!["ready", "failed"].includes(cell.state) && (
                                  <button
                                    type="button"
                                    className="voice-btn quiet"
                                    onClick={() => onHistory(worker.worker_id)}
                                  >
                                    {t("voiceModels.byWorker.viewOrder")}
                                  </button>
                                )}
                              </div>
                            </>
                          )}
                        </div>
                      </td>
                    );
                  })}
                  <td>
                    <button
                      type="button"
                      className="voice-icon-btn"
                      aria-label={t("voiceModels.byWorker.historyOf", {
                        worker: worker.name,
                      })}
                      onClick={() => onHistory(worker.worker_id)}
                    >
                      <History size={18} aria-hidden />
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
      <ConfirmDialog
        open={pending !== undefined}
        onOpenChange={(next) => !next && setPending(undefined)}
        title={dialog?.title ?? ""}
        cancelLabel={t("voiceModels.byWorker.cancelDialog")}
        confirmLabel={dialog?.confirm ?? ""}
        danger={dialog?.danger}
        busy={busy}
        onConfirm={() => pending && void run(pending)}
      >
        {pending?.type === "install" && installPick.length > 0 && (
          <label className="flex flex-col gap-1 text-sm text-ink">
            {pending.replaces
              ? t("voiceModels.byWorker.replaceWith", {
                  worker: pending.worker.name,
                })
              : t("voiceModels.byWorker.chooseTitle", {
                  worker: pending.worker.name,
                  type: kindLabel(pending.kind),
                })}
            <select
              value={choice}
              onChange={(e) => setChoice(e.target.value)}
              className="h-11 rounded-md border border-line-strong bg-surface px-2 md:h-9"
            >
              {installPick.map((m) => (
                <option key={m.id} value={m.id}>
                  {m.label || m.repo}
                </option>
              ))}
            </select>
          </label>
        )}
        {dialog?.body}
        {failure && <ErrorAlert>{error(failure)}</ErrorAlert>}
      </ConfirmDialog>
    </section>
  );
}
