import { Check, Copy } from "lucide-react";
import { useEffect, useId, useState } from "react";
import { buttonClass } from "../../components/buttons";
import { ConfirmDialog } from "../../components/ConfirmDialog";
import { useI18n } from "../../i18n";
import { useFormatDate } from "../format";
import {
  fetchWorkerRestartHistory,
  useWorkerActions,
  type WorkerDetail,
  type WorkerRestartHistory,
} from "./workersApi";
import { hasReturned, recoveryHints, restartCommands } from "./workerView";

/**
 * Diagnose / Reconnect for a Worker that is not revoked. Snitch cannot restart a remote machine:
 * this page gives leads and local commands, and only announces a return when a new pulse arrived
 * (a managed restart is #264).
 */
export function OfflineDiagnostics({ worker }: { worker: WorkerDetail }) {
  const { t, error } = useI18n();
  const formatDate = useFormatDate();
  const actions = useWorkerActions();
  const panelId = useId();
  const titleId = useId();
  const [open, setOpen] = useState(false);
  // The pulse seen when the panel opened: a return needs a newer one.
  const [baseline, setBaseline] = useState<string | null>(null);
  const [restartHistory, setRestartHistory] = useState<WorkerRestartHistory>();
  const [restartConfirm, setRestartConfirm] = useState(false);
  const [restartReason, setRestartReason] = useState("");
  const [restartBusy, setRestartBusy] = useState(false);
  const [restartError, setRestartError] = useState<string>();

  useEffect(() => {
    if (!open || !worker.managed_restart) return;
    void fetchWorkerRestartHistory(worker.id).then((result) => {
      if (result.ok) setRestartHistory(result.value);
      else setRestartError(result.code);
    });
  }, [open, worker.id, worker.managed_restart]);

  if (worker.revoked_at) return null;
  // Online yet no longer claiming (#321): presence is fine, the diagnosis is still needed.
  const stalled = worker.connected && !!worker.claim_stalled;
  if (worker.connected && !stalled && !open) return null;

  const returned = open && hasReturned(worker, baseline);
  const commands = restartCommands(worker.id);
  const restartStateLabel: Record<string, string> = {
    requested: t("workers.diagnostics.restartState.requested"),
    accepted: t("workers.diagnostics.restartState.accepted"),
    stopping: t("workers.diagnostics.restartState.stopping"),
    starting: t("workers.diagnostics.restartState.starting"),
    online: t("workers.diagnostics.restartState.online"),
    failed: t("workers.diagnostics.restartState.failed"),
    expired: t("workers.diagnostics.restartState.expired"),
  };

  function toggle() {
    if (!open) setBaseline(worker.last_seen_at);
    setOpen(!open);
  }

  return (
    <section
      aria-labelledby={titleId}
      className="flex flex-col gap-3 rounded-xl border border-line p-4"
    >
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h3 id={titleId} className="text-base font-semibold">
          {t("workers.diagnostics.title")}
        </h3>
        <button
          type="button"
          className={buttonClass("secondary")}
          aria-expanded={open}
          aria-controls={panelId}
          onClick={toggle}
        >
          {t("workers.diagnostics.open")}
        </button>
      </div>
      {!open && (
        <p className="text-mute">
          {t(
            stalled
              ? "workers.diagnostics.claimStalledLead"
              : "workers.diagnostics.offlineLead",
          )}
        </p>
      )}
      <div id={panelId} hidden={!open} className="flex flex-col gap-3">
        {open && (
          <>
            <dl className="grid grid-cols-[max-content_1fr] gap-x-6 gap-y-1.5">
              <dt className="text-mute">
                {t("workers.diagnostics.lastPulse")}
              </dt>
              <dd className="m-0">
                {worker.last_seen_at
                  ? formatDate(worker.last_seen_at)
                  : t("workers.never")}
              </dd>
              <dt className="text-mute">
                {t("workers.diagnostics.installation")}
              </dt>
              <dd className="m-0">
                {worker.install
                  ? t(`workers.install.state.${worker.install.state}`)
                  : t("workers.notDeclared")}
              </dd>
            </dl>
            <p className="text-mute">{t("workers.diagnostics.noRemote")}</p>
            {worker.managed_restart && (
              <section className="flex flex-col gap-2 rounded-lg border border-line p-3">
                <h4 className="font-medium">
                  {t("workers.diagnostics.managedTitle")}
                </h4>
                <p className="text-sm text-mute">
                  {restartHistory?.supervisor_available
                    ? t("workers.diagnostics.supervisorOnline", {
                        date: restartHistory.supervisor_last_seen_at
                          ? formatDate(restartHistory.supervisor_last_seen_at)
                          : "",
                      })
                    : t("workers.diagnostics.supervisorOffline")}
                </p>
                {restartHistory?.supervisor_available && (
                  <button
                    type="button"
                    className={buttonClass("secondary")}
                    onClick={() => {
                      setRestartReason("");
                      setRestartError(undefined);
                      setRestartConfirm(true);
                    }}
                  >
                    {t("workers.diagnostics.managedRestart")}
                  </button>
                )}
                {restartError && (
                  <p role="alert" className="text-sm text-bad-fg">
                    {error(restartError)}
                  </p>
                )}
                {restartHistory && restartHistory.items.length > 0 && (
                  <>
                    <h5 className="font-medium">
                      {t("workers.diagnostics.restartHistory")}
                    </h5>
                    <ul className="m-0 flex list-disc flex-col gap-1 pl-5 text-sm">
                      {restartHistory.items.map((item) => (
                        <li key={item.id}>
                          {formatDate(item.created_at)} ·{" "}
                          {restartStateLabel[item.status] ?? item.status} ·{" "}
                          {item.actor}: {item.reason}
                        </li>
                      ))}
                    </ul>
                  </>
                )}
              </section>
            )}
            <h4 className="font-medium">{t("workers.diagnostics.hints")}</h4>
            <ul className="m-0 flex list-disc flex-col gap-1 pl-5">
              {recoveryHints(worker.install).map((hint) => (
                <li key={hint}>{t(`workers.diagnostics.hint.${hint}`)}</li>
              ))}
            </ul>
            <h4 className="font-medium">{t("workers.diagnostics.commands")}</h4>
            <p className="text-mute">{t("workers.diagnostics.commandsLead")}</p>
            <CommandLine
              label={t("workers.diagnostics.restart")}
              hint={t("workers.diagnostics.restartHint")}
              command={commands.restart}
            />
            <CommandLine
              label={t("workers.diagnostics.logs")}
              hint={t("workers.diagnostics.logsHint")}
              command={commands.logs}
            />
            <CommandLine
              label={t("workers.diagnostics.launcher")}
              hint={t("workers.diagnostics.launcherHint")}
              command={commands.launcher}
            />
          </>
        )}
      </div>
      {/* Mounted for good, so a screen reader announces its change when the text arrives. */}
      <p
        role="status"
        hidden={!open}
        className={
          returned
            ? "rounded-lg bg-ok-bg px-3 py-2.5 text-ok-fg"
            : "rounded-lg bg-side px-3 py-2.5 text-mute"
        }
      >
        {returned && worker.last_seen_at
          ? t("workers.diagnostics.returned", {
              date: formatDate(worker.last_seen_at),
            })
          : t("workers.diagnostics.waiting")}
      </p>
      <ConfirmDialog
        open={restartConfirm}
        onOpenChange={(value) => !value && setRestartConfirm(false)}
        title={t("workers.diagnostics.restartConfirmTitle")}
        cancelLabel={t("workers.dialog.cancel")}
        confirmLabel={t("workers.diagnostics.managedRestart")}
        confirmDisabled={!restartReason.trim()}
        busy={restartBusy}
        onConfirm={() => {
          setRestartBusy(true);
          void actions
            .restart(worker.id, restartReason)
            .then(async (result) => {
              setRestartBusy(false);
              if (!result.ok) {
                setRestartError(result.code);
                return;
              }
              setRestartConfirm(false);
              setRestartReason("");
              const history = await fetchWorkerRestartHistory(worker.id);
              if (history.ok) setRestartHistory(history.value);
            });
        }}
      >
        <p>{t("workers.diagnostics.restartConfirmBody")}</p>
        <label className="mt-2 flex flex-col gap-1 text-sm">
          {t("workers.diagnostics.restartReason")}
          <input
            value={restartReason}
            maxLength={500}
            onChange={(event) => setRestartReason(event.target.value)}
            className="rounded-lg border border-line-strong bg-side px-3 py-2 text-ink"
          />
        </label>
      </ConfirmDialog>
    </section>
  );
}

