import { useState } from "react";
import type { components } from "../../../api.gen";
import { api, errorCode, mutation } from "../../api/client";
import { useLoaded } from "../../api/loaded";
import { ErrorAlert, StatusNotice } from "../../components/Alert";
import { buttonClass } from "../../components/buttons";
import { useI18n } from "../../i18n";
import { useSession } from "../../session";

type PersonLinkRequest = components["schemas"]["PersonLinkRequest"];

async function load() {
  try {
    const r = await api.GET("/api/admin/person-link-requests", {
      params: { query: { status: "pending" } },
    });
    if (r.data) return { ok: true as const, value: r.data.items };
    return { ok: false as const, code: errorCode(r.error) ?? "unknown" };
  } catch {
    return { ok: false as const, code: "unavailable" };
  }
}

/**
 * The administrators' queue of account↔Person link requests. Approving links the
 * requesting account to the person, replacing any existing link for that person;
 * refusing links nothing. Both decisions are traced server-side.
 */
export function PersonLinksTab() {
  const { t, error, language } = useI18n();
  const { session } = useSession();
  const header = mutation(session?.csrf_token);
  const { state, reload } = useLoaded(load);
  const [busy, setBusy] = useState<{
    id: number;
    kind: "approve" | "refuse";
  }>();
  const [failure, setFailure] = useState<string>();
  const [notice, setNotice] = useState<string>();

  async function decide(id: number, kind: "approve" | "refuse") {
    setBusy({ id, kind });
    setFailure(undefined);
    setNotice(undefined);
    try {
      const r =
        kind === "approve"
          ? await api.POST("/api/admin/person-link-requests/{id}/approve", {
              params: { path: { id }, header },
            })
          : await api.POST("/api/admin/person-link-requests/{id}/refuse", {
              params: { path: { id }, header },
            });
      if (r.error) {
        setFailure(errorCode(r.error) ?? "unknown");
      } else {
        setNotice(
          kind === "approve"
            ? t("personLinks.approved")
            : t("personLinks.refused"),
        );
        reload();
      }
    } catch {
      setFailure("unavailable");
    } finally {
      setBusy(undefined);
    }
  }

  return (
    <section className="flex max-w-3xl flex-col gap-4">
      <h2 className="text-lg font-semibold">{t("personLinks.title")}</h2>
      <p className="text-mute">{t("personLinks.intro")}</p>
      {state.status === "loading" && <p role="status">{t("app.loading")}</p>}
      {state.status === "failed" && (
        <ErrorAlert>{error(state.code)}</ErrorAlert>
      )}
      {failure && <ErrorAlert>{error(failure)}</ErrorAlert>}
      {notice && <StatusNotice>{notice}</StatusNotice>}
      {state.status === "ready" && state.value.length === 0 && (
        <p className="text-mute">{t("personLinks.empty")}</p>
      )}
      {state.status === "ready" && state.value.length > 0 && (
        <ul className="flex list-none flex-col gap-3 p-0">
          {state.value.map((request: PersonLinkRequest) => (
            <li
              key={request.id}
              className="flex flex-col gap-2 rounded-xl border border-line bg-surface p-4"
            >
              <div className="flex flex-wrap items-baseline gap-x-2">
                <span className="font-medium">
                  {request.person.display_name}
                </span>
                <span className="text-mute">· {request.account.username}</span>
                <span className="text-xs text-mute">
                  {new Date(request.created_at).toLocaleString(language)}
                </span>
              </div>
              {request.message && (
                <p className="whitespace-pre-wrap">{request.message}</p>
              )}
              <div className="flex gap-2">
                <button
                  type="button"
                  disabled={busy !== undefined}
                  className={buttonClass("primary")}
                  onClick={() => void decide(request.id, "approve")}
                >
                  {busy?.id === request.id && busy.kind === "approve"
                    ? t("personLinks.approving")
                    : t("personLinks.approve")}
                </button>
                <button
                  type="button"
                  disabled={busy !== undefined}
                  className={buttonClass("secondary")}
                  onClick={() => void decide(request.id, "refuse")}
                >
                  {busy?.id === request.id && busy.kind === "refuse"
                    ? t("personLinks.refusing")
                    : t("personLinks.refuse")}
                </button>
              </div>
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}
