import { useState } from "react";
import type { components } from "../../../api.gen";
import { api, errorCode, mutation } from "../../api/client";
import { ErrorAlert } from "../../components/Alert";
import { buttonClass } from "../../components/buttons";
import { ConfirmDialog } from "../../components/ConfirmDialog";
import { StatusPill, type PillTone } from "../../components/StatusPill";
import { useI18n } from "../../i18n";
import { useSession } from "../../session";

export type Catalogue = components["schemas"]["VoiceModels"];
type VoiceModel = Catalogue["models"][number];
type WorkerState = VoiceModel["workers"][number];

const workerTones: Record<WorkerState["state"], PillTone> = {
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

/**
 * The model cards: diarization (choose a candidate, watch each Worker prepare it, activate it for
 * every Worker) and fingerprint (read only: its model is fixed by the catalogue).
 */
export function VoiceModelCards({
  catalogue,
  onChanged,
}: {
  catalogue: Catalogue;
  onChanged: () => void;
}) {
  const { t, error } = useI18n();
  const { session } = useSession();
  const [failure, setFailure] = useState<string>();
  const [busy, setBusy] = useState(false);
  const [activating, setActivating] = useState<VoiceModel>();
  const [removing, setRemoving] = useState<{
    model: VoiceModel;
    worker: WorkerState;
  }>();
  const headers = mutation(session?.csrf_token);
  async function run(action: () => Promise<{ error?: unknown }>) {
    setBusy(true);
    setFailure(undefined);
    try {
      const result = await action();
      if (result.error) setFailure(errorCode(result.error) ?? "unknown");
      else onChanged();
    } catch {
      setFailure("unavailable");
    } finally {
      setBusy(false);
      setActivating(undefined);
      setRemoving(undefined);
    }
  }
  const diarization = catalogue.models.filter((m) => m.kind === "diarization");
  const fingerprint = catalogue.models.filter((m) => m.kind === "fingerprint");
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
    <>
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
          {diarization.map((model) => (
            <li
              key={model.id}
              aria-label={model.repo}
              className="flex flex-col gap-2 rounded-lg border border-line p-3"
            >
              <ModelHead model={model} />
              <ModelFacts model={model} />
              <ul className="flex flex-col gap-1.5 text-sm">
                {model.workers.map((worker) => (
                  <li
                    key={worker.worker_id}
                    className="flex flex-wrap items-center gap-x-2 gap-y-1"
                  >
                    <span className="font-medium">{worker.name}</span>
                    <StatusPill tone={workerTones[worker.state]}>
                      {t(`voiceModels.state.${worker.state}`)}
                    </StatusPill>
                    {worker.state === "downloading" &&
                      worker.bytes_total > 0 && (
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
                    {!model.active &&
                      !model.candidate &&
                      (worker.state === "ready" ||
                        worker.state === "failed") && (
                        <button
                          type="button"
                          disabled={busy}
                          onClick={() => setRemoving({ model, worker })}
                          className={buttonClass("secondary")}
                        >
                          {t("voiceModels.remove")}
                        </button>
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
                  </>
                )}
              </div>
            </li>
          ))}
        </ul>
      </section>
      {fingerprint.length > 0 && (
        <section
          aria-labelledby="voice-fingerprint-models"
          className="flex flex-col gap-3 rounded-xl border border-line p-4"
        >
          <h3 id="voice-fingerprint-models" className="font-semibold">
            {t("voiceModels.fingerprintTitle")}
          </h3>
          <p className="text-mute">{t("voiceModels.fingerprintHint")}</p>
          <ul className="flex flex-col gap-3">
            {fingerprint.map((model) => (
              <li
                key={model.id}
                aria-label={model.repo}
                className="flex flex-col gap-2 rounded-lg border border-line p-3"
              >
                <ModelHead model={model} />
                <ModelFacts model={model} />
              </li>
            ))}
          </ul>
        </section>
      )}
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
        title={t("voiceModels.removeTitle", {
          model: removing?.model.repo ?? "",
        })}
        cancelLabel={t("voiceModels.cancelDialog")}
        confirmLabel={t("voiceModels.remove")}
        danger
        busy={busy}
        onConfirm={() =>
          removing &&
          void run(() =>
            api.DELETE("/api/admin/voice-models/{id}/installation", {
              params: {
                path: { id: removing.model.id },
                query: { worker_id: removing.worker.worker_id },
                header: headers,
              },
            }),
          )
        }
      >
        <p>
          {t("voiceModels.removeText", { worker: removing?.worker.name ?? "" })}
        </p>
      </ConfirmDialog>
    </>
  );
}

function ModelHead({ model }: { model: VoiceModel }) {
  const { t } = useI18n();
  return (
    <div className="flex flex-wrap items-center justify-between gap-2">
      <span className="font-mono break-all font-medium">{model.repo}</span>
      <span className="flex flex-wrap gap-2">
        {model.active && (
          <StatusPill tone="ok">{t("voiceModels.active")}</StatusPill>
        )}
        {model.candidate && (
          <StatusPill tone="idle">{t("voiceModels.candidate")}</StatusPill>
        )}
      </span>
    </div>
  );
}

function ModelFacts({ model }: { model: VoiceModel }) {
  const { t } = useI18n();
  return (
    <p className="text-sm text-mute">
      {t("voiceModels.facts", {
        engine: model.engine,
        license: model.license,
        size: megabytes(model.size_bytes),
        revision: model.revision.slice(0, 12),
      })}{" "}
      {model.gated && t("voiceModels.gated")}{" "}
      {model.kind === "diarization" && (
        <>
          {model.max_speakers !== null &&
            t("voiceModels.maxSpeakers", { count: model.max_speakers })}{" "}
          {model.supports_num_speakers
            ? t("voiceModels.numSpeakers")
            : t("voiceModels.noNumSpeakers")}
        </>
      )}
    </p>
  );
}
