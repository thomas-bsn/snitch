import { useState } from "react";
import { Pencil } from "lucide-react";
import type { components } from "../../../api.gen";
import { api, errorCode } from "../../api/client";
import { useLoaded } from "../../api/loaded";
import { ErrorAlert } from "../../components/Alert";
import { buttonClass } from "../../components/buttons";
import { useI18n } from "../../i18n";
import { roleLabelKey } from "../../roles";
import { useFormatDate } from "../format";
import { MemberEditor } from "./MemberEditor";

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
  const format = useFormatDate();
  const { state, reload } = useLoaded(load);
  const [editing, setEditing] = useState<number>();
  return (
    <section className="flex flex-col gap-4">
      <h2 className="text-lg font-semibold">{t("members.title")}</h2>
      {state.status === "loading" && <p role="status">{t("app.loading")}</p>}
      {state.status === "failed" && (
        <ErrorAlert>{error(state.code)}</ErrorAlert>
      )}
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
                {(["name", "role", "joined", "clips"] as const).map((k) => (
                  <th key={k} scope="col" className="p-2">
                    {t(`members.columns.${k}`)}
                  </th>
                ))}
                <th scope="col" className="p-2">
                  <span className="sr-only">
                    {t("members.columns.actions")}
                  </span>
                </th>
              </tr>
            </thead>
            <tbody>
              {state.value.items.map((m: Member) => (
                <tr key={m.id} className="border-t border-line">
                  <th scope="row" className="p-2 font-normal">
                    <div className="flex items-center gap-3">
                      <span
                        aria-hidden="true"
                        className="grid size-9 shrink-0 place-items-center rounded-full bg-avatar-bg font-semibold uppercase"
                      >
                        {m.display_name.slice(0, 1)}
                      </span>
                      <span className="min-w-0">
                        <span className="block truncate font-medium">
                          {m.display_name}
                        </span>
                        <span className="block truncate text-xs text-mute">
                          {m.username}
                        </span>
                      </span>
                    </div>
                  </th>
                  <td className="p-2">{t(roleLabelKey(m))}</td>
                  <td className="p-2">{format(m.created_at)}</td>
                  <td className="p-2">{m.clip_count}</td>
                  <td className="p-2 text-right">
                    <button
                      type="button"
                      className={buttonClass()}
                      aria-label={`${t("members.edit.action")} ${m.username}`}
                      onClick={() => setEditing(m.id)}
                    >
                      <Pencil className="size-4" aria-hidden="true" />
                      {t("members.edit.action")}
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
      <MemberEditor
        id={editing}
        name={
          (state.status === "ready" &&
            state.value.items.find((m) => m.id === editing)?.display_name) ||
          ""
        }
        onClose={() => setEditing(undefined)}
        onChanged={reload}
      />
    </section>
  );
}
