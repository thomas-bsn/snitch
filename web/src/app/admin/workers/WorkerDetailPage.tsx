import { ArrowLeft } from "lucide-react";
import { useCallback, useId, useState, type ReactNode } from "react";
import { Link, useNavigate } from "react-router";
import { useLoaded } from "../../api/loaded";
import { useEventStream } from "../../api/useEventStream";
import { ErrorAlert } from "../../components/Alert";
import { buttonClass } from "../../components/buttons";
import { ConfirmDialog } from "../../components/ConfirmDialog";
import { StatusPill } from "../../components/StatusPill";
import { useDocumentTitle } from "../../components/useDocumentTitle";
import { useI18n } from "../../i18n";
import { useFormatDate } from "../format";
import { DeploymentFiles } from "./DeploymentFiles";
import { InstallProgress } from "./InstallProgress";
import { InstallGuide } from "./InstallGuide";
import { CommandLine, OfflineDiagnostics } from "./OfflineDiagnostics";
import {
  fetchWorker,
  useWorkerActions,
  type WorkerDetail,
  type WorkerRegistration,
} from "./workersApi";
import {
  ROLES,
  declaredModelText,
  restartCommands,
  workerStatus,
} from "./workerView";

function Section({ title, children }: { title: string; children: ReactNode }) {
  const id = useId();
  return (
    <section
      aria-labelledby={id}
      className="flex flex-col gap-3 rounded-xl border border-line p-4"
    >
      <h3 id={id} className="text-base font-semibold">
        {title}
      </h3>
      {children}
    </section>
  );
}

function Facts({ children }: { children: ReactNode }) {
  return (
    <dl className="grid grid-cols-[max-content_1fr] gap-x-6 gap-y-1.5">
      {children}
    </dl>
  );
}

function Fact({ label, children }: { label: string; children: ReactNode }) {
  return (
    <>
      <dt className="text-mute">{label}</dt>
      <dd className="m-0 break-words">{children}</dd>
    </>
  );
}

