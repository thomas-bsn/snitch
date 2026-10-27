import { useState } from "react";
import type { components } from "../../../api.gen";
import { api, errorCode, mutation } from "../../api/client";
import { useLoaded } from "../../api/loaded";
import { ErrorAlert } from "../../components/Alert";
import { buttonClass } from "../../components/buttons";
import { ConfirmDialog } from "../../components/ConfirmDialog";
import { useI18n } from "../../i18n";
import { isSuperAdministrator, roleLabelKey } from "../../roles";
import { useSession } from "../../session";
import { useFormatDate } from "../format";

type Member = components["schemas"]["Member"];
async function load() {
  try {
    const r = await api.GET("/api/admin/members");
    return r.data
      ? { ok: true as const, value: r.data }
      : { ok: false as const, code: errorCode(r.error) ?? "unknown" };
  } catch {
    return { ok: false as const, code: "unavailable" };
  }
}
export function MembersTab() {
  const { t, error } = useI18n();
  const { user, session } = useSession();
  const format = useFormatDate();
  const { state, reload } = useLoaded(load);
  const [selected, setSelected] = useState<Member>();
  const [action, setAction] = useState<"role" | "delete">();
  const [hide, setHide] = useState(false);
  const [failure, setFailure] = useState<string>();
  const [busy, setBusy] = useState(false);
  function open(member: Member, kind: "role" | "delete") {
    setSelected(member);
    setAction(kind);
    setHide(false);
    setFailure(undefined);
  }
  async function confirm() {
    if (!selected || !action) return;
    setBusy(true);
    setFailure(undefined);
    try {
      const path = { id: selected.id };
      const header = mutation(session?.csrf_token);
      const r =
        action === "role"
          ? await api.PUT("/api/admin/members/{id}/role", {
              params: { path, header },
              body: { role: selected.role === "user" ? "admin" : "user" },
            })
          : await api.DELETE("/api/admin/members/{id}", {
              params: { path, header },
              body: { hide_clips: hide },
            });
      if (r.error) {
        setFailure(errorCode(r.error) ?? "unknown");
        return;
      }
      setAction(undefined);
      setSelected(undefined);
      reload();
    } catch {
      setFailure("unavailable");
    } finally {
      setBusy(false);
    }
  }
  return (
    <section className="flex flex-col gap-4">
      <h2 className="text-lg font-semibold">{t("members.title")}</h2>
      {state.status === "loading" && <p role="status">{t("app.loading")}</p>}
      {state.status === "failed" && (
        <ErrorAlert>{error(state.code)}</ErrorAlert>
      )}
      {failure && !action && <ErrorAlert>{error(failure)}</ErrorAlert>}
      {state.status === "ready" && (
        <div
          role="region"
          tabIndex={0}
          aria-label={t("members.title")}
          className="overflow-x-auto"
        >
          <table className="w-full min-w-[38rem] text-left">
            <thead>
              <tr>
                {(["name", "role", "joined", "clips", "actions"] as const).map(
                  (k) => (
                    <th key={k} scope="col" className="p-2">
                      {t(`members.columns.${k}`)}
                    </th>
                  ),
                )}
              </tr>
            </thead>
            <tbody>
              {state.value.items.map((m) => {
                const editable =
                  !!user &&
                  m.id !== user.id &&
                  m.role !== "super_admin" &&
                  (m.role !== "admin" || isSuperAdministrator(user));
                return (
                  <tr key={m.id} className="border-t border-line">
                    <th scope="row" className="p-2">
                      {m.username}
                    </th>
                    <td className="p-2">{t(roleLabelKey(m))}</td>
                    <td className="p-2">{format(m.created_at)}</td>
                    <td className="p-2">{m.clip_count}</td>
                    <td className="p-2">
                      <div className="flex gap-2">
                        {editable && user && isSuperAdministrator(user) && (
                          <button
                            type="button"
                            className={buttonClass()}
                            onClick={() => open(m, "role")}
                          >
                            {t("members.changeRole")}
                          </button>
                        )}
                        {editable && (
                          <button
                            type="button"
                            className={buttonClass("danger")}
                            onClick={() => open(m, "delete")}
                          >
                            {t("members.delete")}
                          </button>
                        )}
                      </div>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}
      <ConfirmDialog
        open={!!action}
        onOpenChange={(open) => {
          if (!open) setAction(undefined);
        }}
        title={
          action === "role" ? t("members.roleTitle") : t("members.deleteTitle")
        }
        cancelLabel={t("members.cancel")}
        confirmLabel={
          action === "role" ? t("members.confirm") : t("members.deleteConfirm")
        }
        danger={action === "delete"}
        busy={busy}
        onConfirm={() => void confirm()}
      >
        {selected && (
          <>
            <p>
              {action === "role"
                ? selected.role === "user"
                  ? t("members.promote", { name: selected.username })
                  : t("members.demote", { name: selected.username })
                : t("members.deleteBody", { name: selected.username })}
            </p>
            {action === "delete" && (
              <label className="flex min-h-11 items-center gap-2">
                <input
                  type="checkbox"
                  checked={hide}
                  onChange={(e) => setHide(e.target.checked)}
                />
                {t("members.hide")}
              </label>
            )}
            {failure && <ErrorAlert>{error(failure)}</ErrorAlert>}
          </>
        )}
      </ConfirmDialog>
    </section>
  );
}
