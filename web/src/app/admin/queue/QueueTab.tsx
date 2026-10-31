import { ChevronLeft, ChevronRight, Monitor } from "lucide-react";
import { JobWorkflow } from "./JobWorkflow";
import { useCallback, useEffect, useState, type FormEvent } from "react";
import { api, errorCode, mutation } from "../../api/client";
import { useLoaded } from "../../api/loaded";
import { ErrorAlert, StatusNotice } from "../../components/Alert";
import { buttonClass } from "../../components/buttons";
import { Field } from "../../components/Field";
import { ConfirmDialog } from "../../components/ConfirmDialog";
import { useI18n } from "../../i18n";
import { useSession } from "../../session";
import type { components } from "../../../api.gen";

type Filters = {
  status?: components["schemas"]["Job"]["status"];
  clip_id?: number;
  step?: string;
};
async function load(page: number, filters: Filters) {
  try {
    const [controls, jobs, recent, workers] = await Promise.all([
      api.GET("/api/admin/job-controls"),
      api.GET("/api/admin/jobs", {
        params: {
          query: { page_size: 20, ...filters, page, workflow: "active" },
        },
      }),
      api.GET("/api/admin/jobs", {
        params: { query: { workflow: "done", page: 1, page_size: 3 } },
      }),
      api.GET("/api/admin/workers"),
    ]);
    if (!controls.data || !jobs.data || !recent.data || !workers.data)
      return {
        ok: false as const,
        code: errorCode(controls.error ?? jobs.error) ?? "unknown",
      };
    return {
      ok: true as const,
      value: {
        controls: controls.data,
        workers: workers.data.items,
        jobs: {
          ...jobs.data,
          items: [...jobs.data.items, ...recent.data.items],
        },
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
  const [page, setPage] = useState(1);
  const [filters, setFilters] = useState<Filters>({});
  const loadPage = useCallback(() => load(page, filters), [page, filters]);
  const { state, reload } = useLoaded(loadPage);
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
        setPage(1);
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
  function filterJobs(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const f = new FormData(event.currentTarget);
    const status = String(f.get("status") ?? "") as Filters["status"];
    const clip = String(f.get("clip") ?? "");
    setPage(1);
    setFilters({
      status: status || undefined,
      clip_id: clip ? Number(clip) : undefined,
      step: String(f.get("step") ?? "").trim() || undefined,
    });
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
                    const running = state.value.jobs.items.filter(
                      (job) => job.status !== "done",
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

          <JobWorkflow
            jobs={state.value.jobs.items}
            emptyMessage={t("queue.empty")}
            toolbar={
              <form
                onSubmit={filterJobs}
                className="flex flex-wrap items-end gap-4"
              >
                <label className="flex flex-col gap-1 font-medium">
                  <span className="text-sm">{t("queue.filterStatus")}</span>
                  <select
                    name="status"
                    className="h-11 min-w-48 rounded-lg border border-line-strong bg-surface px-3 text-sm md:h-9"
                  >
                    <option value="">{t("queue.allStatuses")}</option>
                    {(
                      [
                        "pending",
                        "leased",
                        "done",
                        "failed",
                        "retry_wait",
                        "dead",
                        "cancelled",
                      ] as const
                    ).map((status) => (
                      <option value={status} key={status}>
                        {t(`queue.status.${status}`)}
                      </option>
                    ))}
                  </select>
                </label>
                <label className="flex flex-col gap-1 font-medium">
                  <span className="text-sm">{t("queue.filterClip")}</span>
                  <input
                    name="clip"
                    type="number"
                    min={1}
                    placeholder="ex. 18"
                    className="h-11 w-48 rounded-lg border border-line-strong bg-surface px-3 text-sm md:h-9"
                  />
                </label>
                <label className="flex flex-col gap-1 font-medium">
                  <span className="text-sm">{t("queue.filterStep")}</span>
                  <input
                    name="step"
                    className="h-11 w-48 rounded-lg border border-line-strong bg-surface px-3 text-sm md:h-9"
                  />
                </label>
                <button className={buttonClass()}>
                  {language === "fr" ? "Filtrer" : "Filter"}
                </button>
              </form>
            }
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

          <nav
            aria-label={
              language === "fr" ? "Pagination de la file" : "Queue pagination"
            }
            className="flex items-center justify-center gap-3 py-3"
          >
            <button
              aria-label={
                language === "fr" ? "Page précédente" : "Previous page"
              }
              className="grid size-11 place-items-center rounded-full text-mute hover:bg-side disabled:opacity-30"
              disabled={page === 1}
              onClick={() => setPage((p) => p - 1)}
            >
              <ChevronLeft size={18} />
            </button>
            <span className="min-w-16 text-center text-sm text-mute">
              {page} / {Math.max(1, Math.ceil(state.value.jobs.total / 20))}
            </span>
            <button
              aria-label={language === "fr" ? "Page suivante" : "Next page"}
              className="grid size-11 place-items-center rounded-full text-mute hover:bg-side disabled:opacity-30"
              disabled={page * 20 >= state.value.jobs.total}
              onClick={() => setPage((p) => p + 1)}
            >
              <ChevronRight size={18} />
            </button>
          </nav>

          <details className="rounded-xl border border-line p-4">
            <summary className="cursor-pointer list-none p-1">
              <span className="text-base font-semibold">
                {t("queue.schedule")}
              </span>
              <span className="ml-2 text-sm text-mute">
                {language === "fr"
                  ? "Choisir un clip et démarrer au bon traitement"
                  : "Choose a clip and start at the right step"}
              </span>
            </summary>
            <form
              onSubmit={submitJob}
              className="flex flex-col gap-3 px-1 pb-1 pt-4"
            >
              <label className="flex flex-col gap-1 text-sm font-medium">
                {t("queue.clip")}
                <input
                  name="clip"
                  type="number"
                  min={1}
                  required
                  placeholder="ex. 18"
                  className="h-11 w-full rounded-lg border border-line-strong bg-surface px-3 text-sm text-ink md:h-9"
                />
              </label>
              {!state.value.workers.some(
                (w) =>
                  !w.revoked_at &&
                  w.supported_steps?.some(
                    (s) =>
                      s.startsWith("transcription.") ||
                      s === "voice.*" ||
                      s.startsWith("voice.diarization."),
                  ),
              ) && (
                <p
                  role="status"
                  className="rounded-lg bg-warn-bg p-3 text-warn-fg"
                >
                  {language === "fr"
                    ? "Aucun worker n’a déclaré de traitement compatible. Démarrez un worker puis actualisez cette page."
                    : "No worker has reported compatible tasks. Start a worker, then refresh this page."}
                </p>
              )}
              <fieldset className="mt-2 grid gap-3 md:grid-cols-2">
                <legend className="mb-2 text-sm font-medium">
                  {t("queue.step")}
                </legend>
                {state.value.workers.some(
                  (w) =>
                    !w.revoked_at &&
                    w.supported_steps?.some((s) =>
                      s.startsWith("transcription."),
                    ),
                ) && (
                  <label className="flex min-h-16 cursor-pointer items-start gap-2 rounded-lg border border-line p-3 has-[:checked]:border-accent has-[:checked]:bg-accent-soft">
                    <input
                      type="radio"
                      name="step"
                      value="transcription"
                      required
                      className="sr-only"
                    />
                    <span>
                      <strong className="block text-sm">
                        {language === "fr"
                          ? "Étape 0 · Transcription"
                          : "Step 0 · Transcription"}
                      </strong>
                      <span className="mt-1 block text-xs text-mute">
                        {language === "fr"
                          ? "Écrire les paroles du clip"
                          : "Write down the clip’s speech"}
                      </span>
                    </span>
                  </label>
                )}
                {state.value.workers.some(
                  (w) =>
                    !w.revoked_at &&
                    w.supported_steps?.some(
                      (s) =>
                        s === "voice.*" || s.startsWith("voice.diarization."),
                    ),
                ) && (
                  <label className="flex min-h-16 cursor-pointer items-start gap-2 rounded-lg border border-line p-3 has-[:checked]:border-accent has-[:checked]:bg-accent-soft">
                    <input
                      type="radio"
                      name="step"
                      value="voice"
                      required
                      className="sr-only"
                    />
                    <span>
                      <strong className="block text-sm">
                        {language === "fr"
                          ? "Étape 1 · Analyse des voix"
                          : "Step 1 · Voice analysis"}
                      </strong>
                      <span className="mt-1 block text-xs text-mute">
                        {language === "fr"
                          ? "Nécessite une transcription"
                          : "Requires a transcript"}
                      </span>
                    </span>
                  </label>
                )}
              </fieldset>
              <div>
                <button disabled={busy} className={buttonClass("primary")}>
                  {language === "fr" ? "Planifier" : "Schedule"}
                </button>
              </div>
            </form>
          </details>
        </>
      )}
    </div>
  );
}
