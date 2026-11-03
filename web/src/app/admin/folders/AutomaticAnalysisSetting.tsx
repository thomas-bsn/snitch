import { useEffect, useId, useState } from "react";
import { api, errorCode, mutation } from "../../api/client";
import { ErrorAlert } from "../../components/Alert";
import { useI18n } from "../../i18n";
import { useSession } from "../../session";

/** The setting "Analyse complète automatique" (ADR 0035): on by default, changed at once. */
export function AutomaticAnalysisSetting() {
  const { t, error } = useI18n();
  const { session } = useSession();
  const id = useId();
  const [enabled, setEnabled] = useState<boolean | null>(null);
  const [busy, setBusy] = useState(false);
  const [failure, setFailure] = useState<string>();

  useEffect(() => {
    let cancelled = false;
    api
      .GET("/api/admin/settings/automatic-analysis")
      .then((r) => {
        if (cancelled) return;
        if (r.data) setEnabled(r.data.enabled);
        else setFailure(errorCode(r.error) ?? "unknown");
      })
      .catch(() => {
        if (!cancelled) setFailure("unavailable");
      });
    return () => {
      cancelled = true;
    };
  }, []);

  async function change(next: boolean) {
    const previous = enabled;
    setEnabled(next);
    setBusy(true);
    setFailure(undefined);
    try {
      const r = await api.PUT("/api/admin/settings/automatic-analysis", {
        params: { header: mutation(session?.csrf_token) },
        body: { enabled: next },
      });
      if (r.data) setEnabled(r.data.enabled);
      else {
        setEnabled(previous);
        setFailure(errorCode(r.error) ?? "unknown");
      }
    } catch {
      setEnabled(previous);
      setFailure("unavailable");
    } finally {
      setBusy(false);
    }
  }

  return (
    <section
      aria-labelledby={`${id}-title`}
      className="flex flex-col gap-2 rounded-xl border border-line p-4"
    >
      <h2 id={`${id}-title`} className="text-lg font-semibold">
        {t("folders.autoAnalysis.title")}
      </h2>
      <label className="flex min-h-11 items-center gap-2">
        <input
          type="checkbox"
          checked={enabled ?? false}
          disabled={busy || enabled === null}
          onChange={(event) => void change(event.target.checked)}
          aria-describedby={`${id}-help`}
        />
        {t("folders.autoAnalysis.label")}
      </label>
      <p id={`${id}-help`} className="text-mute">
        {t("folders.autoAnalysis.help")}
      </p>
      {failure && <ErrorAlert>{error(failure)}</ErrorAlert>}
    </section>
  );
}
