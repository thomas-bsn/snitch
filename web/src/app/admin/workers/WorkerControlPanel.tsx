import { useCallback, useEffect, useState } from "react";
import { ErrorAlert } from "../../components/Alert";
import { buttonClass } from "../../components/buttons";
import { ConfirmDialog } from "../../components/ConfirmDialog";
import { useI18n } from "../../i18n";
import {
  fetchWorkerControl,
  useWorkerActions,
  type WorkerControlAction,
  type WorkerControlState,
} from "./workersApi";

/** How often the state is read again while an instruction waits for the Worker. */
export const CONTROL_INTERVAL_MS = 2000;

export type ControlPhase = "running" | "pausing" | "paused" | "resuming";

/** What the Worker is doing, from what was asked and what it confirmed. */
export function controlPhase(state: WorkerControlState): ControlPhase {
  const applied = state.applied_paused === true;
  if (state.paused) return applied ? "paused" : "pausing";
  return applied ? "resuming" : "running";
}

/**
 * Pause and resume a Worker. Nothing is sent to it: it reads the state at its next check. Restarting, stopping
 * or starting its container is the managed restart of the connection and recovery section, not repeated here.
 */
export function WorkerControlPanel({
  id,
  managedRestart,
}: {
  id: string;
  managedRestart: boolean;
}) {
  const { t, error } = useI18n();
  const actions = useWorkerActions();
  const [state, setState] = useState<WorkerControlState>();
  const [failure, setFailure] = useState<string>();
  const [busy, setBusy] = useState(false);
  const [confirm, setConfirm] = useState(false);

  const read = useCallback(async () => {
    const result = await fetchWorkerControl(id);
    if (result.ok) {
      setState(result.value);
      setFailure(undefined);
    } else setFailure(result.code);
  }, [id]);

  useEffect(() => {
    void read();
  }, [read]);

  const phase = state ? controlPhase(state) : undefined;
  const waiting = phase === "pausing" || phase === "resuming";

  // Read again while the Worker has not answered: one timer, gone as soon as nothing waits or the page is left.
  useEffect(() => {
    if (!waiting) return;
    const timer = setInterval(() => {
      if (!document.hidden) void read();
    }, CONTROL_INTERVAL_MS);
    return () => clearInterval(timer);
  }, [waiting, read]);

  async function send(action: WorkerControlAction) {
    setBusy(true);
    setFailure(undefined);
    const result = await actions.control(id, action);
    setBusy(false);
    setConfirm(false);
    if (result.ok) setState(result.value);
    else setFailure(result.code);
  }

  return (
    <div className="flex flex-col gap-3">
      <p className="text-sm text-mute">{t("workers.control.lead")}</p>
      {failure && <ErrorAlert>{error(failure)}</ErrorAlert>}
      {phase && (
        <div className="flex flex-wrap items-center gap-x-3 gap-y-1">
          <span className="text-mute">{t("workers.control.stateLabel")}</span>
          {/* The state is announced when it changes; the log lines are not. */}
          <span role="status" className="font-medium">
            {t(`workers.control.state.${phase}`)}
          </span>
          {waiting && (
            <span className="text-sm text-mute">
              {t("workers.control.pending")}
            </span>
          )}
        </div>
      )}
      {state && (
        <div>
          {state.paused ? (
            <button
              type="button"
              className={buttonClass("secondary")}
              disabled={busy}
              onClick={() => void send("resume")}
            >
              {t("workers.control.resume")}
            </button>
          ) : (
            <button
              type="button"
              className={buttonClass("secondary")}
              disabled={busy}
              onClick={() => setConfirm(true)}
            >
              {t("workers.control.pause")}
            </button>
          )}
        </div>
      )}
      <p className="text-sm text-mute">
        {managedRestart
          ? t("workers.control.restartManaged")
          : t("workers.control.restartNotManaged")}
      </p>
      <ConfirmDialog
        open={confirm}
        onOpenChange={(open) => !open && setConfirm(false)}
        title={t("workers.control.pauseTitle")}
        cancelLabel={t("workers.dialog.cancel")}
        confirmLabel={t("workers.control.pauseConfirm")}
        onConfirm={() => void send("pause")}
        busy={busy}
      >
        <p>{t("workers.control.pauseBody")}</p>
      </ConfirmDialog>
    </div>
  );
}
