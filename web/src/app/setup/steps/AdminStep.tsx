import { useState, type FormEvent } from "react";
import { ErrorAlert } from "../../components/Alert";
import { buttonClass } from "../../components/buttons";
import { Field } from "../../components/Field";
import { useI18n } from "../../i18n";
import { StepHeader } from "../StepHeader";

export type AccountDraft = {
  username: string;
  password: string;
  confirmation: string;
};

/**
 * Step 2: the first account, which becomes the Super-administrator. What is typed is kept by
 * the wizard, so that going back to the code or a failed attempt loses nothing.
 */
export function AdminStep({
  draft,
  onChange,
  onBack,
  onSubmit,
}: {
  draft: AccountDraft;
  onChange: (draft: AccountDraft) => void;
  onBack: () => void;
  onSubmit: () => Promise<string | undefined>;
}) {
  const { t, error } = useI18n();
  const [busy, setBusy] = useState(false);
  const [failure, setFailure] = useState<string>();
  const mismatch =
    draft.confirmation !== "" && draft.confirmation !== draft.password;

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (draft.password !== draft.confirmation) {
      setFailure("password_mismatch");
      return;
    }
    setBusy(true);
    setFailure(undefined);
    const code = await onSubmit();
    setBusy(false);
    if (code) setFailure(code);
  }

  return (
    <form onSubmit={submit} className="flex max-w-xl flex-col gap-4">
      <StepHeader
        position={2}
        total={7}
        title={t("setup.admin.title")}
        lead={t("setup.admin.lead")}
      />
      <Field
        label={t("auth.username")}
        hint={t("auth.usernameHint")}
        value={draft.username}
        onChange={(event) =>
          onChange({ ...draft, username: event.target.value })
        }
        autoComplete="username"
        minLength={3}
        maxLength={64}
        required
      />
      <Field
        label={t("auth.password")}
        hint={t("auth.passwordHint")}
        type="password"
        value={draft.password}
        onChange={(event) =>
          onChange({ ...draft, password: event.target.value })
        }
        autoComplete="new-password"
        minLength={12}
        maxLength={256}
        required
      />
      <Field
        label={t("setup.admin.confirm")}
        type="password"
        value={draft.confirmation}
        onChange={(event) =>
          onChange({ ...draft, confirmation: event.target.value })
        }
        autoComplete="new-password"
        aria-invalid={mismatch || undefined}
        required
      />
      {mismatch && <p className="text-bad-fg">{t("setup.admin.mismatch")}</p>}
      {failure === "password_mismatch" && (
        <ErrorAlert>{t("setup.admin.mismatch")}</ErrorAlert>
      )}
      {failure && failure !== "password_mismatch" && (
        <ErrorAlert>
          {failure === "setup_closed_or_invalid_code"
            ? t("setup.admin.codeLost")
            : error(failure)}
        </ErrorAlert>
      )}
      <div className="flex flex-wrap gap-2">
        <button type="button" onClick={onBack} className={buttonClass()}>
          {t("setup.back")}
        </button>
        <button
          type="submit"
          disabled={busy}
          className={buttonClass("primary")}
        >
          {busy ? t("setup.admin.submitting") : t("setup.admin.submit")}
        </button>
      </div>
    </form>
  );
}
