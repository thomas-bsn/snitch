import { useState, type FormEvent } from "react";
import type { components } from "../../../api.gen";
import { api, errorCode, mutation } from "../../api/client";
import { useLoaded } from "../../api/loaded";
import { ErrorAlert, StatusNotice } from "../../components/Alert";
import { ConfirmDialog } from "../../components/ConfirmDialog";
import { buttonClass } from "../../components/buttons";
import { Field } from "../../components/Field";
import { useI18n } from "../../i18n";
import { isSuperAdministrator } from "../../roles";
import { useSession } from "../../session";

type Webhook = components["schemas"]["NotificationWebhook"];
type Category = Webhook["categories"][number];

const CATEGORIES: Category[] = [
  "access_requests",
  "human_actions",
  "worker_offline",
  "job_failures",
  "job_blocked",
];

async function load() {
  try {
    const r = await api.GET("/api/admin/notification-webhook");
    if (r.data) return { ok: true as const, value: r.data };
    return { ok: false as const, code: errorCode(r.error) ?? "unknown" };
  } catch {
    return { ok: false as const, code: "unavailable" };
  }
}

const date = (value: string | null | undefined) =>
  value ? new Date(value).toLocaleString() : "—";

/** Admin > Notifications: the Discord webhook that relays a redacted summary of some notifications (#283). */
export function NotificationWebhookTab() {
  const { t, error } = useI18n();
  const { user, session } = useSession();
  const { state, reload } = useLoaded(load);
  const [address, setAddress] = useState("");
  const [enabled, setEnabled] = useState<boolean>();
  const [categories, setCategories] = useState<Category[]>();
  const [busy, setBusy] = useState(false);
  const [failure, setFailure] = useState<string>();
  const [notice, setNotice] = useState<string>();
  const [confirm, setConfirm] = useState(false);
  const owner = !!user && isSuperAdministrator(user);
  const header = mutation(session?.csrf_token);

  async function run(work: () => Promise<{ error?: unknown }>, done: string) {
    setBusy(true);
    setFailure(undefined);
    setNotice(undefined);
    try {
      const r = await work();
      if (r.error) setFailure(errorCode(r.error) ?? "unknown");
      else {
        setAddress("");
        setEnabled(undefined);
        setCategories(undefined);
        setNotice(done);
      }
    } catch {
      setFailure("unavailable");
    } finally {
      setBusy(false);
      reload();
    }
  }

  const hook = state.status === "ready" ? state.value : undefined;
  const chosen = categories ?? hook?.categories ?? [];
  const on = enabled ?? hook?.enabled ?? false;

  function save(e: FormEvent) {
    e.preventDefault();
    void run(
      () =>
        api.PUT("/api/admin/notification-webhook", {
          params: { header },
          body: {
            enabled: on,
            categories: chosen,
            ...(address.trim() ? { url: address.trim() } : {}),
          },
        }),
      t("notificationWebhook.saved"),
    );
  }

  return (
    <section className="flex max-w-3xl flex-col gap-5">
      <h2 className="text-lg font-semibold">
        {t("notificationWebhook.title")}
      </h2>
      <p className="text-sm text-mute">{t("notificationWebhook.explain")}</p>
      {state.status === "loading" && <p role="status">{t("app.loading")}</p>}
      {state.status === "failed" && (
        <ErrorAlert>{error(state.code)}</ErrorAlert>
      )}
      {failure && <ErrorAlert>{error(failure)}</ErrorAlert>}
      {notice && <StatusNotice>{notice}</StatusNotice>}
      {hook && (
        <>
          <dl className="grid grid-cols-[max-content_1fr] gap-x-6 gap-y-1.5">
            <dt className="text-mute">{t("notificationWebhook.state")}</dt>
            <dd className="font-medium" data-testid="webhook-state">
              {!hook.configured
                ? t("notificationWebhook.notConfigured")
                : hook.enabled
                  ? t("notificationWebhook.enabledHint", { hint: hook.hint })
                  : t("notificationWebhook.disabledHint", { hint: hook.hint })}
            </dd>
            <dt className="text-mute">{t("notificationWebhook.lastStatus")}</dt>
            <dd data-testid="webhook-last-status">
              {t(`notificationWebhook.status.${hook.last_status}`)}
              {hook.last_error ? ` (${hook.last_error})` : ""}
              {hook.last_attempt_at ? ` — ${date(hook.last_attempt_at)}` : ""}
            </dd>
            <dt className="text-mute">{t("notificationWebhook.queue")}</dt>
            <dd>
              {t("notificationWebhook.queueCounts", {
                pending: hook.pending,
                failed: hook.failed,
              })}
            </dd>
          </dl>
          {!owner && <p>{t("notificationWebhook.ownerOnly")}</p>}
          {owner && (
            <form
              onSubmit={save}
              className="flex flex-col gap-3 rounded-xl border border-line p-4"
            >
              <Field
                label={t("notificationWebhook.url")}
                type="password"
                value={address}
                onChange={(e) => setAddress(e.target.value)}
                autoComplete="off"
                maxLength={512}
                placeholder="https://discord.com/api/webhooks/…"
                hint={
                  hook.configured
                    ? t("notificationWebhook.urlKept")
                    : t("notificationWebhook.urlHint")
                }
              />
              <label className="flex min-h-11 items-center gap-2">
                <input
                  type="checkbox"
                  checked={on}
                  onChange={(e) => setEnabled(e.target.checked)}
                />
                {t("notificationWebhook.enabled")}
              </label>
              <fieldset className="flex flex-col gap-1">
                <legend className="mb-1 font-medium">
                  {t("notificationWebhook.categories")}
                </legend>
                {CATEGORIES.map((c) => (
                  <label key={c} className="flex min-h-11 items-center gap-2">
                    <input
                      type="checkbox"
                      checked={chosen.includes(c)}
                      onChange={(e) =>
                        setCategories(
                          e.target.checked
                            ? [...chosen, c]
                            : chosen.filter((x) => x !== c),
                        )
                      }
                    />
                    {t(`notificationWebhook.category.${c}`)}
                  </label>
                ))}
              </fieldset>
              <p className="text-sm text-mute">
                {t("notificationWebhook.privacy")}
              </p>
              <div className="flex flex-wrap gap-2">
                <button disabled={busy} className={buttonClass("primary")}>
                  {t("notificationWebhook.save")}
                </button>
                {hook.configured && (
                  <>
                    <button
                      type="button"
                      disabled={busy}
                      className={buttonClass("secondary")}
                      onClick={() =>
                        void run(
                          () =>
                            api.POST("/api/admin/notification-webhook/test", {
                              params: { header },
                            }),
                          t("notificationWebhook.testOk"),
                        )
                      }
                    >
                      {t("notificationWebhook.test")}
                    </button>
                    <button
                      type="button"
                      disabled={busy}
                      className={buttonClass("secondary")}
                      onClick={() => setConfirm(true)}
                    >
                      {t("notificationWebhook.remove")}
                    </button>
                  </>
                )}
              </div>
            </form>
          )}
          <section aria-labelledby="webhook-deliveries">
            <h3 id="webhook-deliveries" className="mb-2 font-semibold">
              {t("notificationWebhook.deliveries")}
            </h3>
            {hook.deliveries.length === 0 ? (
              <p>{t("notificationWebhook.noDeliveries")}</p>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full text-left text-sm">
                  <thead>
                    <tr>
                      <th scope="col">{t("notificationWebhook.when")}</th>
                      <th scope="col">
                        {t("notificationWebhook.categoryColumn")}
                      </th>
                      <th scope="col">{t("notificationWebhook.outcome")}</th>
                      <th scope="col">{t("notificationWebhook.attempts")}</th>
                    </tr>
                  </thead>
                  <tbody>
                    {hook.deliveries.map((d) => (
                      <tr key={d.id} data-testid="webhook-delivery">
                        <td>{date(d.created_at)}</td>
                        <td>
                          {t(`notificationWebhook.category.${d.category}`)}
                        </td>
                        <td>
                          {t(`notificationWebhook.delivery.${d.status}`)}
                          {d.last_error ? ` (${d.last_error})` : ""}
                        </td>
                        <td>{d.attempts}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </section>
          <section aria-labelledby="webhook-audit">
            <h3 id="webhook-audit" className="mb-2 font-semibold">
              {t("notificationWebhook.audit")}
            </h3>
            {hook.audit.length === 0 ? (
              <p>{t("notificationWebhook.noAudit")}</p>
            ) : (
              <ul className="flex list-none flex-col gap-1 p-0 text-sm">
                {hook.audit.map((a, i) => (
                  <li key={i}>
                    {date(a.created_at)} —{" "}
                    {t(`notificationWebhook.action.${a.action}`)}
                    {a.actor_username ? ` (${a.actor_username})` : ""}
                  </li>
                ))}
              </ul>
            )}
          </section>
        </>
      )}
      <ConfirmDialog
        open={confirm}
        onOpenChange={setConfirm}
        title={t("notificationWebhook.removeTitle")}
        cancelLabel={t("notificationWebhook.cancel")}
        confirmLabel={t("notificationWebhook.remove")}
        danger
        busy={busy}
        onConfirm={() => {
          setConfirm(false);
          void run(
            () =>
              api.DELETE("/api/admin/notification-webhook", {
                params: { header },
              }),
            t("notificationWebhook.removed"),
          );
        }}
      >
        <p>{t("notificationWebhook.removeBody")}</p>
      </ConfirmDialog>
    </section>
  );
}
