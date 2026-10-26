import { useCallback, useState, type FormEvent } from "react";
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
    const [controls, jobs] = await Promise.all([
      api.GET("/api/admin/job-controls"),
      api.GET("/api/admin/jobs", {
        params: { query: { page, page_size: 20, ...filters } },
      }),
    ]);
    if (!controls.data || !jobs.data)
      return {
        ok: false as const,
        code: errorCode(controls.error ?? jobs.error) ?? "unknown",
      };
    return {
      ok: true as const,
      value: { controls: controls.data, jobs: jobs.data },
    };
  } catch {
    return { ok: false as const, code: "unavailable" };
  }
}
type Job = components["schemas"]["Job"];
type Event = components["schemas"]["JobEvent"];
export function QueueTab() {
  const { t, error } = useI18n();
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
  function submitJob(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const f = new FormData(event.currentTarget);
    void schedule({
      clip_id: Number(f.get("clip")),
      step: String(f.get("step")),
      input_version: String(f.get("version")),
      priority: Number(f.get("priority")),
      max_attempts: Number(f.get("attempts")),
    });
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
    <div className="flex max-w-4xl flex-col gap-5">
      <ConfirmDialog
        open={!!warning}
        onOpenChange={(open) => {
          if (!open) setWarning(undefined);
        }}
        title={t("queue.confirmTitle")}
        cancelLabel={t("queue.cancel")}
        confirmLabel={t("queue.confirm")}
        busy={busy}
        onConfirm={() => {
          if (warning?.job) void retry(warning.job, true);
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
      <h2 className="text-lg font-semibold">{t("queue.title")}</h2>
      {state.status === "loading" && <p role="status">{t("app.loading")}</p>}
      {state.status === "failed" && (
        <ErrorAlert>{error(state.code)}</ErrorAlert>
      )}
      {failure && <ErrorAlert>{error(failure)}</ErrorAlert>}
      {notice && <StatusNotice>{t("queue.updated")}</StatusNotice>}
      {state.status === "ready" && (
        <>
          <form
            onSubmit={save}
            className="flex max-w-md flex-col gap-4 rounded-xl border border-line p-4"
          >
            <Field
              label={t("queue.concurrency")}
              type="number"
              min={1}
              max={4}
              required
              value={
                concurrency ?? String(state.value.controls.max_concurrency)
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
            <div>
              <button disabled={busy} className={buttonClass("primary")}>
                {t("queue.save")}
              </button>
            </div>
          </form>
          <details className="rounded-xl border border-line p-4">
            <summary className="cursor-pointer font-medium">
              {t("queue.schedule")}
            </summary>
            <form
              onSubmit={submitJob}
              className="flex max-w-md flex-col gap-3 pt-3"
            >
              <Field
                name="clip"
                label={t("queue.clip")}
                type="number"
                min={1}
                required
              />
              <Field
                name="step"
                label={t("queue.step")}
                required
                maxLength={64}
                pattern="[a-z][a-z0-9_.-]{0,63}"
              />
              <Field
                name="version"
                label={t("queue.version")}
                required
                maxLength={128}
              />
              <Field
                name="priority"
                label={t("queue.priority")}
                type="number"
                min={-100}
                max={100}
                defaultValue={0}
              />
              <Field
                name="attempts"
                label={t("queue.attempts")}
                type="number"
                min={1}
                max={20}
                defaultValue={3}
                required
              />
              <div>
                <button disabled={busy} className={buttonClass()}>
                  {t("queue.schedule")}
                </button>
              </div>
            </form>
          </details>
          <form
            onSubmit={filterJobs}
            className="flex flex-wrap items-end gap-3"
          >
            <label className="flex flex-col gap-1.5 font-medium">
              {t("queue.filterStatus")}
              <select
                name="status"
                className="h-11 rounded-lg border border-line-strong bg-surface px-3"
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
            <Field
              name="clip"
              label={t("queue.filterClip")}
              type="number"
              min={1}
            />
            <Field name="step" label={t("queue.filterStep")} />
            <button className={buttonClass()}>{t("queue.filter")}</button>
          </form>
          {state.value.jobs.items.length === 0 ? (
            <p>{t("queue.empty")}</p>
          ) : (
            <ul className="flex list-none flex-col gap-3 p-0">
              {state.value.jobs.items.map((job) => (
                <li key={job.id} className="rounded-xl border border-line p-4">
                  <strong>{job.step}</strong>
                  <p>
                    {t("queue.job", {
                      id: job.id,
                      clip: job.clip_id,
                      status: t(`queue.status.${job.status}`),
                    })}
                  </p>
                  <p className="text-mute">{job.last_error}</p>
                  <div className="flex gap-2">
                    <button
                      type="button"
                      className={buttonClass()}
                      onClick={() => void showHistory(job)}
                    >
                      {t("queue.history")}
                    </button>
                    {job.status === "dead" && (
                      <button
                        type="button"
                        disabled={busy}
                        className={buttonClass()}
                        onClick={() => void retry(job)}
                      >
                        {t("queue.retry")}
                      </button>
                    )}
                    {["pending", "retry_wait", "leased"].includes(
                      job.status,
                    ) && (
                      <button
                        type="button"
                        disabled={busy}
                        className={buttonClass()}
                        onClick={() => void cancel(job)}
                      >
                        {t("queue.cancel")}
                      </button>
                    )}
                  </div>
                  {history[job.id] && (
                    <ul>
                      {history[job.id].map((item) => (
                        <li key={item.id}>
                          {t(`queue.event.${item.event}`)} · {item.created_at} ·{" "}
                          {item.error_code}
                        </li>
                      ))}
                    </ul>
                  )}
                </li>
              ))}
            </ul>
          )}
          <div className="flex items-center gap-2">
            <button
              type="button"
              disabled={page <= 1}
              className={buttonClass()}
              onClick={() => setPage(page - 1)}
            >
              {t("queue.previous")}
            </button>
            <span>{t("queue.page", { page })}</span>
            <button
              type="button"
              disabled={page * 20 >= state.value.jobs.total}
              className={buttonClass()}
              onClick={() => setPage(page + 1)}
            >
              {t("queue.next")}
            </button>
          </div>
        </>
      )}
    </div>
  );
}
