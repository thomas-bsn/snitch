import { useState, type FormEvent } from "react";
import { ErrorAlert, StatusNotice } from "../../components/Alert";
import { buttonClass } from "../../components/buttons";
import { Field } from "../../components/Field";
import { api, errorCode, mutation } from "../../api/client";
import { useI18n } from "../../i18n";
import { isSuperAdministrator } from "../../roles";
import { useSession } from "../../session";
import { StepHeader } from "../StepHeader";

type StepProps = {
  onDone: () => Promise<void>;
  skip: React.ReactNode;
};

/**
 * Step 7 (optional): the central Hugging Face token, needed only to download voice models. It is
 * entered once, kept encrypted and sent to the Workers that are authorised; the same form lives in
 * Administration, Voice processing. Only the Super-administrator can save it.
 */
export function HuggingFaceStep({ onDone, skip }: StepProps) {
  const { t, error } = useI18n();
  const { user, session } = useSession();
  const [token, setToken] = useState("");
  const [busy, setBusy] = useState(false);
  const [failure, setFailure] = useState<string>();
  const [saved, setSaved] = useState(false);
  const owner = user !== null && isSuperAdministrator(user);

  const header = (
    <StepHeader
      position={7}
      total={9}
      title={t("setup.huggingface.title")}
      lead={t("setup.huggingface.lead")}
    />
  );

  async function save(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setBusy(true);
    setFailure(undefined);
    try {
      const result = await api.PUT("/api/admin/hf-token", {
        params: { header: mutation(session?.csrf_token) },
        body: { token: token.trim() },
      });
      if (result.error) {
        setFailure(errorCode(result.error) ?? "unavailable");
        return;
      }
      setToken("");
      setSaved(true);
      await onDone();
    } catch {
      setFailure("unavailable");
    } finally {
      setBusy(false);
    }
  }

  return (
    <form onSubmit={save} className="flex max-w-xl flex-col gap-4">
      {header}
      {!owner ? (
        <StatusNotice>{t("setup.huggingface.ownerOnly")}</StatusNotice>
      ) : (
        <>
          {saved && <StatusNotice>{t("setup.huggingface.saved")}</StatusNotice>}
          <Field
            label={t("setup.huggingface.field")}
            hint={t("setup.huggingface.hint")}
            type="password"
            value={token}
            onChange={(event) => setToken(event.target.value)}
            autoComplete="off"
            maxLength={210}
          />
          {failure && <ErrorAlert>{error(failure)}</ErrorAlert>}
        </>
      )}
      <div className="flex flex-wrap gap-2">
        {skip}
        {owner ? (
          <button
            type="submit"
            disabled={busy || token.trim() === ""}
            className={buttonClass("primary")}
          >
            {busy ? t("setup.saving") : t("setup.saveContinue")}
          </button>
        ) : null}
      </div>
    </form>
  );
}
