import { useCallback, useState, type FormEvent } from "react";
import { ErrorAlert, StatusNotice } from "../../components/Alert";
import { buttonClass } from "../../components/buttons";
import { Field } from "../../components/Field";
import { api, errorCode, mutation } from "../../api/client";
import { useLoaded } from "../../api/loaded";
import { useI18n } from "../../i18n";
import { isSuperAdministrator } from "../../roles";
import { useSession } from "../../session";
import { StepHeader } from "../StepHeader";

async function fetchSettings() {
  const { data, error } = await api.GET("/api/admin/settings");
  return data
    ? ({ ok: true, value: data } as const)
    : ({ ok: false, code: errorCode(error) ?? "unknown" } as const);
}

type StepProps = {
  onDone: () => Promise<void>;
  skip: React.ReactNode;
};

/**
 * Step 3 (optional): Discord sign-in. The return address to register in Discord is shown
 * here. Only the Super-administrator can save the configuration.
 */
export function DiscordStep({ onDone, skip }: StepProps) {
  const { t, error } = useI18n();
  const { user, session } = useSession();
  const { state } = useLoaded(useCallback(fetchSettings, []));
  const [clientId, setClientId] = useState<string>();
  const [secret, setSecret] = useState("");
  const [busy, setBusy] = useState(false);
  const [failure, setFailure] = useState<string>();
  const owner = user !== null && isSuperAdministrator(user);

  const header = (
    <StepHeader
      position={3}
      total={7}
      title={t("setup.discord.title")}
      lead={t("setup.discord.lead")}
    />
  );
  if (state.status === "loading")
    return (
      <div className="flex flex-col gap-5">
        {header}
        <p role="status">{t("app.loading")}</p>
      </div>
    );
  if (state.status === "failed")
    return (
      <div className="flex max-w-xl flex-col gap-5">
        {header}
        <ErrorAlert>{error(state.code)}</ErrorAlert>
        <div>{skip}</div>
      </div>
    );
  const settings = state.value;
  const configured = settings.client_id !== "" && settings.secret_configured;
  const id = clientId ?? settings.client_id;

  async function save(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setBusy(true);
    setFailure(undefined);
    try {
      const result = await api.PUT("/api/admin/settings", {
        params: { header: mutation(session?.csrf_token) },
        body: {
          client_id: id.trim(),
          ...(secret ? { client_secret: secret } : {}),
          registration_approval: settings.registration_approval,
        },
      });
      if (result.error) {
        setFailure(errorCode(result.error) ?? "unavailable");
        return;
      }
      setSecret("");
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
        <StatusNotice>{t("setup.discord.ownerOnly")}</StatusNotice>
      ) : (
        <>
          {configured && (
            <StatusNotice>{t("setup.discord.already")}</StatusNotice>
          )}
          <Field
            label={t("setup.discord.appId")}
            value={id}
            onChange={(event) => setClientId(event.target.value)}
            inputMode="numeric"
            autoComplete="off"
            required
            maxLength={128}
          />
          <Field
            label={t("setup.discord.secret")}
            hint={
              configured
                ? t("setup.discord.secretKept")
                : t("setup.discord.secretHint")
            }
            type="password"
            value={secret}
            onChange={(event) => setSecret(event.target.value)}
            autoComplete="off"
            required={!configured}
            maxLength={2048}
          />
          <Field
            label={t("setup.discord.returnUrl")}
            hint={t("setup.discord.returnUrlHint")}
            value={settings.callback_url}
            readOnly
          />
          {failure && <ErrorAlert>{error(failure)}</ErrorAlert>}
        </>
      )}
      <div className="flex flex-wrap gap-2">
        {skip}
        {owner ? (
          <button
            type="submit"
            disabled={busy}
            className={buttonClass("primary")}
          >
            {busy ? t("setup.saving") : t("setup.saveContinue")}
          </button>
        ) : null}
      </div>
    </form>
  );
}
