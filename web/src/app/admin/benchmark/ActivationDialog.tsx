import { useCallback, useState } from "react";
import { Link } from "react-router";
import type { components } from "../../../api.gen";
import { api, errorCode, mutation } from "../../api/client";
import { useLoaded } from "../../api/loaded";
import { ErrorAlert, StatusNotice } from "../../components/Alert";
import { buttonClass } from "../../components/buttons";
import { Modal } from "../../components/Modal";
import { useI18n } from "../../i18n";
import { useSession } from "../../session";
import { settle } from "./api";

type Preview = components["schemas"]["BenchmarkActivationPreview"];
export type ActivationStage = "transcription" | "diarization";

/**
 * Putting a model measured by a benchmark into production. It is never automatic and never a reanalysis: the
 * administrator reads what it changes, what it puts at stake and confirms the model it replaces. Only the work to come
 * uses the new model (ADR 0062).
 */
export function ActivationDialog({
  stage,
  model,
  label,
  runId,
  onClose,
}: {
  stage: ActivationStage;
  model: string;
  label: string;
  runId: number;
  onClose: () => void;
}) {
  const { t, error } = useI18n();
  const { session } = useSession();
  // A stable loader: a new function at each render would reload without end.
  const loadPreview = useCallback(
    () =>
      settle(() =>
        api.GET("/api/admin/benchmarks/activation", {
          params: { query: { stage, model } },
        }),
      ),
    [stage, model],
  );
  const { state, reload } = useLoaded(loadPreview);
  const [understood, setUnderstood] = useState(false);
  const [busy, setBusy] = useState(false);
  const [failure, setFailure] = useState<string>();
  const [done, setDone] =
    useState<components["schemas"]["BenchmarkActivationResult"]>();
  const preview: Preview | undefined =
    state.status === "ready" ? state.value : undefined;
  const noWorker = stage === "diarization" && preview?.workers_ready === 0;
  const canConfirm =
    preview !== undefined &&
    !preview.already_active &&
    understood &&
    !noWorker &&
    !busy &&
    !done;

  const confirm = async () => {
    if (!preview) return;
    setBusy(true);
    setFailure(undefined);
    try {
      const result = await api.POST("/api/admin/benchmarks/activation", {
        params: { header: mutation(session?.csrf_token) },
        body: {
          stage,
          model,
          run_id: runId,
          expected_current: preview.current_model,
          confirm: true,
        },
      });
      if (result.data) setDone(result.data);
      else setFailure(errorCode(result.error) ?? "unknown");
    } catch {
      setFailure("unavailable");
    } finally {
      setBusy(false);
    }
  };

  return (
    <Modal
      open
      onOpenChange={(open) => !open && onClose()}
      title={t("benchmark.activation.title", { model: label })}
      closeLabel={t("benchmark.corpus.close")}
    >
      <div className="flex flex-col gap-3">
        {state.status === "loading" && (
          <p role="status">{t("benchmark.run.loading")}</p>
        )}
        {state.status === "failed" && (
          <div className="flex flex-col items-start gap-2">
            <ErrorAlert>{error(state.code)}</ErrorAlert>
            <button type="button" className={buttonClass()} onClick={reload}>
              {t("benchmark.run.retry")}
            </button>
          </div>
        )}
        {preview && (
          <>
            <dl className="m-0 grid grid-cols-[auto_1fr] gap-x-3 gap-y-1 text-sm">
              <dt className="text-mute">{t("benchmark.activation.current")}</dt>
              <dd className="m-0 font-medium">
                {preview.current_model || t("benchmark.activation.none")}
              </dd>
              <dt className="text-mute">{t("benchmark.activation.target")}</dt>
              <dd className="m-0 font-medium">{preview.target_label}</dd>
            </dl>
            {preview.already_active && (
              <StatusNotice>{t("benchmark.activation.already")}</StatusNotice>
            )}
            <ul className="m-0 list-disc pl-5 text-sm">
              <li>{t("benchmark.activation.newWorkOnly")}</li>
              <li>{t("benchmark.activation.noReanalysis")}</li>
              <li>
                {t("benchmark.activation.humanWork", {
                  clips: preview.clips_with_analysis,
                  human: preview.clips_with_human_work,
                })}
              </li>
            </ul>
            {stage === "diarization" && (
              <p
                role={noWorker ? "alert" : "status"}
                className={`rounded-lg px-3 py-2 text-sm ${noWorker ? "bg-bad-bg text-bad-fg" : "bg-side text-mute"}`}
              >
                {noWorker
                  ? t("benchmark.activation.noWorker")
                  : t("benchmark.activation.workers", {
                      count: preview.workers_ready,
                    })}
              </p>
            )}
            {!done && (
              <label className="flex items-start gap-2 text-sm">
                <input
                  type="checkbox"
                  checked={understood}
                  onChange={(event) => setUnderstood(event.target.checked)}
                  className="mt-0.5 size-4"
                />
                <span>{t("benchmark.activation.understood")}</span>
              </label>
            )}
          </>
        )}
        {failure && <ErrorAlert>{error(failure)}</ErrorAlert>}
        {done && (
          <div className="flex flex-col gap-2">
            <StatusNotice>{t("benchmark.activation.done")}</StatusNotice>
            <p className="text-sm text-mute">
              {t("benchmark.activation.reanalysisLater")}{" "}
              <Link className="text-accent-ink underline" to={done.reanalysis}>
                {t("benchmark.activation.openReanalysis")}
              </Link>
            </p>
          </div>
        )}
        <div className="flex flex-wrap justify-end gap-2">
          <button type="button" className={buttonClass()} onClick={onClose}>
            {done ? t("benchmark.corpus.close") : t("benchmark.corpus.cancel")}
          </button>
          {!done && (
            <button
              type="button"
              className={buttonClass("primary")}
              disabled={!canConfirm}
              onClick={() => void confirm()}
            >
              {busy
                ? t("benchmark.activation.confirming")
                : t("benchmark.activation.confirm")}
            </button>
          )}
        </div>
      </div>
    </Modal>
  );
}
