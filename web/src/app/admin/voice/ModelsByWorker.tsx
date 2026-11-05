import { useEffect, useState } from "react";
import {
  ArrowLeftRight,
  CircleDot,
  Download,
  ShieldCheck,
  Trash2,
  type LucideIcon,
} from "lucide-react";
import type { components } from "../../../api.gen";
import { api, errorCode, mutation } from "../../api/client";
import { useLoaded } from "../../api/loaded";
import { ErrorAlert } from "../../components/Alert";
import { buttonClass } from "../../components/buttons";
import { ConfirmDialog } from "../../components/ConfirmDialog";
import { StatusPill, type PillTone } from "../../components/StatusPill";
import { useI18n } from "../../i18n";
import { useSession } from "../../session";
import { ApplyToWorkers } from "./ApplyToWorkers";
import { OrderHistory } from "./OrderHistory";
import type { Catalogue } from "./VoiceModelCards";
import { VoiceModelAdd } from "./VoiceModelAdd";

type Cell = components["schemas"]["WorkerModelCell"];
export type Row = components["schemas"]["WorkerModelsRow"];
export type Kind = Cell["kind"];

/** Separation is folded into a banner until an adapter exists (#268): two columns. */
const kinds: Kind[] = ["diarization", "fingerprint"];
const states: Cell["state"][] = [
  "not_installed",
  "requested",
  "downloading",
  "verifying",
  "ready",
  "switching",
  "removing",
  "failed",
];

const look: Record<Cell["state"], { tone: PillTone; icon?: LucideIcon }> = {
  not_installed: { tone: "idle" },
  requested: { tone: "busy", icon: CircleDot },
  downloading: { tone: "busy", icon: Download },
  verifying: { tone: "busy", icon: ShieldCheck },
  ready: { tone: "ok" },
  switching: { tone: "busy", icon: ArrowLeftRight },
  removing: { tone: "busy", icon: Trash2 },
  failed: { tone: "bad" },
};

const megabytes = (bytes: number) => Math.round(bytes / 1_000_000);

