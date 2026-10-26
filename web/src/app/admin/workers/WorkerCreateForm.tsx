import { TriangleAlert } from "lucide-react";
import {
  useId,
  useMemo,
  useState,
  type FormEvent,
  type ReactNode,
} from "react";
import { ErrorAlert, StatusNotice } from "../../components/Alert";
import { buttonClass } from "../../components/buttons";
import { ConfirmDialog } from "../../components/ConfirmDialog";
import { Field } from "../../components/Field";
import { useI18n } from "../../i18n";
import { DeploymentFiles } from "./DeploymentFiles";
import { placeholdersLeft, withHuggingFaceToken } from "./envFile";
import {
  fetchWorkers,
  useLoaded,
  useWorkerActions,
  type WorkerRegistration,
} from "./workersApi";

/** Languages Whisper handles best; the library has exactly one. */
const LANGUAGES = [
  "fr",
  "en",
  "es",
  "de",
  "it",
  "pt",
  "nl",
  "pl",
  "ru",
  "uk",
  "tr",
  "ar",
  "ja",
  "ko",
  "zh",
];

type WorkerCreateFormProps = {
  /** Called once the person has copied the generated files. */
  onDone: (worker: WorkerRegistration) => void;
  /** Extra actions next to the submit button (the wizard adds "Skip"). */
  secondaryAction?: ReactNode;
};

/**
 * The form that creates a Worker and hands over its deployment files. It stands alone: the
 * Workers screen and the installation wizard both use it.
 *
 * The Hugging Face token only feeds the .env text shown here, in this browser. It is never
 * part of a request to Snitch and is dropped with the component.
 */
export function WorkerCreateForm({
  onDone,
  secondaryAction,
}: WorkerCreateFormProps) {
  const { t, error, language } = useI18n();
  const actions = useWorkerActions();
  const { state } = useLoaded(fetchWorkers);
  const [name, setName] = useState<string>();
  const [whisperUrl, setWhisperUrl] = useState("");
  const [model, setModel] = useState("");
  const [libraryLanguage, setLibraryLanguage] = useState("fr");
  const [huggingFaceToken, setHuggingFaceToken] = useState("");
  const [confirming, setConfirming] = useState(false);
  const [busy, setBusy] = useState(false);
  const [failure, setFailure] = useState<string>();
  const [created, setCreated] = useState<WorkerRegistration>();
  const languageId = useId();
  const languageHintId = `${languageId}-hint`;

  const languageNames = useMemo(() => {
    const names = new Intl.DisplayNames([language], { type: "language" });
    return LANGUAGES.map((code) => ({
      code,
      label: `${names.of(code) ?? code} (${code})`,
    }));
  }, [language]);

  if (state.status === "loading") {
    return <p role="status">{t("workers.loading")}</p>;
  }
  if (state.status === "failed") {
    return <ErrorAlert>{error(state.code)}</ErrorAlert>;
  }
  const page = state.value;
  const first = page.first_worker;
  const insecure = !page.snitch_url.toLowerCase().startsWith("https://");

  async function create() {
    setConfirming(false);
    setBusy(true);
    setFailure(undefined);
    const result = await actions.create({
      name: (name ?? `worker-${page.items.length + 1}`).trim(),
      whisperUrl: whisperUrl.trim(),
      transcription: first
        ? { model: model.trim(), language: libraryLanguage }
        : undefined,
    });
    setBusy(false);
    if (result.ok) setCreated(result.value);
    else setFailure(result.code);
  }

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    // The first Worker engages the whole library: ask before going further.
    if (first) setConfirming(true);
    else void create();
  }

  if (created) {
    const env = withHuggingFaceToken(created.bundle.env, huggingFaceToken);
    const missing = placeholdersLeft(env);
    return (
      <div className="flex flex-col gap-4">
        <p
          role="status"
          className="rounded-lg bg-warn-bg px-3 py-2.5 text-warn-fg"
        >
          {t("workers.create.shownOnce")}
        </p>
        {missing.includes("HF_TOKEN") && (
          <StatusNotice>{t("workers.create.hfMissing")}</StatusNotice>
        )}
        {missing.includes("TRANSCRIPTION_API_URL") && (
          <StatusNotice>{t("workers.create.whisperMissing")}</StatusNotice>
        )}
        <p className="text-mute">{t("workers.create.secret")}</p>
        <DeploymentFiles env={env} compose={created.bundle.compose} />
        <div>
          <button
            type="button"
            className={buttonClass("primary")}
            onClick={() => onDone(created)}
          >
            {t("workers.create.done")}
          </button>
        </div>
      </div>
    );
  }

  return (
    <form onSubmit={submit} className="flex max-w-xl flex-col gap-4">
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
      <Field
        label={t("workers.create.snitch")}
        hint={t("workers.create.snitchHint")}
        value={page.snitch_url}
        readOnly
      />
      {insecure && (
        <p
          role="note"
          className="flex items-start gap-2 rounded-lg bg-warn-bg px-3 py-2.5 text-warn-fg"
        >
          <TriangleAlert className="mt-0.5 size-4 shrink-0" />
          {t("workers.create.insecure")}
        </p>
      )}
      <Field
        label={t("workers.create.whisper")}
        hint={t("workers.create.whisperHint")}
        value={whisperUrl}
        onChange={(event) => setWhisperUrl(event.target.value)}
        placeholder="http://host.docker.internal:8000"
        required
        inputMode="url"
        autoComplete="off"
      />
      {first ? (
        <>
          <Field
            label={t("workers.create.model")}
            hint={t("workers.create.modelHint")}
            value={model}
            onChange={(event) => setModel(event.target.value)}
            placeholder="whisper-large-v3"
            required
            autoComplete="off"
          />
          <div className="flex flex-col gap-1.5">
            <label htmlFor={languageId} className="font-medium">
              {t("workers.create.language")}
            </label>
            <select
              id={languageId}
              aria-describedby={languageHintId}
              value={libraryLanguage}
              onChange={(event) => setLibraryLanguage(event.target.value)}
              className="h-11 rounded-lg border border-line-strong bg-surface px-3 text-ink md:h-10"
            >
              {languageNames.map((option) => (
                <option key={option.code} value={option.code}>
                  {option.label}
                </option>
              ))}
            </select>
            <p id={languageHintId} className="text-xs text-mute">
              {t("workers.create.languageHint")}
            </p>
          </div>
        </>
      ) : (
        <Field
          label={t("workers.create.libraryModel")}
          hint={t("workers.create.libraryModelHint")}
          value={`${page.transcription.model} · ${page.transcription.language}`}
          readOnly
        />
      )}
      <Field
        type="password"
        label={t("workers.create.hf")}
        hint={t("workers.create.hfHint")}
        value={huggingFaceToken}
        onChange={(event) => setHuggingFaceToken(event.target.value)}
        placeholder="hf_…"
        autoComplete="off"
        name="huggingface-token-kept-in-the-browser"
      />
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
      <ConfirmDialog
        open={confirming}
        onOpenChange={setConfirming}
        title={t("workers.create.confirmTitle")}
        cancelLabel={t("workers.create.confirmBack")}
        confirmLabel={t("workers.create.confirmSubmit")}
        onConfirm={() => void create()}
      >
        <p className="font-medium text-ink">
          {model.trim()} · {libraryLanguage}
        </p>
        <p>{t("workers.create.confirmBody")}</p>
      </ConfirmDialog>
    </form>
  );
}
