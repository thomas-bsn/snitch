import { useCallback, useState, type FormEvent } from "react";
import { api, errorCode, mutation } from "../../api/client";
import { useLoaded } from "../../api/loaded";
import { ErrorAlert, StatusNotice } from "../../components/Alert";
import { buttonClass } from "../../components/buttons";
import { Field } from "../../components/Field";
import { useI18n } from "../../i18n";
import { useSession } from "../../session";

async function load() {
  const [settings, status] = await Promise.all([
    api.GET("/api/admin/embeddings/settings"),
    api.GET("/api/admin/embeddings"),
  ]);
  if (settings.data && status.data)
    return {
      ok: true as const,
      value: { settings: settings.data, status: status.data },
    };
  return {
    ok: false as const,
    code: errorCode(settings.error ?? status.error) ?? "unknown",
  };
}

export function SemanticSettings({
  onSaved,
  secondaryAction,
}: {
  onSaved?: () => Promise<void>;
  secondaryAction?: React.ReactNode;
}) {
  const { t, error } = useI18n();
  const { session } = useSession();
  const { state, reload } = useLoaded(useCallback(load, []));
  const [url, setURL] = useState<string>();
  const [model, setModel] = useState<string>();
  const [failure, setFailure] = useState<string>();
  const [saved, setSaved] = useState(false);
  const [busy, setBusy] = useState(false);
  if (state.status === "loading")
    return <p role="status">{t("app.loading")}</p>;
  if (state.status === "failed")
    return (
      <div className="flex flex-col gap-4">
        <ErrorAlert>{error(state.code)}</ErrorAlert>
        {secondaryAction}
      </div>
    );
  const { settings, status } = state.value;
  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setBusy(true);
    setFailure(undefined);
    setSaved(false);
    try {
      const result = await api.PUT("/api/admin/embeddings/settings", {
        params: { header: mutation(session?.csrf_token) },
        body: { url: url ?? settings.url, model: model ?? settings.model },
      });
      if (result.error) {
        setFailure(errorCode(result.error) ?? "unknown");
        return;
      }
      setSaved(true);
      reload();
      await onSaved?.();
    } catch {
      setFailure("unavailable");
    } finally {
      setBusy(false);
    }
  }
  return (
    <div className="flex max-w-xl flex-col gap-4">
      <p>
        {status.model
          ? t("setup.semantic.active", { model: status.model })
          : t("setup.semantic.inactive")}
      </p>
      <p>
        {t("setup.semantic.counts", {
          done: status.indexed_segments,
          pending: status.pending,
          reindex: status.reindex_pending,
        })}
      </p>
      {status.candidate_model && (
        <p>
          {t("setup.semantic.candidate", {
            model: status.candidate_model,
            pending: status.candidate_pending ?? 0,
          })}
        </p>
      )}
      <form onSubmit={(e) => void submit(e)} className="flex flex-col gap-4">
        <Field
          label={t("setup.semantic.url")}
          type="url"
          placeholder="http://embed:11434/v1/embeddings"
          required
          value={url ?? settings.url}
          onChange={(e) => setURL(e.target.value)}
          disabled={settings.url_from_environment}
          hint={
            settings.url_from_environment
              ? t("setup.semantic.environment")
              : undefined
          }
        />
        <Field
          label={t("setup.semantic.model")}
          required
          maxLength={128}
          value={model ?? settings.model}
          onChange={(e) => setModel(e.target.value)}
          disabled={settings.model_from_environment}
          hint={
            settings.model_from_environment
              ? t("setup.semantic.environment")
              : undefined
          }
        />
        {failure && <ErrorAlert>{error(failure)}</ErrorAlert>}
        {saved && <StatusNotice>{t("setup.semantic.saved")}</StatusNotice>}
        <div className="flex flex-wrap gap-2">
          {secondaryAction}
          <button
            type="submit"
            disabled={busy}
            className={buttonClass("primary")}
          >
            {t("setup.semantic.save")}
          </button>
        </div>
      </form>
    </div>
  );
}
