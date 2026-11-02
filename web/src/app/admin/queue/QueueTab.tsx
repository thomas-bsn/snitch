import { Monitor } from "lucide-react";
import { AnalysisBoard, type Queue } from "./AnalysisBoard";
import { ClipHistory } from "./ClipHistory";
import { useCallback, useRef, useState, type FormEvent } from "react";
import { api, errorCode, mutation } from "../../api/client";
import { useEventStream } from "../../api/useEventStream";
import { useLoaded } from "../../api/loaded";
import { ErrorAlert, StatusNotice } from "../../components/Alert";
import { buttonClass } from "../../components/buttons";
import { Field } from "../../components/Field";
import { ConfirmDialog } from "../../components/ConfirmDialog";
import { useI18n } from "../../i18n";
import { useSession } from "../../session";
import type { components } from "../../../api.gen";

async function load() {
  try {
    const [controls, workers] = await Promise.all([
      api.GET("/api/admin/job-controls"),
      api.GET("/api/admin/workers"),
    ]);
    if (!controls.data || !workers.data)
      return {
        ok: false as const,
        code: errorCode(controls.error ?? workers.error) ?? "unknown",
      };
    return {
      ok: true as const,
      value: { controls: controls.data, workers: workers.data.items },
    };
  } catch {
    return { ok: false as const, code: "unavailable" };
  }
}
type Job = components["schemas"]["Job"];
type QueueDelta = components["schemas"]["AnalysisQueueDelta"];
type Event = components["schemas"]["JobEvent"];
export function QueueTab() {
  const { t, error, language } = useI18n();
  const { session } = useSession();
  const { state, reload } = useLoaded(load);
  const [queue, setQueueView] = useState<Queue>();
  // The stream sends what changed after the first snapshot: the view it applies it to is kept here.
  const held = useRef<Queue | undefined>(undefined);
  const setQueue = useCallback((next: Queue) => {
    held.current = next;
    setQueueView(next);
  }, []);
  // The server pushes the whole queue once, then only the runs that changed; polling only covers a stream that is down.
  // A failed refresh keeps the last known view and never raises the global unreachable screen:
  // the page says it is reconnecting and the next snapshot or poll replaces the view.
  const refreshQueue = useCallback(async () => {
    try {
      const response = await fetch("/api/admin/analysis/queue");
      if (response.ok) setQueue((await response.json()) as Queue);
    } catch {
      // Still offline.
    }
  }, [setQueue]);
  const applyDelta = useCallback(
    (delta: QueueDelta) => {
      const before = held.current;
      if (!before) return false;
      const known = new Map(before.active.map((run) => [run.id, run]));
      const changed = new Map(delta.active.map((run) => [run.id, run]));
      const active: Queue["active"] = [];
      for (const id of delta.active_ids) {
        const run = changed.get(id) ?? known.get(id);
        if (!run) return false;
        active.push(run);
      }
      setQueue({
        active,
        completed: delta.completed,
        tasks: delta.tasks,
        summary: delta.summary,
      });
      return true;
    },
    [setQueue],
  );
  const stream = useEventStream<Queue, QueueDelta>(
    "/api/admin/analysis/queue/events",
    setQueue,
    refreshQueue,
    undefined,
    applyDelta,
  );
  const [paused, setPaused] = useState<boolean>();
  const [concurrency, setConcurrency] = useState<string>();
  const [failure, setFailure] = useState<string>();
  const [notice, setNotice] = useState(false);
  const [history, setHistory] = useState<Record<number, Event[]>>({});
  const [busy, setBusy] = useState(false);
  const [transcriptionClip, setTranscriptionClip] = useState<number>();
  const [warning, setWarning] = useState<{
    job?: Job;
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
  async function submitJob(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const f = new FormData(event.currentTarget);
    const clip = Number(f.get("clip"));
    if (!["transcription", "voice"].includes(String(f.get("step")))) return;
    if (f.get("step") === "voice") {
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
  async function save(event: FormEvent) {
    event.preventDefault();
    if (state.status !== "ready") return;
    setBusy(true);
    setFailure(undefined);
    setNotice(false);
    try {
      const result = await api.PUT("/api/admin/job-controls", {
        params: { header: mutation(session?.csrf_token) },
        body: {
          paused: paused ?? state.value.controls.paused,
          max_concurrency: Number(
            concurrency ?? state.value.controls.max_concurrency,
          ),
        },
      });
      if (result.error) setFailure(errorCode(result.error) ?? "unknown");
      else {
        setNotice(true);
        setPaused(undefined);
        setConcurrency(undefined);
        reload();
      }
    } catch {
      setFailure("unavailable");
    } finally {
      setBusy(false);
    }
  }
  async function showHistory(job: Job) {
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
  async function retry(job: Job, confirm = false) {
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
  async function cancel(job: Job) {
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
  const activeWorkers =
    state.status === "ready"
      ? state.value.workers.filter((w) => !w.revoked_at)
      : [];
  const transcriptionReady = activeWorkers.some((w) =>
    w.supported_steps?.some((s) => s.startsWith("transcription.")),
  );
  const voiceReady = activeWorkers.some((w) =>
    w.supported_steps?.some(
      (s) => s === "voice.*" || s.startsWith("voice.diarization."),
    ),
  );
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
        <p role="status" className="m-0 text-sm text-mute">
          {stream === "live"
            ? t("queue.board.live")
            : stream === "forbidden"
              ? t("queue.board.forbidden")
              : t("queue.board.offline")}
        </p>
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
          <details
            open
            data-advanced-schedule
            className="rounded-xl border border-line p-4"
          >
            <summary className="cursor-pointer list-none p-1">
              <span className="text-base font-semibold">
                {t("queue.schedule")}
              </span>
              <span className="ml-2 text-sm text-mute">
                {language === "fr"
                  ? "Mode avancé : une étape isolée sur un clip sans analyse en cours"
                  : "Advanced: a single step on a clip with no analysis in progress"}
              </span>
            </summary>
            <form
              onSubmit={submitJob}
              className="flex flex-wrap items-end gap-3 px-1 pb-1 pt-4"
            >
              <label className="flex flex-col gap-1 text-sm font-medium">
                {t("queue.clip")}
                <input
                  name="clip"
                  type="number"
                  min={1}
                  required
                  placeholder="ex. 18"
                  className="h-11 w-40 rounded-lg border border-line-strong bg-surface px-3 text-sm text-ink md:h-9"
                />
              </label>
              <label className="flex flex-col gap-1 text-sm font-medium">
                {t("queue.step")}
                <select
                  name="step"
                  required
                  defaultValue=""
                  className="h-11 rounded-lg border border-line-strong bg-surface px-3 text-sm text-ink md:h-9"
                >
                  <option value="" disabled>
                    {language === "fr" ? "Choisir…" : "Choose…"}
                  </option>
                  {transcriptionReady && (
                    <option value="transcription">
                      {language === "fr" ? "Transcription" : "Transcription"}
                    </option>
                  )}
                  {voiceReady && (
                    <option value="voice">
                      {language === "fr"
                        ? "Analyse des voix"
                        : "Voice analysis"}
                    </option>
                  )}
                </select>
              </label>
              <button disabled={busy} className={buttonClass("primary")}>
                {language === "fr" ? "Planifier" : "Schedule"}
              </button>
              {!transcriptionReady && !voiceReady && (
                <p
                  role="status"
                  className="m-0 w-full rounded-lg bg-warn-bg p-3 text-warn-fg"
                >
                  {language === "fr"
                    ? "Aucun worker n’a déclaré de traitement compatible. Démarrez un worker : la page se met à jour seule."
                    : "No worker has reported compatible tasks. Start a worker: the page updates on its own."}
                </p>
              )}
            </form>
          </details>

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
                    const running = queue?.summary.running_tasks ?? 0;
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
            <details className="relative">
              <summary className="cursor-pointer list-none px-2 py-2 text-sm font-medium hover:text-accent-ink">
                {language === "fr"
                  ? "Réglages de concurrence　⌄"
                  : "Concurrency settings　⌄"}
              </summary>
              <div className="absolute right-0 z-20 mt-2 w-80 rounded-xl border border-line bg-surface p-4 shadow-lg">
                <p className="mb-3 text-sm text-mute">
                  {language === "fr"
                    ? "Nombre maximal de tâches lancées simultanément sur tous les workers."
                    : "Maximum number of tasks started at once across all workers."}
                </p>
                <form onSubmit={save} className="flex flex-col gap-4">
                  <Field
                    label={t("queue.concurrency")}
                    type="number"
                    min={1}
                    max={4}
                    required
                    value={
                      concurrency ??
                      String(state.value.controls.max_concurrency)
                    }
                    onChange={(e) => setConcurrency(e.target.value)}
                  />
                  <label className="flex min-h-11 items-center gap-2">
                    <input
                      type="checkbox"
                      checked={paused ?? state.value.controls.paused}
                      onChange={(e) => setPaused(e.target.checked)}
                    />
                    {t("queue.pause")}
                  </label>
                  <button disabled={busy} className={buttonClass("primary")}>
                    {t("queue.save")}
                  </button>
                </form>
              </div>
            </details>
          </div>

          <div className="flex justify-end">
            <ClipHistory />
          </div>
          {queue ? (
            <AnalysisBoard
              queue={queue}
              taskActions={(job) => (
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
          ) : (
            <p role="status">{t("app.loading")}</p>
          )}
        </>
      )}
    </div>
  );
}
