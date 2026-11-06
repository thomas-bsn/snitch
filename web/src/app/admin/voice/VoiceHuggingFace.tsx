import { useCallback, useState, type FormEvent } from "react";
import { api, errorCode, mutation } from "../../api/client";
import { useLoaded } from "../../api/loaded";
import { ErrorAlert, StatusNotice } from "../../components/Alert";
import { buttonClass } from "../../components/buttons";
import { ConfirmDialog } from "../../components/ConfirmDialog";
import { Field } from "../../components/Field";
import { StatusPill, type PillTone } from "../../components/StatusPill";
import { useI18n } from "../../i18n";
import type { MessageKey } from "../../i18n/messages";
import { isSuperAdministrator } from "../../roles";
import { useSession } from "../../session";

async function fetchStatus() {
  const { data, error } = await api.GET("/api/admin/hf-token");
  return data
    ? ({ ok: true, value: data } as const)
    : ({ ok: false, code: errorCode(error) ?? "unknown" } as const);
}

const stateTones: Record<string, PillTone> = {
  synced: "ok",
  pending: "idle",
  error: "bad",
};
const stateLabels: Record<string, MessageKey> = {
  synced: "voiceAdmin.hf.state.synced",
  pending: "voiceAdmin.hf.state.pending",
  error: "voiceAdmin.hf.state.error",
};
const sourceLabels: Record<string, MessageKey> = {
  central: "voiceAdmin.hf.source.central",
  local: "voiceAdmin.hf.source.local",
  none: "voiceAdmin.hf.source.none",
  unknown: "voiceAdmin.hf.source.unknown",
};

/**
 * The central Hugging Face token: entered once, kept encrypted, never shown again. Each Worker's
 * synchronisation is shown beside it. Only the Super-administrator can save, replace or remove it.
 */
export function VoiceHuggingFace() {
  const { t, error } = useI18n();
  const { user, session } = useSession();
  const { state, reload } = useLoaded(useCallback(fetchStatus, []));
  const [token, setToken] = useState("");
  const [busy, setBusy] = useState(false);
  const [failure, setFailure] = useState<string>();
  const [notice, setNotice] = useState<"saved" | "removed">();
  const [confirmRemoval, setConfirmRemoval] = useState(false);
  const owner = user !== null && isSuperAdministrator(user);

  async function save(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setBusy(true);
    setFailure(undefined);
    setNotice(undefined);
    try {
      const result = await api.PUT("/api/admin/hf-token", {
        params: { header: mutation(session?.csrf_token) },
        body: { token: token.trim() },
      });
      if (result.error) {
        setFailure(errorCode(result.error) ?? "unavailable");
        return;
      }
      setToken("");
      setNotice("saved");
      await reload();
    } catch {
      setFailure("unavailable");
    } finally {
      setBusy(false);
    }
  }

  async function remove() {
    setBusy(true);
    setFailure(undefined);
    setNotice(undefined);
    try {
      const result = await api.DELETE("/api/admin/hf-token", {
        params: { header: mutation(session?.csrf_token) },
      });
      if (result.error) {
        setFailure(errorCode(result.error) ?? "unavailable");
        return;
      }
      setNotice("removed");
      await reload();
    } catch {
      setFailure("unavailable");
    } finally {
      setBusy(false);
      setConfirmRemoval(false);
    }
  }

  const heading = (
    <h3 id="voice-hf" className="font-semibold">
      {t("voiceAdmin.hf.title")}
    </h3>
  );
  if (state.status === "loading")
    return (
      <section
        aria-labelledby="voice-hf"
        className="flex flex-col gap-3 rounded-xl border border-line p-4"
      >
        {heading}
        <p role="status">{t("app.loading")}</p>
      </section>
    );
  if (state.status === "failed")
    return (
      <section
        aria-labelledby="voice-hf"
        className="flex flex-col gap-3 rounded-xl border border-line p-4"
      >
        {heading}
        <ErrorAlert>{error(state.code)}</ErrorAlert>
      </section>
    );

  const view = state.value;
  const configuredText = view.configured
    ? t("voiceAdmin.hf.configured")
    : view.unreadable
      ? t("voiceAdmin.hf.unreadable")
      : t("voiceAdmin.hf.absent");

  return (
    <section
      aria-labelledby="voice-hf"
      className="flex flex-col gap-4 rounded-xl border border-line p-4"
    >
      {heading}
      <p className="text-sm text-ink-soft">{t("voiceAdmin.hf.lead")}</p>
      <StatusNotice>{configuredText}</StatusNotice>
      {notice && (
        <StatusNotice>
          {notice === "saved"
            ? t("voiceAdmin.hf.savedNotice")
            : t("voiceAdmin.hf.removedNotice")}
        </StatusNotice>
      )}
      {failure && <ErrorAlert>{error(failure)}</ErrorAlert>}
      {!owner ? (
        <StatusNotice>{t("voiceAdmin.hf.owner")}</StatusNotice>
      ) : (
        <form onSubmit={save} className="flex max-w-xl flex-col gap-3">
          <Field
            label={t("voiceAdmin.hf.field")}
            hint={t("voiceAdmin.hf.hint")}
            type="password"
            value={token}
            onChange={(event) => setToken(event.target.value)}
            autoComplete="off"
            required
            maxLength={210}
          />
          <div className="flex flex-wrap gap-2">
            <button
              type="submit"
              disabled={busy || token.trim() === ""}
              className={buttonClass("primary")}
            >
              {view.configured
                ? t("voiceAdmin.hf.replace")
                : t("voiceAdmin.hf.save")}
            </button>
            {view.configured && (
              <button
                type="button"
                disabled={busy}
                onClick={() => setConfirmRemoval(true)}
                className={buttonClass("danger")}
              >
                {t("voiceAdmin.hf.remove")}
              </button>
            )}
          </div>
        </form>
      )}
      {view.workers.length > 0 && (
        <div className="flex flex-col gap-2">
          <h4 className="text-sm font-semibold">
            {t("voiceAdmin.hf.workersTitle")}
          </h4>
          <p className="text-xs text-ink-soft">
            {t("voiceAdmin.hf.workersLead")}
          </p>
          <ul className="flex flex-col gap-2">
            {view.workers.map((worker) => (
              <li
                key={worker.id}
                className="flex flex-wrap items-center gap-2 text-sm"
              >
                <span className="font-medium">{worker.name}</span>
                <StatusPill tone={stateTones[worker.state] ?? "idle"}>
                  {t(
                    stateLabels[worker.state] ?? "voiceAdmin.hf.state.pending",
                  )}
                </StatusPill>
                <span className="text-ink-soft">
                  {t(
                    sourceLabels[worker.source || "unknown"] ??
                      "voiceAdmin.hf.source.unknown",
                  )}
                </span>
                {worker.error_code && (
                  <span className="text-bad-fg">
                    {error(worker.error_code)}
                  </span>
                )}
              </li>
            ))}
          </ul>
        </div>
      )}
      <ConfirmDialog
        open={confirmRemoval}
        onOpenChange={(open) => !open && setConfirmRemoval(false)}
        title={t("voiceAdmin.hf.remove")}
        cancelLabel={t("voiceAdmin.hf.cancelRemove")}
        confirmLabel={t("voiceAdmin.hf.remove")}
        onConfirm={() => void remove()}
        danger
        busy={busy}
      >
        {t("voiceAdmin.hf.removeHint")}
      </ConfirmDialog>
    </section>
  );
}