async function load() {
  try {
    const result = await api.GET("/api/admin/voice-models/workers");
    if (result.data) return { ok: true as const, value: result.data };
    return { ok: false as const, code: errorCode(result.error) ?? "unknown" };
  } catch {
    return { ok: false as const, code: "unavailable" };
  }
}

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
  canAdd,
  onCatalogueChanged,
}: {
  catalogue: Catalogue;
  canAdd: boolean;
  onCatalogueChanged: () => void;
}) {
  const { t, error } = useI18n();
  const { session } = useSession();
  const { state, reload } = useLoaded(load);
  const [choice, setChoice] = useState<string>("");
  const [pending, setPending] = useState<Pending>();
  const [busy, setBusy] = useState(false);
  const [failure, setFailure] = useState<string>();
  const [history, setHistory] = useState<{ workerId?: string } | undefined>();
  const [applying, setApplying] = useState(false);
  const headers = mutation(session?.csrf_token);
  const active =
    state.status === "ready" &&
    state.value.workers.some((w) =>
      w.models.some((m) =>
        [
          "requested",
          "downloading",
          "verifying",
          "switching",
          "removing",
        ].includes(m.state),
      ),
    );
  useEffect(() => {
    if (!active) return;
    const timer = window.setInterval(reload, 3000);
    return () => window.clearInterval(timer);
  }, [active, reload]);
  if (state.status === "loading")
    return <p role="status">{t("app.loading")}</p>;
  if (state.status === "failed")
    return <ErrorAlert>{error(state.code)}</ErrorAlert>;

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
    setFailure(undefined);
    setChoice(next.type === "install" ? next.modelId : "");
    setPending(next);
  }

  async function run(item: Pending) {
    setBusy(true);
    setFailure(undefined);
    try {
      const id = { path: { id: item.worker.worker_id }, header: headers };
      const key = crypto.randomUUID();
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
    <section
      aria-labelledby="models-by-worker"
      className="flex flex-col gap-4 rounded-xl border border-line bg-surface p-4"
    >
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="min-w-0 max-w-md flex-1">
          <h3 id="models-by-worker" className="text-lg font-semibold">
            {t("voiceModels.byWorker.title")}
          </h3>
          <p className="text-mute">{t("voiceModels.byWorker.hint")}</p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <button
            type="button"
            className={buttonClass("secondary")}
            onClick={() => setHistory({})}
          >
            {t("voiceModels.byWorker.history")}
          </button>
          <button
            type="button"
            className={buttonClass("secondary")}
            onClick={() => setApplying(true)}
          >
            {t("voiceModels.byWorker.applyMany")}
          </button>
          {canAdd && <VoiceModelAdd onChanged={onCatalogueChanged} />}
        </div>
      </div>
      <ul
        aria-label={t("voiceModels.byWorker.statuses")}
        className="flex flex-wrap gap-2"
      >
        {states.map((s) => (
          <li key={s}>
            <StatusPill tone={look[s].tone} icon={look[s].icon}>
              {t(`voiceModels.state.${s}`)}
            </StatusPill>
          </li>
        ))}
      </ul>
      <p className="rounded-lg bg-accent-soft px-3 py-2 text-sm text-accent-ink">
        {t("voiceModels.byWorker.separationBanner")}
      </p>
      {failure && <ErrorAlert>{error(failure)}</ErrorAlert>}
      <div
        role="region"
        tabIndex={0}
        aria-label={t("voiceModels.byWorker.tableLabel")}
        className="overflow-x-auto rounded-lg border border-line"
      >
        <table className="w-full min-w-[40rem] text-left text-sm">
          <thead>
            <tr className="text-xs uppercase tracking-wide text-mute">
              <th scope="col" className="px-3 py-2.5 font-medium">
                {t("voiceModels.byWorker.worker")}
              </th>
              {kinds.map((kind) => (
                <th key={kind} scope="col" className="px-3 py-2.5 font-medium">
                  {kindLabel(kind)}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {state.value.workers.map((worker) => (
              <tr
                key={worker.worker_id}
                className="border-t border-line align-top"
              >
                <th scope="row" className="px-3 py-3 font-medium">
                  <div>{worker.name}</div>
                  <button
                    type="button"
                    onClick={() => setHistory({ workerId: worker.worker_id })}
                    aria-label={t("voiceModels.byWorker.historyOf", {
                      worker: worker.name,
                    })}
                    className="mt-1 text-xs font-normal text-accent-ink underline underline-offset-2"
                  >
                    {t("voiceModels.byWorker.history")}
                  </button>
                </th>
                {kinds.map((kind) => {
                  const cell = worker.models.find((m) => m.kind === kind);
                  const replaceable =
                    kind === "diarization" &&
                    cell !== undefined &&
                    options(kind, cell.model_id).length > 0;
                  return (
                    <td key={kind} className="px-3 py-3">
                      {cell && (
                        <div className="flex flex-col gap-1.5">
                          {cell.state === "not_installed" ? (
                            <>
                              <span className="text-mute">
                                {t("voiceModels.byWorker.absent")}
                              </span>
                              <div>
                                <StatusPill tone="idle">
                                  {t("voiceModels.state.not_installed")}
                                </StatusPill>
                              </div>
                              {cell.supported && options(kind).length > 0 && (
                                <div>
                                  <button
                                    type="button"
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
                                    className={buttonClass("primary")}
                                  >
                                    {t("voiceModels.byWorker.choosePrompt")}
                                  </button>
                                </div>
                              )}
                            </>
                          ) : (
                            <>
                              <span className="font-medium">
                                {cell.label || cell.repo}
                              </span>
                              <span className="font-mono text-xs text-mute tabular-nums">
                                {[
                                  cell.revision !== "" &&
                                    t("voiceModels.byWorker.revision", {
                                      revision: cell.revision.slice(0, 7),
                                    }),
                                  (cell.occupied_bytes > 0 ||
                                    cell.bytes_total > 0) &&
                                    t("voiceModels.byWorker.volume", {
                                      size: megabytes(
                                        cell.occupied_bytes > 0
                                          ? cell.occupied_bytes
                                          : cell.bytes_total,
                                      ),
                                    }),
                                ]
                                  .filter(Boolean)
                                  .join(" · ")}
                              </span>
                              <div>
                                <StatusPill
                                  tone={look[cell.state].tone}
                                  icon={look[cell.state].icon}
                                >
                                  {t(`voiceModels.state.${cell.state}`)}
                                </StatusPill>
                              </div>
                              {[
                                "requested",
                                "downloading",
                                "verifying",
                                "switching",
                                "removing",
                              ].includes(cell.state) && (
                                <div
                                  role="progressbar"
                                  aria-label={t(
                                    `voiceModels.state.${cell.state}`,
                                  )}
                                  aria-valuemin={0}
                                  aria-valuemax={100}
                                  aria-valuenow={Math.round(progress(cell))}
                                  className="h-1.5 w-full overflow-hidden rounded-full bg-side"
                                >
                                  <div
                                    className="h-full rounded-full bg-accent-solid"
                                    style={{ width: `${progress(cell)}%` }}
                                  />
                                </div>
                              )}
                              {cell.state === "downloading" &&
                                cell.bytes_total > 0 && (
                                  <span className="tabular-nums text-mute">
                                    {t("voiceModels.progress", {
                                      received: megabytes(cell.bytes_received),
                                      total: megabytes(cell.bytes_total),
                                    })}
                                  </span>
                                )}
                              {cell.replaces_model_id !== "" &&
                                cell.state !== "removing" && (
                                  <span className="text-mute">
                                    {t("voiceModels.byWorker.replacing", {
                                      model: modelName(cell.replaces_model_id),
                                    })}
                                  </span>
                                )}
                              {cell.state === "removing" && (
                                <span className="text-mute">
                                  {cell.replaces_model_id !== ""
                                    ? t("voiceModels.byWorker.removingOld", {
                                        model: modelName(
                                          cell.replaces_model_id,
                                        ),
                                      })
                                    : t(
                                        "voiceModels.byWorker.removingStandalone",
                                      )}
                                </span>
                              )}
                              {cell.state === "switching" &&
                                cell.waiting_for_jobs && (
                                  <span className="text-mute">
                                    {t("voiceModels.byWorker.waitingJobs")}
                                  </span>
                                )}
                              {cell.state === "failed" &&
                                cell.cause_code !== "" && (
                                  <p
                                    role="alert"
                                    className="rounded-md bg-bad-bg px-2.5 py-1.5 text-bad-fg"
                                  >
                                    {t(`voiceModels.cause.${cell.cause_code}`)}
                                  </p>
                                )}
                              <div className="flex flex-wrap gap-2">
                                {cell.state === "failed" && (
                                  <button
                                    type="button"
                                    disabled={busy}
                                    onClick={() =>
                                      open({
                                        type: "retry",
                                        worker,
                                        kind,
                                        modelId: cell.model_id,
                                      })
                                    }
                                    className={buttonClass("primary")}
                                  >
                                    {t("voiceModels.byWorker.retry")}
                                  </button>
                                )}
                                {cell.replaces_model_id !== "" &&
                                  (cell.state === "requested" ||
                                    cell.state === "failed") && (
                                    <button
                                      type="button"
                                      disabled={busy}
                                      onClick={() =>
                                        void cancelReplacement(worker, kind)
                                      }
                                      className={buttonClass("secondary")}
                                    >
                                      {t(
                                        "voiceModels.byWorker.cancelReplacement",
                                      )}
                                    </button>
                                  )}
                                {cell.replaces_model_id !== "" &&
                                  cell.state === "failed" &&
                                  cell.cause_code === "disk_full" && (
                                    <button
                                      type="button"
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
                                      className={buttonClass("danger")}
                                    >
                                      {t("voiceModels.byWorker.removeOld")}
                                    </button>
                                  )}
                                {cell.state === "ready" &&
                                  cell.replaces_model_id === "" && (
                                    <>
                                      {replaceable && (
                                        <button
                                          type="button"
                                          disabled={busy}
                                          onClick={() =>
                                            open({
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
                                          className={buttonClass("secondary")}
                                        >
                                          {t("voiceModels.byWorker.replace")}
                                        </button>
                                      )}
                                      <button
                                        type="button"
                                        disabled={busy}
                                        onClick={() =>
                                          open({
                                            type: "uninstall",
                                            worker,
                                            kind,
                                            modelId: cell.model_id,
                                          })
                                        }
                                        className={buttonClass("secondary")}
                                      >
                                        {t("voiceModels.byWorker.uninstall")}
                                      </button>
                                    </>
                                  )}
                                {cell.state !== "ready" &&
                                  cell.state !== "failed" && (
                                    <button
                                      type="button"
                                      onClick={() =>
                                        setHistory({
                                          workerId: worker.worker_id,
                                        })
                                      }
                                      className={buttonClass("secondary")}
                                    >
                                      {t("voiceModels.byWorker.viewOrder")}
                                    </button>
                                  )}
                              </div>
                            </>
                          )}
                        </div>
                      )}
                    </td>
                  );
                })}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
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
      <OrderHistory
        open={history !== undefined}
        onClose={() => setHistory(undefined)}
        workers={state.value.workers}
        initialWorkerId={history?.workerId}
      />
      <ApplyToWorkers
        open={applying}
        onClose={() => {
          setApplying(false);
          reload();
        }}
        workers={state.value.workers}
        catalogue={catalogue}
        kinds={kinds}
      />
    </section>
  );
}
