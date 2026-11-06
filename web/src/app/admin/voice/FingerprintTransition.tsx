import { useCallback, useEffect, useState } from "react";
import { Fingerprint } from "lucide-react";
import type { components } from "../../../api.gen";
import { api, errorCode, mutation } from "../../api/client";
import { useLoaded } from "../../api/loaded";
import { useI18n } from "../../i18n";
import { useSession } from "../../session";
import { Modal } from "../../components/Modal";
import { ErrorAlert } from "../../components/Alert";
import { buttonClass } from "../../components/buttons";
import { ConfirmDialog } from "../../components/ConfirmDialog";
import type { Catalogue } from "./VoiceModelCards";
type Preview = components["schemas"]["FingerprintTransitionPreview"];
const endpoint = "/api/admin/voice-models/fingerprint-transition";
async function load() {
  try {
    const result = await api.GET(endpoint);
    return result.data
      ? { ok: true as const, value: result.data }
      : { ok: false as const, code: errorCode(result.error) ?? "unknown" };
  } catch {
    return { ok: false as const, code: "unavailable" };
  }
}
/** Preview and counts come from the server; no recomputation or model switch is simulated by the UI. */
export function FingerprintTransition({
  catalogue,
  target,
  onClose,
  onChanged,
  onPrepare,
}: {
  catalogue: Catalogue;
  target?: string;
  onClose: () => void;
  onChanged: () => void;
  onPrepare: (modelId: string) => void;
}) {
  const { t, error } = useI18n();
  const { session } = useSession();
  const { state, reload } = useLoaded(load);
  const [viewing, setViewing] = useState(false);
  const [preview, setPreview] = useState<Preview>();
  const [failure, setFailure] = useState<string>();
  const [confirmed, setConfirmed] = useState(false);
  const [busy, setBusy] = useState(false);
  const [cancelling, setCancelling] = useState(false);
  const transition =
    state.status === "ready" ? state.value.transition : undefined;
  const active = state.status === "ready" && state.value.open;
  const loadPreview = useCallback(async () => {
    if (!target) return;
    setPreview(undefined);
    setFailure(undefined);
    setConfirmed(false);
    try {
      const r = await api.GET(
        "/api/admin/voice-models/fingerprint-transition/preview",
        { params: { query: { to: target } } },
      );
      if (r.data) setPreview(r.data);
      else setFailure(errorCode(r.error) ?? "unknown");
    } catch {
      setFailure("unavailable");
    }
  }, [target]);
  useEffect(() => {
    let current = true;
    if (!target) {
      setPreview(undefined);
      setFailure(undefined);
      setConfirmed(false);
      return;
    }
    setPreview(undefined);
    setFailure(undefined);
    setConfirmed(false);
    api
      .GET("/api/admin/voice-models/fingerprint-transition/preview", {
        params: { query: { to: target } },
      })
      .then((r) => {
        if (!current) return;
        if (r.data) setPreview(r.data);
        else setFailure(errorCode(r.error) ?? "unknown");
      })
      .catch(() => current && setFailure("unavailable"));
    return () => {
      current = false;
    };
  }, [target]);
  useEffect(() => {
    if (!active && !viewing) return;
    const timer = window.setInterval(() => {
      reload();
      onChanged();
    }, 3000);
    return () => window.clearInterval(timer);
  }, [active, viewing, reload, onChanged]);
  const text =
    transition?.phase === "preparing"
      ? t("voiceWorkspace.transitionPreparing")
      : transition?.phase === "recomputing"
        ? t("voiceWorkspace.transitionRecomputing", {
            done: transition.references_done,
            total: transition.references_total,
          })
        : transition?.phase === "switched"
          ? t("voiceWorkspace.transitionSwitched", {
              count: transition.catch_up_remaining,
            })
          : t("voiceWorkspace.transitionCancelled");
  async function run(action: "start" | "resume" | "cancel") {
    if (action === "start" && (!preview || !confirmed)) return;
    setBusy(true);
    setFailure(undefined);
    const params = { header: mutation(session?.csrf_token) };
    try {
      const r =
        action === "start"
          ? await api.POST(endpoint, {
              params,
              body: {
                to_model_id: preview!.to_model_id,
                confirm_references: preview!.references,
              },
            })
          : action === "cancel"
            ? await api.DELETE(endpoint, { params })
            : await api.POST(
                "/api/admin/voice-models/fingerprint-transition/resume",
                { params },
              );
      if (r.error) {
        const code = errorCode(r.error) ?? "unknown";
        if (code === "voice_transition_preview_stale") {
          await loadPreview();
        }
        setFailure(code);
      } else {
        reload();
        onChanged();
        onClose();
        setViewing(true);
        setCancelling(false);
      }
    } catch {
      setFailure("unavailable");
    } finally {
      setBusy(false);
    }
  }
  return (
    <>
      {state.status === "failed" && (
        <ErrorAlert>{error(state.code)}</ErrorAlert>
      )}
      {transition &&
        transition.phase !== "cancelled" &&
        (active || transition.catch_up_remaining > 0) && (
          <div className="voice-transition-banner">
            <Fingerprint size={19} aria-hidden />
            <div>
              <strong>
                {
                  catalogue.models.find((m) => m.id === transition.to_model_id)
                    ?.label
                }
              </strong>
              <p>{text}</p>
            </div>
            <button
              type="button"
              className="voice-btn"
              onClick={() => {
                setViewing(true);
                setFailure(undefined);
              }}
            >
              {t("voiceWorkspace.viewTransition")}
            </button>
          </div>
        )}
      <Modal
        open={!!target || viewing}
        onOpenChange={(open) => {
          if (!open && !busy) {
            onClose();
            setViewing(false);
            setFailure(undefined);
          }
        }}
        title={t("voiceWorkspace.transition")}
        closeLabel={t("voiceModels.byWorker.close")}
        variant="side"
      >
        <p className="text-sm text-mute">
          {t("voiceWorkspace.transitionHint")}
        </p>
        {failure && <ErrorAlert>{error(failure)}</ErrorAlert>}
        {target && !active ? (
          <>
            <strong>
              {catalogue.models.find((m) => m.id === target)?.label || target}
            </strong>
            {!preview && !failure && <p role="status">{t("app.loading")}</p>}
            {preview && (
              <>
                <p className="text-sm">
                  {t("voiceWorkspace.readyImpact", {
                    count: preview.workers_ready,
                  })}
                </p>
                {preview.workers_ready === 0 && (
                  <p className="text-sm text-mute">
                    {t("voiceWorkspace.waitingTarget")}
                  </p>
                )}
                <dl className="voice-modal-facts">
                  {(
                    [
                      ["referencesImpact", preview.references],
                      ["clipsImpact", preview.clips],
                      ["personsImpact", preview.persons],
                      ["othersImpact", preview.other_fingerprints],
                      ["validatedImpact", preview.validated_attributions],
                    ] as const
                  ).map(([key, value]) => (
                    <div key={key}>
                      <dt>{t(`voiceWorkspace.${key}`)}</dt>
                      <dd className="text-2xl font-semibold">{value}</dd>
                    </div>
                  ))}
                </dl>
                <label className="flex min-h-11 items-center gap-2 text-sm">
                  <input
                    type="checkbox"
                    checked={confirmed}
                    onChange={(e) => setConfirmed(e.target.checked)}
                  />
                  {t("voiceWorkspace.transitionConfirm", {
                    count: preview.references,
                  })}
                </label>
                <button
                  type="button"
                  className={buttonClass("primary")}
                  disabled={busy || !confirmed}
                  onClick={() => void run("start")}
                >
                  {t("voiceWorkspace.startTransition")}
                </button>
              </>
            )}
          </>
        ) : transition ? (
          <>
            <p role="status">{text}</p>
            <p>
              {t("voiceWorkspace.readyImpact", {
                count: transition.workers_ready,
              })}
            </p>
            {transition.jobs_failed > 0 && (
              <p className="text-bad-fg">
                {t("voiceWorkspace.transitionFailed", {
                  count: transition.jobs_failed,
                })}
              </p>
            )}
            {active && (
              <button
                type="button"
                className={buttonClass("secondary")}
                onClick={() => {
                  onClose();
                  setViewing(false);
                  onPrepare(transition.to_model_id);
                }}
              >
                {t("voiceWorkspace.prepareWorkers")}
              </button>
            )}
            {transition.phase !== "cancelled" && (
              <button
                type="button"
                className={buttonClass("primary")}
                disabled={busy}
                onClick={() => void run("resume")}
              >
                {t("voiceWorkspace.resume")}
              </button>
            )}
            {active && (
              <button
                type="button"
                className={buttonClass("danger")}
                disabled={busy}
                onClick={() => setCancelling(true)}
              >
                {t("voiceWorkspace.cancelTransition")}
              </button>
            )}
          </>
        ) : null}
        <ConfirmDialog
          open={cancelling}
          onOpenChange={setCancelling}
          title={t("voiceWorkspace.cancelTransition")}
          cancelLabel={t("voiceModels.cancelDialog")}
          confirmLabel={t("voiceWorkspace.cancelTransition")}
          danger
          busy={busy}
          onConfirm={() => void run("cancel")}
        >
          <p>{t("voiceWorkspace.transitionPreparing")}</p>
          {failure && <ErrorAlert>{error(failure)}</ErrorAlert>}
        </ConfirmDialog>
      </Modal>
    </>
  );
}
