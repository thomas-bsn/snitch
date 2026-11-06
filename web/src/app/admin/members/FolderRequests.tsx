import { useCallback, useEffect, useState } from "react";
import { ErrorAlert } from "../../components/Alert";
import { buttonClass } from "../../components/buttons";
import { useI18n } from "../../i18n";
import { useSession } from "../../session";

type FolderRequest = {
  id: number;
  path: string;
  requester: string;
  created_at: string;
};

/**
 * Folder requests waiting for a decision (ADR 0033, #225), next to the sign-ups waiting for approval.
 * Approving creates the folder in the Racine vidéo; refusing creates nothing. The requester is notified.
 */
export function FolderRequests() {
  const { t, error } = useI18n();
  const { session } = useSession();
  const csrf = session?.csrf_token ?? "";
  const [items, setItems] = useState<FolderRequest[]>([]);
  const [failure, setFailure] = useState<string>();
  const [busy, setBusy] = useState(false);

  const load = useCallback(async () => {
    const response = await fetch("/api/admin/folder-requests", {
      headers: { "X-CSRF-Token": csrf },
    });
    const data = await response.json();
    if (!response.ok) throw new Error(data.error ?? "unavailable");
    setItems(data.items);
  }, [csrf]);

  useEffect(() => {
    void load().catch((e: Error) => setFailure(e.message));
  }, [load]);

  async function decide(id: number, outcome: "approve" | "refuse") {
    setBusy(true);
    setFailure(undefined);
    try {
      const response = await fetch(
        `/api/admin/folder-requests/${id}/${outcome}`,
        {
          method: "POST",
          headers: { "X-CSRF-Token": csrf },
        },
      );
      const data = await response.json().catch(() => ({}));
      if (!response.ok) throw new Error(data.error ?? "unavailable");
      await load();
    } catch (e) {
      setFailure((e as Error).message);
    } finally {
      setBusy(false);
    }
  }

  return (
    <section
      aria-labelledby="folder-requests"
      className="flex max-w-3xl flex-col gap-3"
    >
      <h3 id="folder-requests" className="font-semibold">
        {t("members.folderRequests")}
      </h3>
      {failure && <ErrorAlert>{error(failure)}</ErrorAlert>}
      <ul className="flex list-none flex-col gap-2 p-0">
        {items.map((item) => (
          <li key={item.id} className="flex flex-wrap items-center gap-3">
            <span className="min-w-0 flex-1 break-all">
              {t("members.folderRequest", {
                name: item.requester,
                path: `/${item.path}`,
              })}
            </span>
            <button
              type="button"
              disabled={busy}
              className={buttonClass()}
              aria-label={t("members.folderApprove", { path: `/${item.path}` })}
              onClick={() => void decide(item.id, "approve")}
            >
              {t("members.approve", { name: item.path })}
            </button>
            <button
              type="button"
              disabled={busy}
              className={buttonClass("secondary")}
              aria-label={t("members.folderRefuse", { path: `/${item.path}` })}
              onClick={() => void decide(item.id, "refuse")}
            >
              {t("members.folderRefuse", { path: `/${item.path}` })}
            </button>
          </li>
        ))}
      </ul>
      {items.length === 0 && <p>{t("members.noFolderRequests")}</p>}
    </section>
  );
}
