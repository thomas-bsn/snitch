import { TriangleAlert } from "lucide-react";
import { useState, type FormEvent, type ReactNode } from "react";
import { Link } from "react-router";
import { useLoaded } from "../../api/loaded";
import { ErrorAlert, StatusNotice } from "../../components/Alert";
import { buttonClass } from "../../components/buttons";
import { Field } from "../../components/Field";
import { useI18n } from "../../i18n";
import { whisperLanguageLabel } from "../transcription/whisperLanguages";
import { DeploymentFiles } from "./DeploymentFiles";
import { InstallGuide } from "./InstallGuide";
import {
  fetchWorkers,
  useWorkerActions,
  type WorkerRegistration,
} from "./workersApi";

type WorkerCreateFormProps = {
  onDone: (worker: WorkerRegistration) => void;
  secondaryAction?: ReactNode;
};

/** The library's configuration belongs to Transcription, even for the first Worker. */
export function WorkerCreateForm({
  onDone,
  secondaryAction,
}: WorkerCreateFormProps) {
  const { t, error, language } = useI18n();
  const actions = useWorkerActions();
  const { state } = useLoaded(fetchWorkers);
  const [name, setName] = useState<string>();
  const [busy, setBusy] = useState(false);
  const [failure, setFailure] = useState<string>();
  const [created, setCreated] = useState<WorkerRegistration>();
  const [copied, setCopied] = useState(false);
  const [allowInsecure, setAllowInsecure] = useState(false);
  const [managedRestart, setManagedRestart] = useState(false);

  if (state.status === "loading")
    return <p role="status">{t("workers.loading")}</p>;
  if (state.status === "failed")
    return <ErrorAlert>{error(state.code)}</ErrorAlert>;

  const page = state.value;
  const profile = page.transcription;
  const enabled = profile.mode === "enabled";
  const insecure = !page.snitch_url.toLowerCase().startsWith("https://");

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setBusy(true);
    setFailure(undefined);
    const result = await actions.create({
      name: (name ?? `worker-${page.items.length + 1}`).trim(),
      allowInsecureInstall: insecure && allowInsecure,
      managedRestart,
    });
    setBusy(false);
    if (result.ok) setCreated(result.value);
    else setFailure(result.code);
  }

  if (created) {
    return (
      <div className="flex max-w-xl flex-col gap-4">
        <StatusNotice>{t("workers.create.shownOnce")}</StatusNotice>
        {!created.install_command && (
          <ErrorAlert>{t("workers.create.insecureUnavailable")}</ErrorAlert>
        )}
        <h3 className="text-base font-semibold">{t("workers.guide.title")}</h3>
        <p>{t("workers.create.installLead")}</p>
        <InstallGuide />
        <textarea
          readOnly
          aria-label={t("workers.create.installCommand")}
          value={created.install_command ?? ""}
          rows={4}
          className="w-full resize-y rounded-lg border border-line-strong bg-side p-3 font-mono text-xs text-ink"
        />
        <div className="flex flex-wrap gap-2">
          <button
            type="button"
            className={buttonClass("secondary")}
            onClick={async () => {
              try {
                await navigator.clipboard.writeText(
                  created.install_command ?? "",
                );
                setCopied(true);
              } catch {
                setFailure("clipboard_unavailable");
              }
            }}
          >
            {copied
              ? t("workers.files.copied")
              : t("workers.create.copyCommand")}
          </button>
          <button
            type="button"
            className={buttonClass("primary")}
            onClick={() => onDone(created)}
          >
            {t("workers.create.done")}
          </button>
        </div>
        {failure && <ErrorAlert>{error(failure)}</ErrorAlert>}
        <details>
          <summary>{t("workers.create.advancedFiles")}</summary>
          <DeploymentFiles
            env={created.bundle.env}
            compose={created.bundle.compose}
          />
        </details>
      </div>
    );
  }

  return (
    <form
      onSubmit={(event) => void submit(event)}
      className="flex max-w-xl flex-col gap-4"
    >
      {failure && <ErrorAlert>{error(failure)}</ErrorAlert>}
      <Field
        label={t("workers.create.name")}
        hint={t("workers.create.nameHint")}
        value={name ?? `worker-${page.items.length + 1}`}
        onChange={(event) => setName(event.target.value)}
        required
        maxLength={64}
        autoComplete="off"
      />
      <label className="flex items-start gap-2 rounded-lg border border-line p-3">
        <input
          type="checkbox"
          className="mt-1"
          checked={managedRestart}
          onChange={(event) => setManagedRestart(event.target.checked)}
        />
        <span>
          <span className="block font-medium">{t("workers.create.managedRestart")}</span>
          <span className="block text-sm text-mute">{t("workers.create.managedRestartHint")}</span>
        </span>
      </label>
      <Field
        label={t("workers.create.snitch")}
        hint={t("workers.create.snitchHint")}
        value={page.snitch_url}
        readOnly
      />
      <a
        className="text-sm text-accent underline"
        href="https://github.com/thomas-bsn/snitch/blob/main/docs/access.md#adresse-publique"
        target="_blank"
        rel="noreferrer"
      >
        {t("workers.create.publicAddressLink")}
      </a>
      {insecure && (
        <p
          role="note"
          className="flex items-start gap-2 rounded-lg bg-warn-bg px-3 py-2.5 text-warn-fg"
        >
          <TriangleAlert className="mt-0.5 size-4 shrink-0" />
          {t("workers.create.insecure")}
        </p>
      )}
      {insecure && (
        <label className="flex items-start gap-2">
          <input
            type="checkbox"
            className="mt-1"
            checked={allowInsecure}
            onChange={(event) => setAllowInsecure(event.target.checked)}
          />
          <span>{t("workers.create.allowInsecure")}</span>
        </label>
      )}
      {insecure && allowInsecure && (
        <p
          role="alert"
          className="flex items-start gap-2 rounded-lg bg-bad-bg px-3 py-2.5 text-bad-fg"
        >
          <TriangleAlert className="mt-0.5 size-4 shrink-0" />
          {t("workers.create.allowInsecureWarning")}
        </p>
      )}
      <Field
        label={t("workers.create.whisper")}
        hint={t("workers.create.whisperHint")}
        value={enabled ? profile.endpoint : ""}
        readOnly
      />
      <Field
        label={t("workers.create.libraryModel")}
        hint={t("workers.create.libraryModelHint")}
        value={enabled ? profile.model : ""}
        readOnly
      />
      <Field
        label={t("workers.create.language")}
        hint={t("workers.create.languageHint")}
        value={enabled ? whisperLanguageLabel(profile.language, language) : ""}
        readOnly
      />
      <Link className="text-sm text-accent underline" to="/admin/transcription">
        {t("workers.create.editTranscription")}
      </Link>
      {!enabled && (
        <StatusNotice>
          {profile.mode === "disabled"
            ? t("workers.create.transcriptionDisabled")
            : t("workers.create.transcriptionUnconfigured")}
        </StatusNotice>
      )}
      <p className="text-sm text-mute">{t("workers.create.hfTerminalHint")}</p>
      <div className="flex flex-wrap gap-2">
        <button
          type="submit"
          disabled={busy}
          className={buttonClass("primary")}
        >
          {busy ? t("workers.create.submitting") : t("workers.create.submit")}
        </button>
        {secondaryAction}
      </div>
    </form>
  );
}