/** The page of one Worker: state, activity, declared models, deployment files, token, revocation. */
export function WorkerDetailPage({ id }: { id: string }) {
  const { t, error } = useI18n();
  const formatDate = useFormatDate();
  const actions = useWorkerActions();
  const navigate = useNavigate();
  const load = useCallback(() => fetchWorker(id), [id]);
  const { state, reload, set } = useLoaded(load);
  // Live: the page follows the stream; the reload is the fallback while the stream is down.
  useEventStream<WorkerDetail>(`/api/admin/workers/${id}/events`, set, reload);
  const [confirm, setConfirm] = useState<"token" | "revoke" | "forget" | null>(
    null,
  );
  const [reason, setReason] = useState("");
  const [allowInsecure, setAllowInsecure] = useState(false);
  const insecure = window.location.protocol !== "https:";
  const [busy, setBusy] = useState(false);
  const [failure, setFailure] = useState<string>();
  // Shown once: the replacement identity and its single-use command, dropped when the page is left.
  const [fresh, setFresh] = useState<WorkerRegistration>();
  useDocumentTitle(
    state.status === "ready" ? state.value.name : t("workers.title"),
  );

  const back = (
    <Link
      to="/admin/workers"
      className="inline-flex min-h-11 items-center gap-1.5 text-accent-ink hover:underline md:min-h-0"
    >
      <ArrowLeft className="size-4" />
      {t("workers.detail.back")}
    </Link>
  );

  if (state.status === "loading") {
    return <p role="status">{t("workers.loading")}</p>;
  }
  if (state.status === "failed") {
    return (
      <div className="flex flex-col gap-3">
        {back}
        <ErrorAlert>
          {state.code === "worker_not_found"
            ? t("workers.detail.notFound")
            : error(state.code)}
        </ErrorAlert>
      </div>
    );
  }
  const worker = state.value;
  const { tone, label } = workerStatus(worker);
  const revoked = worker.revoked_at !== null;
  // The local launcher exists only once the installation reached its first pulse.
  const installed =
    worker.install?.state === "connected" ||
    worker.install?.state === "voice_ready";

  async function renew() {
    setBusy(true);
    setFailure(undefined);
    const result = await actions.renewToken(id, insecure && allowInsecure);
    setBusy(false);
    setConfirm(null);
    if (result.ok) {
      setFresh(result.value);
      reload();
    } else setFailure(result.code);
  }

  async function forget() {
    setBusy(true);
    setFailure(undefined);
    const result = await actions.forget(id);
    setBusy(false);
    setConfirm(null);
    if (result.ok) navigate("/admin/workers");
    else setFailure(result.code);
  }

  async function revoke() {
    setBusy(true);
    setFailure(undefined);
    const result = await actions.revoke(id, reason);
    setBusy(false);
    setConfirm(null);
    if (result.ok) {
      setFresh(undefined);
      reload();
    } else setFailure(result.code);
  }

  return (
    <div className="flex flex-col gap-4">
      {back}
      <div className="flex flex-wrap items-center gap-3">
        <h2 className="text-lg font-semibold">{worker.name}</h2>
        <StatusPill tone={tone}>{t(label)}</StatusPill>
      </div>
      {failure && <ErrorAlert>{error(failure)}</ErrorAlert>}
      {revoked && worker.revoked_at && (
        <div
          role="status"
          className="flex flex-col gap-1 rounded-lg bg-bad-bg px-3 py-2.5 text-bad-fg"
        >
          <p>
            {t("workers.detail.revokedNotice", {
              date: formatDate(worker.revoked_at),
            })}
          </p>
          {worker.revoke_reason && (
            <p>
              {t("workers.detail.revokedReason", {
                reason: worker.revoke_reason,
              })}
            </p>
          )}
          <p>
            {worker.revocation_acknowledged_at
              ? t("workers.detail.cleanupDone", {
                  date: formatDate(worker.revocation_acknowledged_at),
                })
              : t("workers.detail.cleanupPending")}
          </p>
        </div>
      )}

      {worker.install && (
        <Section title={t("workers.install.title")}>
          <InstallProgress
            install={worker.install}
            transcriptionConfig={worker.transcription_config}
            createdAt={worker.created_at}
          />
        </Section>
      )}

      <OfflineDiagnostics worker={worker} />

      <Section title={t("workers.detail.activity")}>
        <p className="text-sm text-mute">{t("workers.connection.hint")}</p>
        <Facts>
          <Fact label={t("workers.detail.lastPulse")}>
            {worker.last_seen_at
              ? formatDate(worker.last_seen_at)
              : t("workers.never")}
          </Fact>
          <Fact label={t("workers.detail.concurrency")}>
            {t("workers.detail.concurrencyValue", {
              count: worker.max_concurrency,
            })}
          </Fact>
          <Fact label={t("workers.detail.created")}>
            {formatDate(worker.created_at)}
          </Fact>
        </Facts>
      </Section>

      <Section title={t("workers.detail.recentTasks")}>
        {worker.recent_tasks.length === 0 ? (
          <p className="text-mute">{t("workers.detail.noTasks")}</p>
        ) : (
          <div
            role="region"
            tabIndex={0}
            aria-label={t("workers.detail.recentTasks")}
            className="overflow-x-auto"
          >
            <table className="w-full min-w-[28rem] border-collapse text-left">
              <thead className="text-xs text-mute">
                <tr>
                  <th scope="col" className="py-1.5 pr-3 font-medium">
                    {t("workers.detail.taskColumns.kind")}
                  </th>
                  <th scope="col" className="py-1.5 pr-3 font-medium">
                    {t("workers.detail.taskColumns.clip")}
                  </th>
                  <th scope="col" className="py-1.5 pr-3 font-medium">
                    {t("workers.detail.taskColumns.outcome")}
                  </th>
                  <th scope="col" className="py-1.5 font-medium">
                    {t("workers.detail.taskColumns.when")}
                  </th>
                </tr>
              </thead>
              <tbody>
                {worker.recent_tasks.map((task) => (
                  <tr key={task.job_id} className="border-t border-line">
                    <th scope="row" className="py-2 pr-3 font-medium">
                      {t(`workers.detail.taskKinds.${task.kind}`)}
                    </th>
                    <td className="py-2 pr-3 text-mute">#{task.clip_id}</td>
                    <td className="py-2 pr-3">
                      {t(`workers.detail.outcomes.${task.outcome}`)}
                      {task.error_code && (
                        <span className="ml-1.5 text-mute">
                          ({task.error_code})
                        </span>
                      )}
                    </td>
                    <td className="py-2 text-mute">{formatDate(task.at)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </Section>

      <Section title={t("workers.detail.models")}>
        <Facts>
          {ROLES.map((role) => (
            <Fact key={role} label={t(`workers.models.${role}`)}>
              {declaredModelText(worker.models, role) ??
                t("workers.notDeclared")}
            </Fact>
          ))}
        </Facts>
      </Section>

      <Section title={t("workers.machine.title")}>
        <Facts>
          <Fact label={t("workers.machine.name")}>
            {worker.machine?.name || t("workers.notDeclared")}
          </Fact>
          <Fact label={t("workers.machine.system")}>
            {worker.machine?.system || t("workers.notDeclared")}
          </Fact>
          <Fact label={t("workers.machine.processor")}>
            {worker.machine?.processor || t("workers.notDeclared")}
          </Fact>
          <Fact label={t("workers.machine.memory")}>
            {worker.machine?.memory_bytes
              ? `${(worker.machine.memory_bytes / 1024 ** 3).toFixed(1)} GiB`
              : t("workers.notDeclared")}
          </Fact>
          <Fact label={t("workers.machine.gpu")}>
            {worker.machine?.gpu_name || t("workers.notDeclared")}
          </Fact>
          <Fact label={t("workers.machine.gpuMemory")}>
            {worker.machine?.gpu_memory_bytes
              ? `${(worker.machine.gpu_memory_bytes / 1024 ** 3).toFixed(1)} GiB`
              : t("workers.notDeclared")}
          </Fact>
          <Fact label={t("workers.machine.driver")}>
            {worker.machine?.gpu_driver || t("workers.notDeclared")}
          </Fact>
          <Fact label={t("workers.machine.version")}>
            {worker.machine?.worker_version || t("workers.notDeclared")}
          </Fact>
          <Fact label={t("workers.machine.address")}>
            {worker.remote_address || t("workers.notDeclared")}
          </Fact>
        </Facts>
        <p className="text-sm text-mute">{t("workers.machine.systemHint")}</p>
      </Section>

      {fresh && (
        <Section title={t("workers.detail.replacedTitle")}>
          <p
            role="status"
            className="rounded-lg bg-warn-bg px-3 py-2.5 text-warn-fg"
          >
            {t("workers.detail.newToken")}
          </p>
          {!fresh.install_command && (
            <ErrorAlert>{t("workers.create.insecureUnavailable")}</ErrorAlert>
          )}
          <InstallGuide />
          <textarea
            readOnly
            aria-label={t("workers.create.installCommand")}
            value={fresh.install_command ?? ""}
            rows={4}
            className="w-full resize-y rounded-lg border border-line-strong bg-side p-3 font-mono text-xs text-ink"
          />
          <Link
            to={`/admin/workers/${fresh.id}`}
            className="text-accent-ink underline"
          >
            {t("workers.detail.openReplacement")}
          </Link>
        </Section>
      )}

      {!revoked && (
        <Section title={t("workers.detail.oneCommand")}>
          <p className="text-mute">{t("workers.detail.oneCommandLead")}</p>
          {installed && (
            <>
              <h4 className="font-medium">
                {t("workers.detail.alreadyInstalledTitle")}
              </h4>
              <p className="text-mute">
                {t("workers.detail.alreadyInstalledBody")}
              </p>
              <CommandLine
                label={t("workers.diagnostics.launcher")}
                hint={t("workers.diagnostics.launcherHint")}
                command={restartCommands(worker.id).launcher}
              />
            </>
          )}
          <h4 className="font-medium">{t("workers.detail.newMachineTitle")}</h4>
          <p className="text-mute">{t("workers.detail.newMachineBody")}</p>
          <div>
            <button
              type="button"
              className={buttonClass("secondary")}
              onClick={() => setConfirm("token")}
            >
              {t("workers.detail.regenerate")}
            </button>
          </div>
          <details>
            <summary className="cursor-pointer text-accent-ink">
              {t("workers.create.advancedFiles")}
            </summary>
            <div className="mt-3 flex flex-col gap-3">
              <p className="text-mute">{t("workers.detail.deploymentLead")}</p>
              <p className="text-mute">{t("workers.detail.tokenHidden")}</p>
              <DeploymentFiles
                env={worker.bundle.env}
                compose={worker.bundle.compose}
              />
            </div>
          </details>
        </Section>
      )}

      {revoked && worker.revocation_acknowledged_at && (
        <Section title={t("workers.detail.forget")}>
          <p className="text-mute">{t("workers.detail.forgetBody")}</p>
          <div>
            <button
              type="button"
              className={buttonClass("danger")}
              onClick={() => setConfirm("forget")}
            >
              {t("workers.detail.forget")}
            </button>
          </div>
        </Section>
      )}

      {!revoked && (
        <Section title={t("workers.detail.revoke")}>
          <p className="text-mute">{t("workers.detail.revokeBody")}</p>
          <div>
            <button
              type="button"
              className={buttonClass("danger")}
              onClick={() => setConfirm("revoke")}
            >
              {t("workers.detail.revokeButton")}
            </button>
          </div>
        </Section>
      )}

      <ConfirmDialog
        open={confirm === "token"}
        onOpenChange={(open) => !open && setConfirm(null)}
        title={t("workers.detail.regenerateTitle")}
        cancelLabel={t("workers.dialog.cancel")}
        confirmLabel={t("workers.detail.regenerateConfirm")}
        onConfirm={() => void renew()}
        busy={busy}
      >
        <p>{t("workers.detail.regenerateBody")}</p>
        {insecure && (
          <>
            <label className="mt-3 flex items-start gap-2">
              <input
                type="checkbox"
                className="mt-1"
                checked={allowInsecure}
                onChange={(event) => setAllowInsecure(event.target.checked)}
              />
              <span>{t("workers.create.allowInsecure")}</span>
            </label>
            {allowInsecure && (
              <p
                role="alert"
                className="mt-2 rounded-lg bg-bad-bg px-3 py-2.5 text-bad-fg"
              >
                {t("workers.create.allowInsecureWarning")}
              </p>
            )}
          </>
        )}
      </ConfirmDialog>
      <ConfirmDialog
        open={confirm === "forget"}
        onOpenChange={(open) => !open && setConfirm(null)}
        title={t("workers.detail.forgetTitle", { name: worker.name })}
        cancelLabel={t("workers.dialog.cancel")}
        confirmLabel={t("workers.detail.forgetConfirm")}
        onConfirm={() => void forget()}
        danger
        busy={busy}
      >
        <p>{t("workers.detail.forgetBody")}</p>
      </ConfirmDialog>
      <ConfirmDialog
        open={confirm === "revoke"}
        onOpenChange={(open) => !open && setConfirm(null)}
        title={t("workers.detail.revokeTitle", { name: worker.name })}
        cancelLabel={t("workers.dialog.cancel")}
        confirmLabel={t("workers.detail.revokeConfirm")}
        onConfirm={() => void revoke()}
        danger
        busy={busy}
      >
        <p>{t("workers.detail.revokeConfirmBody")}</p>
        <label className="mt-3 flex flex-col gap-1 text-sm">
          {t("workers.detail.revokeReason")}
          <input
            type="text"
            maxLength={200}
            value={reason}
            onChange={(event) => setReason(event.target.value)}
            className="rounded-lg border border-line-strong bg-side px-3 py-2 text-ink"
          />
        </label>
      </ConfirmDialog>
    </div>
  );
}
