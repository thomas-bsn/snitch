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

type Overview = components["schemas"]["VoiceOverview"];
type Row = Overview["workers"][number];
type Model = Row["diarization"];

async function load() {
  try {
    const result = await api.GET("/api/admin/voice-overview");
    if (result.data) return { ok: true as const, value: result.data };
    return { ok: false as const, code: errorCode(result.error) ?? "unknown" };
  } catch {
    return { ok: false as const, code: "unavailable" };
  }
}

const tones: Record<Row["voice_state"], PillTone> = {
  ready: "ok",
  untested: "idle",
  unavailable: "bad",
  models_missing: "idle",
};

/** What each Worker really reports about the voice models: never typed by an administrator. */
export function VoiceInventory() {
  const { t, error } = useI18n();
  const { state, reload } = useLoaded(load);
  if (state.status === "loading")
    return <p role="status">{t("app.loading")}</p>;
  if (state.status === "failed")
    return <ErrorAlert>{error(state.code)}</ErrorAlert>;
  const overview = state.value;
  const model = (value: Model) =>
    value
      ? `${value.model}${value.version ? ` @ ${value.version}` : ""}`
      : t("voiceInventory.none");
  return (
    <>
      <VoiceModelManager onChanged={reload} />
      <section
        aria-labelledby="voice-inventory"
        className="flex flex-col gap-3 rounded-xl border border-line p-4"
      >
        <h3 id="voice-inventory" className="font-semibold">
          {t("voiceInventory.title")}
        </h3>
        <p className="text-mute">{t("voiceInventory.hint")}</p>
        <div className="flex flex-wrap gap-x-8 gap-y-2">
          <p>
            <span className="text-mute">{t("voiceInventory.ready")} </span>
            <strong className="tabular-nums">
              {overview.workers_voice_ready}/{overview.workers_total}
            </strong>
          </p>
          <p>
            <span className="text-mute">{t("voiceInventory.pending")} </span>
            <strong className="tabular-nums">
              {overview.pending_voice_jobs}
            </strong>
          </p>
        </div>
        {overview.workers_total === 0 ? (
          <p className="text-mute">{t("voiceInventory.empty")}</p>
        ) : (
          <ul className="flex flex-col divide-y divide-line rounded-lg border border-line">
            {overview.workers.map((worker) => (
              <li key={worker.id} className="flex flex-col gap-1.5 px-3 py-2.5">
                <div className="flex flex-wrap items-center justify-between gap-2">
                  <span className="font-medium">{worker.name}</span>
                  <span className="flex flex-wrap gap-2">
                    {!worker.connected && (
                      <StatusPill tone="idle">
                        {t("voiceInventory.offline")}
                      </StatusPill>
                    )}
                    <StatusPill tone={tones[worker.voice_state]}>
                      {t(`voiceInventory.state.${worker.voice_state}`)}
                    </StatusPill>
                  </span>
                </div>
                <dl className="grid gap-x-4 gap-y-1 text-sm sm:grid-cols-[10rem_1fr]">
                  <dt className="text-mute">
                    {t("voiceInventory.diarization")}
                  </dt>
                  <dd className="font-mono break-all">
                    {model(worker.diarization)}
                  </dd>
                  <dt className="text-mute">
                    {t("voiceInventory.fingerprint")}
                  </dt>
                  <dd className="font-mono break-all">
                    {model(worker.fingerprint)}
                  </dd>
                </dl>
                {worker.voice_state === "models_missing" && (
                  <p className="text-sm text-mute">
                    {t("voiceInventory.missing", {
                      model: overview.active_models.diarization,
                    })}
                  </p>
                )}
                {worker.cause_code && (
                  <p className="text-sm text-bad-fg">
                    {t("voiceInventory.cause", { code: worker.cause_code })}
                  </p>
                )}
              </li>
            ))}
          </ul>
        )}
      </section>
    </>
  );
}

type Catalogue = components["schemas"]["VoiceModels"];
type VoiceModel = Catalogue["models"][number];
type WorkerState = VoiceModel["workers"][number];

async function loadModels() {
  try {
    const result = await api.GET("/api/admin/voice-models");
    if (result.data) return { ok: true as const, value: result.data };
    return { ok: false as const, code: errorCode(result.error) ?? "unknown" };
  } catch {
    return { ok: false as const, code: "unavailable" };
  }
}