export function CommandLine({
  label,
  hint,
  command,
}: {
  label: string;
  hint: string;
  command: string;
}) {
  const { t } = useI18n();
  const [copied, setCopied] = useState(false);
  const [failed, setFailed] = useState(false);
  // The confirmation fades after 2 s; leaving the page cancels it.
  useEffect(() => {
    if (!copied) return;
    const timer = setTimeout(() => setCopied(false), 2000);
    return () => clearTimeout(timer);
  }, [copied]);
  async function copy() {
    try {
      await navigator.clipboard.writeText(command);
      setFailed(false);
      setCopied(true);
    } catch {
      setFailed(true);
    }
  }
  return (
    <div className="flex flex-col gap-1">
      <p className="font-medium">{label}</p>
      <p className="text-sm text-mute">{hint}</p>
      <div className="flex flex-wrap items-center gap-2">
        <code className="min-w-0 flex-1 break-all rounded-lg border border-line-strong bg-side px-3 py-2 font-mono text-xs text-ink">
          {command}
        </code>
        <button
          type="button"
          className={buttonClass("secondary")}
          onClick={copy}
          aria-label={t("workers.diagnostics.copy", { command: label })}
        >
          {copied ? (
            <Check className="size-4" aria-hidden="true" />
          ) : (
            <Copy className="size-4" aria-hidden="true" />
          )}
          {copied
            ? t("workers.diagnostics.copied")
            : t("workers.diagnostics.copyShort")}
        </button>
      </div>
      {failed && (
        <p role="alert" className="text-sm text-bad-fg">
          {t("workers.diagnostics.copyFailed")}
        </p>
      )}
    </div>
  );
}
