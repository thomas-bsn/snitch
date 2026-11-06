import { Check, Copy } from "lucide-react";
import { useEffect, useId, useState } from "react";
import { buttonClass } from "../../components/buttons";
import { useI18n } from "../../i18n";
import { useFormatDate } from "../format";
import type { WorkerDetail } from "./workersApi";
import { hasReturned, recoveryHints, restartCommands } from "./workerView";

/**
 * Diagnose / Reconnect for a Worker that is not revoked. Snitch cannot restart a remote machine:
 * this page gives leads and local commands, and only announces a return when a new pulse arrived
 * (a managed restart is #264).
 */
export function OfflineDiagnostics({ worker }: { worker: WorkerDetail }) {
  const { t } = useI18n();
  const formatDate = useFormatDate();
  const panelId = useId();
  const titleId = useId();
  const [open, setOpen] = useState(false);
  // The pulse seen when the panel opened: a return needs a newer one.
  const [baseline, setBaseline] = useState<string | null>(null);

  if (worker.revoked_at) return null;
  // Online yet no longer claiming (#321): presence is fine, the diagnosis is still needed.
  const stalled = worker.connected && !!worker.claim_stalled;
  if (worker.connected && !stalled && !open) return null;

  const returned = open && hasReturned(worker, baseline);
  const commands = restartCommands(worker.id);

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
