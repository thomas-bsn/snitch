import { useId, useState, type FormEvent } from "react";
import { ErrorAlert } from "../../components/Alert";
import { buttonClass } from "../../components/buttons";
import { useI18n } from "../../i18n";
import { checkInstallationCode } from "../setupApi";
import { StepHeader } from "../StepHeader";

/**
 * Step 1: the installation code. It is checked as soon as it is entered, before the account
 * form, so that a wrong or expired code never costs the password typed afterwards.
 */
export function CodeStep({
  code,
  onCodeChange,
  onValid,
}: {
  code: string;
  onCodeChange: (code: string) => void;
  onValid: () => void;
}) {
  const { t, error } = useI18n();
  const [busy, setBusy] = useState(false);
  const [failure, setFailure] = useState<string>();
  const inputId = useId();
  const errorId = `${inputId}-error`;

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setBusy(true);
    setFailure(undefined);
    const result = await checkInstallationCode(code.trim());
    setBusy(false);
    if (result.ok) onValid();
    else setFailure(result.code);
  }

  const invalid = failure === "setup_closed_or_invalid_code";
  return (
    <form onSubmit={submit} className="flex max-w-xl flex-col gap-5">
      <StepHeader
        position={1}
        total={9}
        title={t("setup.code.title")}
        lead={t("setup.code.lead")}
        focus={false}
      />
      <div className="flex flex-col gap-2 rounded-lg border border-line bg-side p-4">
        <p className="font-medium">{t("setup.code.where")}</p>
        <code className="rounded-md bg-surface px-3 py-2 font-mono text-sm select-all">
          docker compose logs app
        </code>
        <p className="text-sm text-mute">{t("setup.code.whereHint")}</p>
      </div>
      <div className="flex flex-col gap-1.5">
        <label htmlFor={inputId} className="font-medium">
          {t("setup.code.label")}
        </label>
        <input
          id={inputId}
          value={code}
          onChange={(event) => {
            onCodeChange(event.target.value);
            setFailure(undefined);
          }}
          required
          spellCheck={false}
          autoComplete="off"
          aria-invalid={invalid || undefined}
          aria-describedby={invalid ? errorId : undefined}
          className="h-11 rounded-lg border border-line-strong bg-surface px-3 font-mono text-ink md:h-10"
        />
        {invalid && (
          <div id={errorId}>
            <ErrorAlert>{t("setup.code.invalid")}</ErrorAlert>
          </div>
        )}
        {failure && !invalid && <ErrorAlert>{error(failure)}</ErrorAlert>}
      </div>
      <div>
        <button
          type="submit"
          disabled={busy || code.trim() === ""}
          className={buttonClass("primary")}
        >
          {busy ? t("setup.code.checking") : t("setup.continue")}
        </button>
      </div>
    </form>
  );
}
