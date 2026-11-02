import { useCallback, useEffect, useState, type FormEvent } from "react";
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

type Proof = {
  url: string;
  model: string;
  token: string;
  dimensions: number;
  expiresAt: number;
};

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
  const [testing, setTesting] = useState(false);
  // The proof of a successful Test, valid for the exact values tested and for a short time only.
  const [proof, setProof] = useState<Proof>();
  const [expired, setExpired] = useState(false);
  const proofExpiresAt = proof?.expiresAt;
  useEffect(() => {
    if (proofExpiresAt === undefined) return;
    const timer = setTimeout(
      () => {
        setProof(undefined);
        setExpired(true);
      },
      Math.max(0, proofExpiresAt - Date.now()),
    );
    return () => clearTimeout(timer);
  }, [proofExpiresAt]);
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
  const values = {
    url: (url ?? settings.url).trim(),
    model: (model ?? settings.model).trim(),
  };
  const validated = proof?.url === values.url && proof.model === values.model;
  // Any edit voids the previous Test: the proof only ever covers what was tested.
  function edit(change: () => void) {
    change();
    setProof(undefined);
    setExpired(false);
    setFailure(undefined);
    setSaved(false);
  }
  async function runTest() {
    const tested = values;
    setTesting(true);
    setFailure(undefined);
    setSaved(false);
    setExpired(false);
    setProof(undefined);
    try {
      const result = await api.POST("/api/admin/embeddings/settings/test", {
        params: { header: mutation(session?.csrf_token) },
        body: tested,
      });
      if (result.error || !result.data) {
        setFailure(errorCode(result.error) ?? "unknown");
        return;
      }
      setProof({
        ...tested,
        token: result.data.validation_token,
        dimensions: result.data.dimensions,
        expiresAt: Date.parse(result.data.expires_at),
      });
    } catch {
      setFailure("unavailable");
    } finally {
      setTesting(false);
    }
  }
  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!proof || !validated || busy) return;
    setBusy(true);
    setFailure(undefined);
    setSaved(false);
    try {
      const result = await api.PUT("/api/admin/embeddings/settings", {
        params: { header: mutation(session?.csrf_token) },
        body: { ...values, validation_token: proof.token },
      });
      if (result.error) {
        const code = errorCode(result.error) ?? "unknown";
        // A refused proof is spent or stale: back to "not validated".
        if (code.startsWith("embedding_validation_")) setProof(undefined);
        setFailure(code);
        return;
      }
      setProof(undefined);
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
          onChange={(e) => edit(() => setURL(e.target.value))}
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
          onChange={(e) => edit(() => setModel(e.target.value))}
          disabled={settings.model_from_environment}
          hint={
            settings.model_from_environment
              ? t("setup.semantic.environment")
              : undefined
          }
        />
        <div className="flex flex-wrap items-center gap-2">
          {secondaryAction}
          <button
            type="button"
            disabled={testing || busy}
            onClick={() => void runTest()}
            className={buttonClass("secondary")}
          >
            {testing ? t("setup.semantic.testing") : t("setup.semantic.test")}
          </button>
          <button
            type="submit"
            disabled={!validated || busy || testing}
            aria-describedby="semantic-state"
            className={buttonClass("primary")}
          >
            {t("setup.semantic.save")}
          </button>
          <span
            id="semantic-state"
            className="rounded-full bg-accent-soft px-2.5 py-0.5 text-xs font-medium text-accent-ink"
          >
            {validated
              ? t("setup.semantic.validated")
              : t("setup.semantic.notValidated")}
          </span>
        </div>
        <div aria-live="polite" className="flex flex-col gap-2">
          {testing && (
            <StatusNotice>{t("setup.semantic.testing")}</StatusNotice>
          )}
          {validated && proof && (
            <StatusNotice>
              {t("setup.semantic.tested", { dimensions: proof.dimensions })}
            </StatusNotice>
          )}
          {expired && <ErrorAlert>{t("setup.semantic.expired")}</ErrorAlert>}
          {failure && <ErrorAlert>{error(failure)}</ErrorAlert>}
          {saved && <StatusNotice>{t("setup.semantic.saved")}</StatusNotice>}
        </div>
      </form>
    </div>
  );
}
