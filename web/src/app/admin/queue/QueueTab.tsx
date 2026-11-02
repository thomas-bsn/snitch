import { Monitor } from "lucide-react";
import { JobWorkflow, type JobRef } from "./JobWorkflow";
import { usePolling } from "./usePolling";
import { QueueControlBar } from "./QueueControlBar";
import { ScheduleTaskForm, type UnitStep } from "./ScheduleTaskForm";
import { useState } from "react";
import { api, errorCode, mutation } from "../../api/client";
import { useLoaded } from "../../api/loaded";
import { ErrorAlert, StatusNotice } from "../../components/Alert";
import { buttonClass } from "../../components/buttons";
import { ConfirmDialog } from "../../components/ConfirmDialog";
import { useI18n } from "../../i18n";
import { useSession } from "../../session";
import type { components } from "../../../api.gen";

async function load() {
  try {
    const [controls, cards, workers] = await Promise.all([
      api.GET("/api/admin/job-controls"),
      api.GET("/api/admin/task-cards"),
      api.GET("/api/admin/workers"),
    ]);
    if (!controls.data || !cards.data || !workers.data)
      return {
        ok: false as const,
        code: errorCode(controls.error ?? cards.error) ?? "unknown",
      };
    return {
      ok: true as const,
      value: {
        controls: controls.data,
        workers: workers.data.items,
        board: cards.data,
      },
    };
  } catch {
    return { ok: false as const, code: "unavailable" };
  }
}
type Job = components["schemas"]["Job"];
type Event = components["schemas"]["JobEvent"];
export function QueueTab() {
  const { t, error, language } = useI18n();
  const { session } = useSession();
  const { state, reload } = useLoaded(load);
  usePolling(reload);
  const [failure, setFailure] = useState<string>();
  const [notice, setNotice] = useState(false);
  const [history, setHistory] = useState<Record<number, Event[]>>({});
  const [busy, setBusy] = useState(false);
  const [transcriptionClip, setTranscriptionClip] = useState<number>();
  const [warning, setWarning] = useState<{
    job?: JobRef;
    input?: components["schemas"]["JobInput"];
    corrections: components["schemas"]["CorrectionsToReplace"];
  }>();
  async function schedule(
    input: components["schemas"]["JobInput"],
    confirm = false,
  ) {
    setBusy(true);
    setFailure(undefined);
    try {
      const result = await api.POST("/api/admin/jobs", {
        params: {
          header: mutation(session?.csrf_token),
          query: { confirm_replace_corrections: confirm || undefined },
        },
        body: input,
      });
      if (
        result.error &&
        "corrections" in result.error &&
        result.error.corrections &&
        result.error.error === "corrections_would_be_replaced"
      )
        setWarning({ input, corrections: result.error.corrections });
      else if (result.error) setFailure(errorCode(result.error) ?? "unknown");
      else {
        setWarning(undefined);
        setNotice(true);
        reload();
      }
    } catch {
      setFailure("unavailable");
    } finally {
      setBusy(false);
    }
  }
  async function submitJob(clip: number, step: UnitStep) {
    if (step === "voice") {
      setBusy(true);
      setFailure(undefined);
      try {
        const r = await api.POST("/api/admin/clips/{id}/voice", {
          params: { path: { id: clip }, header: mutation(session?.csrf_token) },
          body: {},
        });
        if (r.error) setFailure(errorCode(r.error) ?? "unknown");
        else {
          setNotice(true);
          reload();
        }
      } catch {
        setFailure("unavailable");
      } finally {
        setBusy(false);
      }
    } else await scheduleTranscription(clip);
  }
  async function scheduleTranscription(clip: number, confirm = false) {
    setBusy(true);
    setFailure(undefined);
    try {
      const result = await api.POST("/api/admin/clips/{id}/transcription", {
        params: {
          path: { id: clip },
          header: mutation(session?.csrf_token),
          query: { confirm_replace_corrections: confirm || undefined },
        },
        body: {},
      });
      if (
        result.error &&
        "corrections" in result.error &&
        result.error.corrections &&
        result.error.error === "corrections_would_be_replaced"
      ) {
        setTranscriptionClip(clip);
        setWarning({ corrections: result.error.corrections });
      } else if (result.error) setFailure(errorCode(result.error) ?? "unknown");
      else {
        setWarning(undefined);
        setTranscriptionClip(undefined);
        setNotice(true);
        reload();
      }
    } catch {
      setFailure("unavailable");
    } finally {
      setBusy(false);
    }
  }
  async function save(paused: boolean, maxConcurrency: number) {
    setBusy(true);
    setFailure(undefined);
    setNotice(false);
    try {
      const result = await api.PUT("/api/admin/job-controls", {
        params: { header: mutation(session?.csrf_token) },
        body: { paused, max_concurrency: maxConcurrency },
      });
      if (result.error) setFailure(errorCode(result.error) ?? "unknown");
      else {
        setNotice(true);
        reload();
      }
    } catch {
      setFailure("unavailable");
    } finally {
      setBusy(false);
    }
  }
  async function showHistory(job: JobRef) {
    try {
      const result = await api.GET("/api/admin/jobs/{id}/history", {
        params: { path: { id: job.id } },
      });
      if (result.data)
        setHistory((old) => ({ ...old, [job.id]: result.data.items }));
      else setFailure(errorCode(result.error) ?? "unknown");
    } catch {
      setFailure("unavailable");
    }
  }
  async function retry(job: JobRef, confirm = false) {
    setBusy(true);
    setFailure(undefined);
    try {
      const result = await api.POST("/api/admin/jobs/{id}/retry", {
        params: {
          path: { id: job.id },
          header: mutation(session?.csrf_token),
          query: { confirm_replace_corrections: confirm || undefined },
        },
      });
      if (
        result.error &&
        "corrections" in result.error &&
        result.error.corrections &&
        result.error.error === "corrections_would_be_replaced"
      )
        setWarning({ job, corrections: result.error.corrections });
      else if (result.error) setFailure(errorCode(result.error) ?? "unknown");
      else {
        setWarning(undefined);
        reload();
      }
    } catch {
      setFailure("unavailable");
    } finally {
      setBusy(false);
    }
  }
  async function cancel(job: JobRef) {
    setBusy(true);
    setFailure(undefined);
    try {
      const result = await api.DELETE("/api/admin/jobs/{id}", {
        params: { path: { id: job.id }, header: mutation(session?.csrf_token) },
      });
      if (result.error) setFailure(errorCode(result.error) ?? "unknown");
      else reload();
    } catch {
      setFailure("unavailable");
    } finally {
      setBusy(false);
    }
  }
  return (
    <div className="flex flex-col gap-4">
      <ConfirmDialog
        open={!!warning}
        onOpenChange={(open) => {
          if (!open) {
            setWarning(undefined);
            setTranscriptionClip(undefined);
          }
        }}
        title={t("queue.confirmTitle")}
        cancelLabel={t("queue.cancel")}
        confirmLabel={t("queue.confirm")}
        busy={busy}
        onConfirm={() => {
          if (transcriptionClip)
            void scheduleTranscription(transcriptionClip, true);
          else if (warning?.job) void retry(warning.job, true);
          else if (warning?.input) void schedule(warning.input, true);
        }}
      >
        {warning && (
          <p>
            {t("queue.confirmBody", {
              texts: warning.corrections.text_segments,
              speakers: warning.corrections.speaker_segments,
              authors: warning.corrections.authors.join(", "),
            })}
          </p>
        )}
      </ConfirmDialog>
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h2 className="text-lg font-semibold">{t("queue.title")}</h2>
          <p className="mt-1 max-w-xl text-mute">
            {language === "fr"
              ? "Suivez chaque clip et les traitements effectués par vos workers."
              : "Follow each clip and the processing performed by your workers."}
          </p>
        </div>
        <button onClick={reload} className={buttonClass()}>
          {language === "fr" ? "⟳ Actualiser" : "⟳ Refresh"}
        </button>
      </div>
      {state.status === "loading" && <p role="status">{t("app.loading")}</p>}
      {state.status === "failed" && (
        <ErrorAlert>
          {error(state.code)}{" "}
          <button onClick={reload} className={buttonClass()}>
            {t("queue.retry")}
          </button>
        </ErrorAlert>
      )}
      {failure && <ErrorAlert>{error(failure)}</ErrorAlert>}
      {notice && <StatusNotice>{t("queue.updated")}</StatusNotice>}
      {state.status === "ready" && (
        <>
          <ScheduleTaskForm
            workers={state.value.workers}
            busy={busy}
            onSchedule={(clip, step) => void submitJob(clip, step)}
          />
          <div className="flex flex-wrap items-center justify-between gap-4 rounded-xl border border-line bg-surface p-4">
            <div className="flex min-w-0 items-center gap-3">
              <span className="grid size-10 shrink-0 place-items-center rounded-xl bg-accent-soft text-accent-ink">
                <Monitor size={20} />
              </span>
              <div className="min-w-0">
                <p className="font-semibold">
                  {state.value.workers.find((w) => !w.revoked_at)?.name ??
                    (language === "fr"
                      ? "Aucun worker actif"
                      : "No active worker")}
                </p>
                <p className="mt-1 text-sm text-mute">
                  {(() => {
                    const machine = state.value.workers.find(
                      (w) => !w.revoked_at,
                    )?.machine;
                    const memory = machine?.gpu_memory_bytes
                      ? `${Math.round(machine.gpu_memory_bytes / 1024 ** 3)} ${language === "fr" ? "Go" : "GB"}`
                      : undefined;
                    const running = state.value.board.cards.filter(
                      (card) => card.state === "running",
                    ).length;
                    return [
                      machine?.gpu_name,
                      memory,
                      `${running} ${language === "fr" ? "traitements en cours" : "running tasks"}`,
                    ]
                      .filter(Boolean)
                      .join(" · ");
                  })()}
                </p>
              </div>
            </div>
          </div>

          <QueueControlBar
            queue={state.value.board.queue}
            busy={busy}
            onSave={(paused, max) => void save(paused, max)}
          />

          <JobWorkflow
            board={state.value.board}
            emptyMessage={t("queue.empty")}
            actions={(job) => (
              <div className="flex flex-wrap gap-2">
                <button
                  type="button"
                  className={buttonClass()}
                  onClick={() => void showHistory(job)}
                >
                  {t("queue.history")}
                </button>
                {job.status === "dead" && (
                  <button
                    disabled={busy}
                    className={buttonClass()}
                    onClick={() => void retry(job)}
                  >
                    {t("queue.retry")}
                  </button>
                )}
                {["pending", "retry_wait", "leased"].includes(job.status) && (
                  <button
                    disabled={busy}
                    className={buttonClass()}
                    onClick={() => void cancel(job)}
                  >
                    {t("queue.cancel")}
                  </button>
                )}
                {history[job.id] && (
                  <ul className="w-full">
                    {history[job.id].map((item) => (
                      <li key={item.id}>
                        {t(`queue.event.${item.event}`)} · {item.created_at} ·{" "}
                        {item.error_code}
                      </li>
                    ))}
                  </ul>
                )}
              </div>
            )}
          />
        </>
      )}
    </div>
  );
}
