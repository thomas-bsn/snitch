import { useState, type FormEvent } from "react";
import { api, errorCode, mutation } from "../../api/client";
import { useLoaded } from "../../api/loaded";
import { ErrorAlert, StatusNotice } from "../../components/Alert";
import { buttonClass } from "../../components/buttons";
import { Field } from "../../components/Field";
import { useI18n } from "../../i18n";
import { isSuperAdministrator } from "../../roles";
import { useSession } from "../../session";
import { FolderRequests } from "./FolderRequests";
async function load() {
  try {
    const [users, settings] = await Promise.all([
      api.GET("/api/admin/users"),
      api.GET("/api/admin/settings"),
    ]);
    if (users.data && settings.data)
      return {
        ok: true as const,
        value: { users: users.data, settings: settings.data },
      };
    return {
      ok: false as const,
      code: errorCode(users.error ?? settings.error) ?? "unknown",
    };
  } catch {
    return { ok: false as const, code: "unavailable" };
  }
}
export function SignupsTab() {
  const { t, error } = useI18n();
  const { user, session } = useSession();
  const { state, reload } = useLoaded(load);
  const [failure, setFailure] = useState<string>();
  const [notice, setNotice] = useState(false);
  const [busy, setBusy] = useState(false);
  const [id, setId] = useState<string>();
  const [secret, setSecret] = useState("");
  const [approval, setApproval] = useState<boolean>();
  const owner = !!user && isSuperAdministrator(user);
  async function approve(id: number) {
    setBusy(true);
    setFailure(undefined);
    try {
      const r = await api.POST("/api/admin/users/{id}/approve", {
        params: { path: { id }, header: mutation(session?.csrf_token) },
      });
      if (r.error) setFailure(errorCode(r.error) ?? "unknown");
      else {
        setNotice(true);
        reload();
      }
    } catch {
      setFailure("unavailable");
    } finally {
      setBusy(false);
    }
  }
  async function save(e: FormEvent) {
    e.preventDefault();
    if (state.status !== "ready") return;
    setBusy(true);
    setFailure(undefined);
    try {
      const r = await api.PUT("/api/admin/settings", {
        params: { header: mutation(session?.csrf_token) },
        body: {
          client_id: id ?? state.value.settings.client_id,
          registration_approval:
            approval ?? state.value.settings.registration_approval,
          ...(secret ? { client_secret: secret } : {}),
        },
      });
      if (r.error) setFailure(errorCode(r.error) ?? "unknown");
      else {
        setId(undefined);
        setSecret("");
        setApproval(undefined);
        setNotice(true);
        reload();
      }
    } catch {
      setFailure("unavailable");
    } finally {
      setBusy(false);
    }
  }
  return (
    <section className="flex max-w-3xl flex-col gap-5">
      <h2 className="text-lg font-semibold">{t("members.signups")}</h2>
      {state.status === "loading" && <p role="status">{t("app.loading")}</p>}
      {state.status === "failed" && (
        <ErrorAlert>{error(state.code)}</ErrorAlert>
      )}
      {failure && <ErrorAlert>{error(failure)}</ErrorAlert>}
      {notice && <StatusNotice>{t("members.saved")}</StatusNotice>}
      {state.status === "ready" && (
        <>
          <ul className="flex list-none flex-col gap-2 p-0">
            {state.value.users.map((u) => (
              <li key={u.id} className="flex items-center gap-3">
                {u.username}
                <button
                  disabled={busy}
                  type="button"
                  className={buttonClass()}
                  onClick={() => void approve(u.id)}
                >
                  {t("members.approve", { name: u.username })}
                </button>
              </li>
            ))}
          </ul>
          {state.value.users.length === 0 && <p>{t("members.noPending")}</p>}
          <FolderRequests />
          <form
            onSubmit={save}
            className="flex max-w-xl flex-col gap-3 rounded-xl border border-line p-4"
          >
            <h3 className="font-semibold">{t("members.accessSettings")}</h3>
            {!owner && <p>{t("members.ownerOnly")}</p>}
            <label className="flex min-h-11 items-center gap-2">
              <input
                type="checkbox"
                disabled={!owner}
                checked={approval ?? state.value.settings.registration_approval}
                onChange={(e) => setApproval(e.target.checked)}
              />
              {t("members.requireApproval")}
            </label>
            <Field
              label={t("members.discordId")}
              disabled={!owner}
              value={id ?? state.value.settings.client_id}
              onChange={(e) => setId(e.target.value)}
              maxLength={128}
            />
            <Field
              label={t("members.discordSecret")}
              disabled={!owner}
              type="password"
              value={secret}
              onChange={(e) => setSecret(e.target.value)}
              autoComplete="off"
              maxLength={2048}
              hint={
                state.value.settings.secret_configured
                  ? t("members.secretKept")
                  : undefined
              }
            />
            <Field
              label={t("members.callback")}
              value={state.value.settings.callback_url}
              readOnly
            />
            {state.value.settings.unreadable_secrets.length > 0 && (
              <p>
                {t("members.unreadable", {
                  names: state.value.settings.unreadable_secrets.join(", "),
                })}
              </p>
            )}
            {owner && (
              <div>
                <button disabled={busy} className={buttonClass("primary")}>
                  {t("members.saveSettings")}
                </button>
              </div>
            )}
          </form>
        </>
      )}
    </section>
  );
}
