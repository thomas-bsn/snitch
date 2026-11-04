import { useCallback, useEffect, useId, useRef, useState } from "react";
import { ErrorAlert, StatusNotice } from "../components/Alert";
import { buttonClass } from "../components/buttons";
import { ConfirmDialog } from "../components/ConfirmDialog";
import { StatusPill } from "../components/StatusPill";
import { useI18n } from "../i18n";
import { useSession } from "../session";
import {
  changeIndexRun,
  fetchCurrentIndexRun,
  previewIndexRun,
  startIndexRun,
  type IndexAction,
  type IndexMode,
  type IndexRun,
  type IndexRunPreview,
} from "./rootIndexApi";

/** Per-video failure codes the scan leaves on a video; anything else reads as "other". */
const FAILURE_CODES = new Set([
  "missing",
  "unreadable",
  "video_missing",
  "video_unreadable",
  "transcription_scheduling_failed",
]);
type FailureCode =
  | "missing"
  | "unreadable"
  | "video_missing"
  | "video_unreadable"
  | "transcription_scheduling_failed";

/** How often the progress is read while an indexation is running or paused. */
const POLL_MS = 3000;

/**
 * Indexation of the whole Racine vidéo (#275, ADR 0038): one explicit, durable action for
 * administrators, in the administration and in the setup wizard. The work lives on the server:
 * leaving the page neither stops nor duplicates it.
 */
