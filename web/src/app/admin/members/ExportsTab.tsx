import { useState } from "react";
import { api, errorCode, mutation } from "../../api/client";
import { ErrorAlert } from "../../components/Alert";
import { buttonClass } from "../../components/buttons";
import { useI18n } from "../../i18n";
import { isSuperAdministrator } from "../../roles";
import { useSession } from "../../session";
export function ExportsTab() {
  const { t, error } = useI18n();
  const { user, session } = useSession();
  const [key, setKey] = useState<string>();
  const [failure, setFailure] = useState<string>();
  const [busy, setBusy] = useState(false);
  async function exportKey() {
    setBusy(true);
    setFailure(undefined);
    try {
      const r = await api.POST("/api/admin/master-key/export", {
        params: { header: mutation(session?.csrf_token) },
      });
      if (r.data) setKey(r.data.master_key);
      else setFailure(errorCode(r.error) ?? "unknown");
    } catch {
      setFailure("unavailable");
    } finally {
      setBusy(false);
    }
  }
  return (
    <section className="flex max-w-xl flex-col gap-4">
      <h2 className="text-lg font-semibold">{t("members.exports")}</h2>
      <p>{t("members.exportHint")}</p>
      <div>
        <a
          href="/api/admin/segment-corrections/export?format=ndjson"
          className={buttonClass()}
        >
          {t("members.exportSegments")}
        </a>
      </div>
      {user && isSuperAdministrator(user) && (
        <div className="flex flex-col gap-3 rounded-xl border border-line p-4">
          <h3 className="font-semibold">{t("members.masterKey")}</h3>
          <p>{t("members.masterWarning")}</p>
          <div>
            <button
              type="button"
              disabled={busy}
              onClick={() => void exportKey()}
              className={buttonClass()}
            >
              {t("members.exportKey")}
            </button>
          </div>
          {key && (
            <label className="flex flex-col gap-2">
              {t("members.masterKey")}
              <textarea
                readOnly
                value={key}
                rows={2}
                className="rounded-lg border border-line-strong bg-surface p-2"
              />
            </label>
          )}
        </div>
      )}
      {failure && <ErrorAlert>{error(failure)}</ErrorAlert>}
    </section>
  );
}
