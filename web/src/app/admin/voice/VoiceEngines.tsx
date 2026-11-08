import { useCallback, useState, type FormEvent } from "react";
import { api, errorCode, mutation } from "../../api/client";
import { useLoaded } from "../../api/loaded";
import { ErrorAlert } from "../../components/Alert";
import { buttonClass } from "../../components/buttons";
import { ConfirmDialog } from "../../components/ConfirmDialog";
import { useI18n } from "../../i18n";
import { isSuperAdministrator } from "../../roles";
import { useSession } from "../../session";

async function fetchEngines() {
  try {
    const { data, error } = await api.GET("/api/admin/voice-engines");
    return data
      ? ({ ok: true, value: data } as const)
      : ({ ok: false, code: errorCode(error) ?? "unknown" } as const);
  } catch {
    return { ok: false, code: "unavailable" } as const;
  }
}

/** The trust keys belong to the operator; this form only imports signed releases. */
export function VoiceEngines() {
  const { t, error } = useI18n();
  const { user, session } = useSession();
  const { state, reload } = useLoaded(useCallback(fetchEngines, []));
  const [envelope, setEnvelope] = useState("");
  const [busy, setBusy] = useState(false);
  const [failure, setFailure] = useState<string>();
  const [revoking, setRevoking] = useState<string>();
  const owner = user !== null && isSuperAdministrator(user);

  async function importRelease(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setFailure(undefined);
    let body: { key_id: string; payload: string; signature: string };
    try {
      body = JSON.parse(envelope);
      if (
        !body ||
        [body.key_id, body.payload, body.signature].some(
          (v) => typeof v !== "string" || !v,
        )
      )
        throw new Error();
    } catch {
      setFailure("engine_release_untrusted");
      return;
    }
    setBusy(true);
    try {
      const result = await api.POST("/api/admin/voice-engines", {
        params: { header: mutation(session?.csrf_token) },
        body,
      });
      if (result.error) setFailure(errorCode(result.error) ?? "unknown");
      else {
        setEnvelope("");
        await reload();
      }
    } catch {
      setFailure("unavailable");
    } finally {
      setBusy(false);
    }
  }

  async function revoke() {
    if (!revoking) return;
    setBusy(true);
    setFailure(undefined);
    try {
      const result = await api.DELETE("/api/admin/voice-engines/{digest}", {
        params: {
          path: { digest: revoking },
          header: mutation(session?.csrf_token),
        },
      });
      if (result.error) setFailure(errorCode(result.error) ?? "unknown");
      else await reload();
    } catch {
      setFailure("unavailable");
    } finally {
      setBusy(false);
      setRevoking(undefined);
    }
  }

  return (
    <section
      aria-labelledby="voice-engines"
      className="flex flex-col gap-4 rounded-xl border border-line p-4"
    >
      <h3 id="voice-engines" className="font-semibold">
        {t("voiceEngines.title")}
      </h3>
      <p className="text-sm text-ink-soft">{t("voiceEngines.lead")}</p>
      {failure && <ErrorAlert>{error(failure)}</ErrorAlert>}
      {state.status === "loading" && <p role="status">{t("app.loading")}</p>}
      {state.status === "failed" && (
        <ErrorAlert>{error(state.code)}</ErrorAlert>
      )}
      {state.status === "ready" && (
        <ul className="flex flex-col gap-3">
          {state.value.engines.length === 0 && (
            <li>{t("voiceEngines.empty")}</li>
          )}
          {state.value.engines.map((release) => (
            <li
              key={release.digest}
              className="flex flex-wrap items-center justify-between gap-2 border-b border-line pb-2"
            >
              <div>
                <strong>
                  {release.manifest.engine} · {release.manifest.version}
                </strong>
                <p className="text-sm text-ink-soft">
                  {release.manifest.kind} · Python {release.manifest.python} ·{" "}
                  {release.manifest.platform} · CUDA {release.manifest.cuda}
                </p>
                <p className="text-sm text-ink-soft">
                  {t("voiceEngines.constraints", {
                    disk: Math.ceil(release.manifest.disk_bytes / 1024 ** 2),
                    vram: Math.ceil(release.manifest.vram_bytes / 1024 ** 2),
                    license: release.manifest.license,
                  })}
                </p>
              </div>
              {release.revoked ? (
                <span>{t("voiceEngines.revoked")}</span>
              ) : (
                owner && (
                  <button
                    type="button"
                    className={buttonClass("danger")}
                    disabled={busy}
                    onClick={() => setRevoking(release.digest)}
                  >
                    {t("voiceEngines.revoke")}
                  </button>
                )
              )}
            </li>
          ))}
        </ul>
      )}
      {owner && (
        <form onSubmit={importRelease} className="flex flex-col gap-3">
          <label className="flex flex-col gap-1 text-sm">
            {t("voiceEngines.envelope")}
            <textarea
              value={envelope}
              onChange={(e) => setEnvelope(e.target.value)}
              required
              maxLength={400000}
              rows={4}
              className="rounded-md border border-line-strong bg-surface p-2 font-mono"
            />
          </label>
          <button
            type="submit"
            disabled={busy || !envelope.trim()}
            className={buttonClass("primary")}
          >
            {t("voiceEngines.import")}
          </button>
        </form>
      )}
      <ConfirmDialog
        open={!!revoking}
        onOpenChange={(open) => !open && !busy && setRevoking(undefined)}
        title={t("voiceEngines.revoke")}
        cancelLabel={t("voiceEngines.cancel")}
        confirmLabel={t("voiceEngines.revoke")}
        danger
        busy={busy}
        onConfirm={() => void revoke()}
      >
        {t("voiceEngines.revokeHint")}
      </ConfirmDialog>
    </section>
  );
}
