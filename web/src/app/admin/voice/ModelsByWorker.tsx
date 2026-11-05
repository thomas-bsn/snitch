import { useEffect, useState } from "react";
import type { components } from "../../../api.gen";
import { api, errorCode, mutation } from "../../api/client";
import { useLoaded } from "../../api/loaded";
import { ErrorAlert } from "../../components/Alert";
import { buttonClass } from "../../components/buttons";
import { ConfirmDialog } from "../../components/ConfirmDialog";
import { StatusPill, type PillTone } from "../../components/StatusPill";
import { useI18n } from "../../i18n";
import { useSession } from "../../session";
import type { Catalogue } from "./VoiceModelCards";

type Cell = components["schemas"]["WorkerModelCell"];
type Row = components["schemas"]["WorkerModelsRow"];
type Kind = Cell["kind"];

const kinds: Kind[] = ["diarization", "fingerprint", "separation"];

const tones: Record<Cell["state"], PillTone> = {
  not_installed: "idle",
  requested: "idle",
  downloading: "idle",
  verifying: "idle",
  ready: "ok",
  switching: "idle",
  removing: "idle",
  failed: "bad",
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

type Pending = {
  worker: Row;
  kind: Kind;
  modelId: string;
  /** The model this one replaces on the Worker: a replacement, not a plain installation. */
  replaces?: string;
  retry?: boolean;
};

/**
 * The minimal "Models per Worker" view (#306): for each Worker, the model of each type with its real
 * state. An Administrator orders an installation explicitly (a confirmation) and retries a failure.
 */
export function ModelsByWorker({ catalogue }: { catalogue: Catalogue }) {
  const { t, error } = useI18n();
  const { session } = useSession();
  const { state, reload } = useLoaded(load);
  const [choice, setChoice] = useState<Record<string, string>>({});
  const [pending, setPending] = useState<Pending>();
  const [busy, setBusy] = useState(false);
  const [failure, setFailure] = useState<string>();
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

  async function order(item: Pending) {
    const { worker, modelId, replaces, retry } = item;
    setBusy(true);
    setFailure(undefined);
    try {
      const body = {
        model_id: modelId,
        idempotency_key: crypto.randomUUID(),
      };
      const result = retry
        ? await api.POST("/api/admin/workers/{id}/voice-models/retry", {
            params: { path: { id: worker.worker_id }, header: headers },
            body,
          })
        : await api.POST("/api/admin/workers/{id}/voice-model-orders", {
            params: { path: { id: worker.worker_id }, header: headers },
            body: replaces ? { ...body, replaces_model_id: replaces } : body,
          });
      if (result.error) setFailure(errorCode(result.error) ?? "unknown");
      else reload();
    } catch {
      setFailure("unavailable");
    } finally {
      setBusy(false);
      setPending(undefined);
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
  const labelOf = (id: string) => {
    const model = catalogue.models.find((m) => m.id === id);
    return model?.label || model?.repo || id;
  };
  const options = (kind: Kind) =>
    catalogue.models.filter((m) => m.kind === kind && m.enabled && m.supported);
  const selected = pending
    ? catalogue.models.find((m) => m.id === pending.modelId)
    : undefined;
  return (
    <section
      aria-labelledby="models-by-worker"
      className="flex flex-col gap-3 rounded-xl border border-line p-4"
    >
      <h3 id="models-by-worker" className="font-semibold">
        {t("voiceModels.byWorker.title")}
      </h3>
      <p className="text-mute">{t("voiceModels.byWorker.hint")}</p>
      {failure && <ErrorAlert>{error(failure)}</ErrorAlert>}
      <div className="overflow-x-auto">
        <table className="w-full text-left text-sm">
          <thead>
            <tr>
              <th scope="col" className="py-1 pr-3 font-medium">
                {t("voiceModels.byWorker.worker")}
              </th>
              {kinds.map((kind) => (
                <th key={kind} scope="col" className="py-1 pr-3 font-medium">
                  {t(`voiceModels.byWorker.type.${kind}`)}
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
                <th scope="row" className="py-2 pr-3 font-medium">
                  {worker.name}
                </th>
                {kinds.map((kind) => {
                  const cell = worker.models.find((m) => m.kind === kind);
                  const key = `${worker.worker_id}/${kind}`;
                  const candidates = options(kind);
                  const picked = choice[key] ?? candidates[0]?.id ?? "";
                  return (
                    <td key={kind} className="py-2 pr-3">
                      {cell && (
                        <div className="flex flex-col gap-1">
                          <StatusPill tone={tones[cell.state]}>
                            {t(`voiceModels.state.${cell.state}`)}
                          </StatusPill>
                          {cell.state === "not_installed" ? (
                            !cell.supported ? (
                              <span className="text-mute">
                                {t(
                                  "voiceModels.byWorker.separationUnsupported",
                                )}
                              </span>
                            ) : (
                              <>
                                <span className="text-mute">
                                  {t("voiceModels.byWorker.absent")}
                                </span>
                                {candidates.length > 0 && (
                                  <div className="flex flex-wrap items-center gap-2">
                                    <select
                                      aria-label={t(
                                        "voiceModels.byWorker.choose",
                                        {
                                          worker: worker.name,
                                          type: t(
                                            `voiceModels.byWorker.type.${kind}`,
                                          ),
                                        },
                                      )}
                                      value={picked}
                                      onChange={(e) =>
                                        setChoice({
                                          ...choice,
                                          [key]: e.target.value,
                                        })
                                      }
                                      className="rounded-md border border-line bg-surface px-2 py-1"
                                    >
                                      {candidates.map((m) => (
                                        <option key={m.id} value={m.id}>
                                          {m.label || m.repo}
                                        </option>
                                      ))}
                                    </select>
                                    <button
                                      type="button"
                                      disabled={busy || picked === ""}
                                      onClick={() =>
                                        setPending({
                                          worker,
                                          kind,
                                          modelId: picked,
                                        })
                                      }
                                      className={buttonClass("secondary")}
                                    >
                                      {t("voiceModels.byWorker.install")}
                                    </button>
                                  </div>
                                )}
                              </>
                            )
                          ) : (
                            <>
                              <span className="font-medium">
                                {cell.label || cell.repo}
                              </span>
                              {cell.state === "downloading" &&
                                cell.bytes_total > 0 && (
                                  <span className="tabular-nums text-mute">
                                    {t("voiceModels.progress", {
                                      received: megabytes(cell.bytes_received),
                                      total: megabytes(cell.bytes_total),
                                    })}
                                  </span>
                                )}
                              {cell.occupied_bytes > 0 && (
                                <span className="tabular-nums text-mute">
                                  {t("voiceModels.byWorker.volume", {
                                    size: megabytes(cell.occupied_bytes),
                                  })}
                                </span>
                              )}
                              {cell.revision !== "" && (
                                <span className="font-mono text-xs text-mute">
                                  {t("voiceModels.byWorker.revision", {
                                    revision: cell.revision.slice(0, 7),
                                  })}
                                </span>
                              )}
                              {cell.state === "failed" && (
                                <>
                                  <span className="text-bad-fg">
                                    {cell.cause_code !== "" &&
                                      t(`voiceModels.cause.${cell.cause_code}`)}
                                  </span>
                                  <button
                                    type="button"
                                    disabled={busy}
                                    onClick={() =>
                                      setPending({
                                        worker,
                                        kind,
                                        modelId: cell.model_id,
                                        retry: true,
                                      })
                                    }
                                    className={buttonClass("secondary")}
                                  >
                                    {t("voiceModels.byWorker.retry")}
                                  </button>
                                </>
                              )}
                              {cell.replaces_model_id !== "" && (
                                <span className="text-mute">
                                  {t("voiceModels.byWorker.replacing", {
                                    model: labelOf(cell.replaces_model_id),
                                  })}
                                </span>
                              )}
                              {cell.state === "switching" &&
                                cell.waiting_for_jobs && (
                                  <span className="text-mute">
                                    {t("voiceModels.byWorker.waitingJobs")}
                                  </span>
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
                              {kind === "diarization" &&
                                cell.state === "ready" &&
                                cell.replaces_model_id === "" &&
                                candidates.some(
                                  (m) => m.id !== cell.model_id,
                                ) && (
                                  <div className="flex flex-wrap items-center gap-2">
                                    <select
                                      aria-label={t(
                                        "voiceModels.byWorker.replaceWith",
                                        { worker: worker.name },
                                      )}
                                      value={
                                        choice[key] &&
                                        choice[key] !== cell.model_id
                                          ? choice[key]
                                          : (candidates.find(
                                              (m) => m.id !== cell.model_id,
                                            )?.id ?? "")
                                      }
                                      onChange={(e) =>
                                        setChoice({
                                          ...choice,
                                          [key]: e.target.value,
                                        })
                                      }
                                      className="rounded-md border border-line bg-surface px-2 py-1"
                                    >
                                      {candidates
                                        .filter((m) => m.id !== cell.model_id)
                                        .map((m) => (
                                          <option key={m.id} value={m.id}>
                                            {m.label || m.repo}
                                          </option>
                                        ))}
                                    </select>
                                    <button
                                      type="button"
                                      disabled={busy}
                                      onClick={() =>
                                        setPending({
                                          worker,
                                          kind,
                                          replaces: cell.model_id,
                                          modelId:
                                            choice[key] &&
                                            choice[key] !== cell.model_id
                                              ? choice[key]
                                              : (candidates.find(
                                                  (m) => m.id !== cell.model_id,
                                                )?.id ?? ""),
                                        })
                                      }
                                      className={buttonClass("secondary")}
                                    >
                                      {t("voiceModels.byWorker.replace")}
                                    </button>
                                  </div>
                                )}
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
        onOpenChange={(open) => !open && setPending(undefined)}
        title={t("voiceModels.byWorker.confirmTitle", {
          model: selected?.label || selected?.repo || "",
          worker: pending?.worker.name ?? "",
        })}
        cancelLabel={t("voiceModels.byWorker.cancelDialog")}
        confirmLabel={t("voiceModels.byWorker.confirmAction")}
        busy={busy}
        onConfirm={() => pending && void order(pending)}
      >
        <p>
          {t(
            pending?.replaces
              ? "voiceModels.byWorker.confirmReplaceText"
              : "voiceModels.byWorker.confirmText",
            {
              model: selected?.label || selected?.repo || "",
              worker: pending?.worker.name ?? "",
              old: pending?.replaces ? labelOf(pending.replaces) : "",
            },
          )}
        </p>
        {selected?.gated && <p>{t("voiceModels.byWorker.confirmGated")}</p>}
      </ConfirmDialog>
    </section>
  );
}