export function RootIndexingPanel({
  defaultMode = "full",
  statusOnly = false,
}: {
  defaultMode?: IndexMode;
  /** The task queue shows a run in progress with its controls, and nothing otherwise. */
  statusOnly?: boolean;
}) {
  const { t, error } = useI18n();
  const { session } = useSession();
  const csrf = session?.csrf_token;
  const id = useId();
  const [run, setRun] = useState<IndexRun | null | undefined>(undefined);
  const [mode, setMode] = useState<IndexMode>(defaultMode);
  const [preview, setPreview] = useState<IndexRunPreview | null>(null);
  const [confirm, setConfirm] = useState<"start" | "cancel" | null>(null);
  const [busy, setBusy] = useState(false);
  const [failure, setFailure] = useState<string>();

  // One read at a time, and none after the panel is gone: a slow answer never overwrites a newer one.
  const reading = useRef(false);
  const mounted = useRef(true);
  useEffect(() => {
    mounted.current = true;
    return () => {
      mounted.current = false;
    };
  }, []);
  const refresh = useCallback(async () => {
    if (reading.current) return;
    reading.current = true;
    const result = await fetchCurrentIndexRun();
    reading.current = false;
    if (!mounted.current) return;
    if (result.ok) {
      setRun(result.value);
      setFailure(undefined);
    } else setFailure(result.code);
  }, []);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  const active = run?.status === "running" || run?.status === "paused";
  useEffect(() => {
    if (!active && !statusOnly) return;
    const timer = window.setInterval(() => void refresh(), POLL_MS);
    return () => window.clearInterval(timer);
  }, [active, statusOnly, refresh]);

  async function openStart() {
    setFailure(undefined);
    setBusy(true);
    const result = await previewIndexRun();
    setBusy(false);
    if (!result.ok) {
      setFailure(result.code);
      return;
    }
    setPreview(result.value);
    setConfirm("start");
  }

  async function start() {
    setBusy(true);
    const result = await startIndexRun(mode, csrf);
    setBusy(false);
    setConfirm(null);
    if (result.ok) {
      setRun(result.value);
      setFailure(undefined);
    } else setFailure(result.code);
  }

  async function act(action: IndexAction) {
    if (!run) return;
    setBusy(true);
    setFailure(undefined);
    const result = await changeIndexRun(run.id, action, csrf);
    setBusy(false);
    setConfirm(null);
    if (result.ok) setRun(result.value);
    else {
      setFailure(result.code);
      void refresh();
    }
  }

  const failureText = (code: string) =>
    FAILURE_CODES.has(code)
      ? t(`libraryIndex.failureCode.${code as FailureCode}`)
      : t("libraryIndex.failureCode.other");

  const statusTone = {
    running: "ok",
    paused: "idle",
    completed: "ok",
    cancelled: "idle",
  } as const;

  if (statusOnly && !active) return null;

  return (
    <section
      aria-labelledby={`${id}-title`}
      className="flex max-w-3xl flex-col gap-3 rounded-xl border border-line p-4"
    >
      <h2 id={`${id}-title`} className="text-lg font-semibold">
        {t("libraryIndex.title")}
      </h2>
      {!statusOnly && <p className="text-mute">{t("libraryIndex.lead")}</p>}
      {failure && <ErrorAlert>{error(failure)}</ErrorAlert>}

      {run === undefined && !failure && <p role="status">{t("app.loading")}</p>}

      {run && (
        <div className="flex flex-col gap-2">
          <div className="flex flex-wrap items-center gap-2">
            <StatusPill tone={statusTone[run.status]}>
              {t(`libraryIndex.status.${run.status}`)}
            </StatusPill>
            <span className="text-mute">
              {t(`libraryIndex.mode.${run.mode}`)}
            </span>
          </div>
          {/* Announced only when the state changes, not at every refresh of the counters. */}
          <p role="status" className="sr-only">
            {t(`libraryIndex.status.${run.status}`)}
          </p>
          <p className="font-medium">
            {t("libraryIndex.progress", {
              done: run.indexed,
              total: run.total,
            })}
          </p>
          <progress
            className="h-2 w-full"
            value={run.indexed + run.failed}
            max={Math.max(run.total, 1)}
            aria-label={t("libraryIndex.progressLabel")}
          />
          {run.mode === "full" && (
            <p className="text-mute">
              {t("libraryIndex.analysisStarted", {
                count: run.analysis_started,
              })}
            </p>
          )}
          {run.waiting_for_worker && run.status === "running" && (
            <StatusNotice>{t("libraryIndex.waitingForWorker")}</StatusNotice>
          )}
          {run.failed > 0 && (
            <div role="group" aria-label={t("libraryIndex.failuresTitle")}>
              <p className="font-medium">
                {t("libraryIndex.failures", { count: run.failed })}
              </p>
              <ul className="m-0 list-disc pl-5 text-mute">
                {run.failures.map((item) => (
                  <li key={item.path} className="break-all">
                    {item.path} · {failureText(item.code)}
                  </li>
                ))}
              </ul>
            </div>
          )}
          {active && (
            <div className="flex flex-wrap gap-2">
              {run.status === "running" ? (
                <button
                  type="button"
                  className={buttonClass("secondary")}
                  disabled={busy}
                  onClick={() => void act("pause")}
                >
                  {t("libraryIndex.pause")}
                </button>
              ) : (
                <button
                  type="button"
                  className={buttonClass("primary")}
                  disabled={busy}
                  onClick={() => void act("resume")}
                >
                  {t("libraryIndex.resume")}
                </button>
              )}
              <button
                type="button"
                className={buttonClass("danger")}
                disabled={busy}
                onClick={() => setConfirm("cancel")}
              >
                {t("libraryIndex.cancel")}
              </button>
            </div>
          )}
        </div>
      )}

      {run !== undefined && !active && (
        <fieldset className="flex flex-col gap-2 border-0 p-0">
          <legend className="font-medium">
            {t("libraryIndex.modeLegend")}
          </legend>
          {(["full", "cards"] as const).map((value) => (
            <label key={value} className="flex min-h-11 items-start gap-2">
              <input
                type="radio"
                name={`${id}-mode`}
                className="mt-1"
                checked={mode === value}
                onChange={() => setMode(value)}
              />
              <span>
                <span className="font-medium">
                  {t(`libraryIndex.mode.${value}`)}
                </span>
                <span className="block text-mute">
                  {t(`libraryIndex.modeHelp.${value}`)}
                </span>
              </span>
            </label>
          ))}
          <div>
            <button
              type="button"
              className={buttonClass("primary")}
              disabled={busy}
              onClick={() => void openStart()}
            >
              {t("libraryIndex.start")}
            </button>
          </div>
        </fieldset>
      )}

      <ConfirmDialog
        open={confirm === "start"}
        onOpenChange={(open) => !open && setConfirm(null)}
        title={t("libraryIndex.confirmTitle")}
        cancelLabel={t("workers.dialog.cancel")}
        confirmLabel={t("libraryIndex.confirm", { count: preview?.new ?? 0 })}
        confirmDisabled={(preview?.new ?? 0) === 0}
        onConfirm={() => void start()}
        busy={busy}
      >
        <p>
          {t("libraryIndex.estimate", {
            detected: preview?.detected ?? 0,
            known: preview?.already_indexed ?? 0,
            fresh: preview?.new ?? 0,
          })}
        </p>
        <p className="text-mute">{t(`libraryIndex.modeHelp.${mode}`)}</p>
        <p className="text-mute">{t("libraryIndex.safety")}</p>
      </ConfirmDialog>
      <ConfirmDialog
        open={confirm === "cancel"}
        onOpenChange={(open) => !open && setConfirm(null)}
        title={t("libraryIndex.cancelTitle")}
        cancelLabel={t("libraryIndex.keep")}
        confirmLabel={t("libraryIndex.cancel")}
        danger
        onConfirm={() => void act("cancel")}
        busy={busy}
      >
        <p>{t("libraryIndex.cancelBody")}</p>
      </ConfirmDialog>
    </section>
  );
}
