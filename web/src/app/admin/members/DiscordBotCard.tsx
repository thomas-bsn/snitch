import { useCallback, useState, type FormEvent } from "react";
import { api, errorCode, mutation } from "../../api/client";
import { useLoaded } from "../../api/loaded";
import { ErrorAlert, StatusNotice } from "../../components/Alert";
import { ConfirmDialog } from "../../components/ConfirmDialog";
import { buttonClass } from "../../components/buttons";
import { Field } from "../../components/Field";
import { useI18n } from "../../i18n";
import { isSuperAdministrator } from "../../roles";
import { useSession } from "../../session";

async function load() {
  try {
    const r = await api.GET("/api/admin/discord-bot");
    if (r.data) return { ok: true as const, value: r.data };
    return { ok: false as const, code: errorCode(r.error) ?? "unknown" };
  } catch {
    return { ok: false as const, code: "unavailable" };
  }
}

/** The Discord bot that keeps profile pictures up to date. Its token is independent of the OAuth credentials. */
export function DiscordBotCard() {
  const { t, error } = useI18n();
  const { user, session } = useSession();
  const stableLoad = useCallback(load, []);
  const { state, reload } = useLoaded(stableLoad);
  const [token, setToken] = useState("");
  const [busy, setBusy] = useState(false);
  const [failure, setFailure] = useState<string>();
  const [notice, setNotice] = useState<string>();
  const [confirm, setConfirm] = useState(false);
  const owner = !!user && isSuperAdministrator(user);
  const header = mutation(session?.csrf_token);

  async function run(work: () => Promise<unknown>, done: string) {
    setBusy(true);
    setFailure(undefined);
    setNotice(undefined);
    try {
      const r = (await work()) as { error?: unknown };
      if (r.error) setFailure(errorCode(r.error) ?? "unknown");
      else setNotice(done);
    } catch {
      setFailure("unavailable");
    } finally {
      setBusy(false);
      reload();
    }
  }
  function connect(e: FormEvent) {
    e.preventDefault();
    void run(async () => {
      const r = await api.PUT("/api/admin/discord-bot", {
        params: { header },
        body: { token },
      });
      if (!r.error) setToken("");
      return r;
    }, t("members.bot.connected"));
  }

  const bot = state.status === "ready" ? state.value : undefined;
  const configured = !!bot && bot.status !== "not_configured";
  const statusText = bot
    ? {
        connected: t("members.bot.statusConnected"),
        not_configured: t("members.bot.statusNotConfigured"),
        error: t("members.bot.statusError"),
      }[bot.status]
    : "";
  return (
    <section
      aria-labelledby="discord-bot-title"
      className="flex max-w-xl flex-col gap-3 rounded-xl border border-line p-4"
    >
      <h3 id="discord-bot-title" className="font-semibold">
        {t("members.bot.title")}
      </h3>
      <p className="text-sm text-mute">{t("members.bot.explain")}</p>
      {state.status === "loading" && <p role="status">{t("app.loading")}</p>}
      {state.status === "failed" && (
        <ErrorAlert>{error(state.code)}</ErrorAlert>
      )}
      {failure && <ErrorAlert>{error(failure)}</ErrorAlert>}
      {notice && <StatusNotice>{notice}</StatusNotice>}
      {bot && (
        <>
          <dl className="grid grid-cols-[max-content_1fr] gap-x-6 gap-y-1.5">
            <dt className="text-mute">{t("members.bot.state")}</dt>
            <dd className="font-medium">
              {statusText}
              {bot.status === "error" && bot.last_error
                ? ` — ${error(bot.last_error)}`
                : ""}
            </dd>
            {bot.bot_name && (
              <>
                <dt className="text-mute">{t("members.bot.name")}</dt>
                <dd className="font-medium">{bot.bot_name}</dd>
              </>
            )}
            {configured && (
              <>
                <dt className="text-mute">{t("members.bot.lastSync")}</dt>
                <dd>
                  {bot.running
                    ? t("members.bot.running")
                    : bot.last_sync_at
                      ? t("members.bot.lastSyncResult", {
                          date: new Date(bot.last_sync_at).toLocaleString(),
                          refreshed: bot.last_refreshed,
                          failed: bot.last_failed,
                        })
                      : t("members.bot.never")}
                </dd>
              </>
            )}
          </dl>
          {!owner && <p>{t("members.bot.ownerOnly")}</p>}
          {owner && (
            <form onSubmit={connect} className="flex flex-col gap-3">
              <Field
                label={t("members.bot.token")}
                type="password"
                value={token}
                onChange={(e) => setToken(e.target.value)}
                autoComplete="off"
                maxLength={128}
                hint={configured ? t("members.bot.tokenKept") : undefined}
              />
              <div className="flex flex-wrap gap-2">
                <button
                  disabled={busy || token.trim() === ""}
                  className={buttonClass("primary")}
                >
                  {configured
                    ? t("members.bot.replace")
                    : t("members.bot.connect")}
                </button>
                {configured && (
                  <>
                    <button
                      type="button"
                      disabled={busy}
                      className={buttonClass("secondary")}
                      onClick={() =>
                        void run(
                          () =>
                            api.POST("/api/admin/discord-bot/test", {
                              params: { header },
                            }),
                          t("members.bot.testOk"),
                        )
                      }
                    >
                      {t("members.bot.test")}
                    </button>
                    <button
                      type="button"
                      disabled={busy}
                      className={buttonClass("secondary")}
                      onClick={() => setConfirm(true)}
                    >
                      {t("members.bot.disconnect")}
                    </button>
                  </>
                )}
              </div>
            </form>
          )}
          {configured && (
            <div>
              <button
                type="button"
                disabled={busy || bot.running}
                className={buttonClass("secondary")}
                onClick={() =>
                  void run(
                    () =>
                      api.POST("/api/admin/discord-bot/sync", {
                        params: { header },
                      }),
                    t("members.bot.syncStarted"),
                  )
                }
              >
                {t("members.bot.sync")}
              </button>
            </div>
          )}
        </>
      )}
      <ConfirmDialog
        open={confirm}
        onOpenChange={setConfirm}
        title={t("members.bot.disconnectTitle")}
        cancelLabel={t("members.bot.cancel")}
        confirmLabel={t("members.bot.disconnect")}
        danger
        busy={busy}
        onConfirm={() => {
          setConfirm(false);
          void run(
            () => api.DELETE("/api/admin/discord-bot", { params: { header } }),
            t("members.bot.disconnected"),
          );
        }}
      >
        <p>{t("members.bot.disconnectBody")}</p>
      </ConfirmDialog>
    </section>
  );
}