const workerTones: Record<WorkerState["state"], PillTone> = {
  not_installed: "idle",
  downloading: "idle",
  verifying: "idle",
  ready: "ok",
  failed: "bad",
};

const megabytes = (bytes: number) => Math.round(bytes / 1_000_000);

/**
 * The diarization models of the catalogue: choose a candidate, watch each Worker prepare it, then
 * activate it for every Worker. Nothing is typed: the catalogue lives in the code.
 */
function VoiceModelManager({ onChanged }: { onChanged: () => void }) {
  const { t, error } = useI18n();
  const { session } = useSession();
  const { state, reload } = useLoaded(loadModels);
  const [failure, setFailure] = useState<string>();
  const [busy, setBusy] = useState(false);
  const [activating, setActivating] = useState<VoiceModel>();
  const [removing, setRemoving] = useState<VoiceModel>();
  const headers = mutation(session?.csrf_token);
  const preparing =
    state.status === "ready" &&
    state.value.models.some((m) =>
      m.workers.some(
        (w) => w.state === "downloading" || w.state === "verifying",
      ),
    );
  useEffect(() => {
    if (!preparing) return;
    const timer = window.setInterval(reload, 3000);
    return () => window.clearInterval(timer);
  }, [preparing, reload]);
  async function run(action: () => Promise<{ error?: unknown }>) {
    setBusy(true);
    setFailure(undefined);
    try {
      const result = await action();
      if (result.error) setFailure(errorCode(result.error) ?? "unknown");
      else {
        reload();
        onChanged();
      }
    } catch {
      setFailure("unavailable");
    } finally {
      setBusy(false);
      setActivating(undefined);
      setRemoving(undefined);
    }
  }
  if (state.status === "loading")
    return <p role="status">{t("app.loading")}</p>;
  if (state.status === "failed")
    return <ErrorAlert>{error(state.code)}</ErrorAlert>;
  const models = state.value.models.filter((m) => m.kind === "diarization");
  const choose = (m: VoiceModel) =>
    run(() =>
      api.PUT("/api/admin/voice-models/candidate", {
        params: { header: headers },
        body: { model_id: m.id },
      }),
    );
  const retry = (worker: WorkerState) =>
    run(() =>
      api.POST("/api/admin/workers/{id}/voice-models/retry", {
        params: { path: { id: worker.worker_id }, header: headers },
      }),
    );
  return (
    <section
      aria-labelledby="voice-models"
      className="flex flex-col gap-3 rounded-xl border border-line p-4"
    >
      <h3 id="voice-models" className="font-semibold">
        {t("voiceModels.title")}
      </h3>
      <p className="text-mute">{t("voiceModels.hint")}</p>
      {failure && <ErrorAlert>{error(failure)}</ErrorAlert>}
      <ul className="flex flex-col gap-3">
        {models.map((model) => (
          <li
            key={model.id}
            aria-label={model.repo}
            className="flex flex-col gap-2 rounded-lg border border-line p-3"
          >
            <div className="flex flex-wrap items-center justify-between gap-2">
              <span className="font-mono break-all font-medium">
                {model.repo}
              </span>
              <span className="flex flex-wrap gap-2">
                {model.active && (
                  <StatusPill tone="ok">{t("voiceModels.active")}</StatusPill>
                )}
                {model.candidate && (
                  <StatusPill tone="idle">
                    {t("voiceModels.candidate")}
                  </StatusPill>
                )}
              </span>
            </div>
            <p className="text-sm text-mute">
              {t("voiceModels.facts", {
                engine: model.engine,
                license: model.license,
                size: megabytes(model.size_bytes),
                revision: model.revision.slice(0, 12),
              })}{" "}
              {model.gated && t("voiceModels.gated")}{" "}
              {model.max_speakers !== null &&
                t("voiceModels.maxSpeakers", {
                  count: model.max_speakers,
                })}{" "}
              {model.supports_num_speakers
                ? t("voiceModels.numSpeakers")
                : t("voiceModels.noNumSpeakers")}
            </p>
            <ul className="flex flex-col gap-1 text-sm">
              {model.workers.map((worker) => (
                <li
                  key={worker.worker_id}
                  className="flex flex-wrap items-center gap-2"
                >
                  <span className="font-medium">{worker.name}</span>
                  <StatusPill tone={workerTones[worker.state]}>
                    {t(`voiceModels.state.${worker.state}`)}
                  </StatusPill>
                  {worker.state === "downloading" && worker.bytes_total > 0 && (
                    <span className="tabular-nums text-mute">
                      {t("voiceModels.progress", {
                        received: megabytes(worker.bytes_received),
                        total: megabytes(worker.bytes_total),
                      })}
                    </span>
                  )}
                  {worker.state === "failed" && (
                    <>
                      <span className="text-bad-fg">
                        {worker.cause_code !== "" &&
                          t(`voiceModels.cause.${worker.cause_code}`)}
                      </span>
                      {model.candidate && (
                        <button
                          type="button"
                          disabled={busy}
                          onClick={() => void retry(worker)}
                          className={buttonClass("secondary")}
                        >
                          {t("voiceModels.retry")}
                        </button>
                      )}
                    </>
                  )}
                </li>
              ))}
            </ul>
            <div className="flex flex-wrap gap-2">
              {model.candidate && (
                <>
                  <button
                    type="button"
                    disabled={busy || !model.installed}
                    onClick={() => setActivating(model)}
                    className={buttonClass("primary")}
                  >
                    {t("voiceModels.activate")}
                  </button>
                  <button
                    type="button"
                    disabled={busy}
                    onClick={() =>
                      void run(() =>
                        api.DELETE("/api/admin/voice-models/candidate", {
                          params: { header: headers },
                        }),
                      )
                    }
                    className={buttonClass("secondary")}
                  >
                    {t("voiceModels.cancel")}
                  </button>
                </>
              )}
              {!model.active && !model.candidate && !model.installed && (
                <button
                  type="button"
                  disabled={busy}
                  onClick={() => void choose(model)}
                  className={buttonClass("primary")}
                >
                  {t("voiceModels.choose")}
                </button>
              )}
              {!model.active && !model.candidate && model.installed && (
                <>
                  <button
                    type="button"
                    disabled={busy}
                    onClick={() => setActivating(model)}
                    className={buttonClass("primary")}
                  >
                    {t("voiceModels.revert")}
                  </button>
                  <button
                    type="button"
                    disabled={busy}
                    onClick={() => setRemoving(model)}
                    className={buttonClass("secondary")}
                  >
                    {t("voiceModels.remove")}
                  </button>
                </>
              )}
            </div>
          </li>
        ))}
      </ul>
      <ConfirmDialog
        open={activating !== undefined}
        onOpenChange={(open) => !open && setActivating(undefined)}
        title={t("voiceModels.confirmTitle", { model: activating?.repo ?? "" })}
        cancelLabel={t("voiceModels.cancelDialog")}
        confirmLabel={t("voiceModels.confirmActivate")}
        busy={busy}
        onConfirm={() =>
          activating &&
          void run(() =>
            api.POST("/api/admin/voice-models/{id}/activate", {
              params: { path: { id: activating.id }, header: headers },
            }),
          )
        }
      >
        <p>{t("voiceModels.effectVersion")}</p>
        <p>{t("voiceModels.effectRunning")}</p>
        <p>{t("voiceModels.effectClips")}</p>
        {activating && (
          <>
            <p>
              {t("voiceModels.effectReady", {
                names:
                  activating.workers
                    .filter((w) => w.state === "ready")
                    .map((w) => w.name)
                    .join(", ") || "-",
              })}
            </p>
            {activating.workers.some((w) => w.state !== "ready") && (
              <p className="font-medium text-ink">
                {t("voiceModels.effectNotReady", {
                  names: activating.workers
                    .filter((w) => w.state !== "ready")
                    .map((w) => w.name)
                    .join(", "),
                })}
              </p>
            )}
          </>
        )}
      </ConfirmDialog>
      <ConfirmDialog
        open={removing !== undefined}
        onOpenChange={(open) => !open && setRemoving(undefined)}
        title={t("voiceModels.removeTitle", { model: removing?.repo ?? "" })}
        cancelLabel={t("voiceModels.cancelDialog")}
        confirmLabel={t("voiceModels.remove")}
        danger
        busy={busy}
        onConfirm={() =>
          removing &&
          void run(() =>
            api.DELETE("/api/admin/voice-models/{id}/installation", {
              params: { path: { id: removing.id }, header: headers },
            }),
          )
        }
      >
        <p>{t("voiceModels.removeText")}</p>
      </ConfirmDialog>
    </section>
  );
}
