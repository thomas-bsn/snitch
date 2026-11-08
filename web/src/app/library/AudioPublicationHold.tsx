import { useCallback, useEffect, useState } from "react";
import type { components } from "../../api.gen";
import { api, errorCode, mutation } from "../api/client";
import { ErrorAlert } from "../components/Alert";
import { buttonClass } from "../components/buttons";
import { useI18n } from "../i18n";
import { useSession } from "../session";

type Preview = components["schemas"]["AudioPublicationPreview"];

/**
 * A finished audio plan whose publication is held: it would replace human work the administrator has not
 * confirmed (or that changed since). The current transcription stays visible; this shows the losses as they
 * are now and records a new confirmation, after which the server publishes by itself.
 */
export function AudioPublicationHold({
  clipId,
  planId,
  onConfirmed,
}: {
  clipId: number;
  planId: number;
  onConfirmed: () => void;
}) {
  const { t, error } = useI18n();
  const { session } = useSession();
  const [preview, setPreview] = useState<Preview>();
  const [failure, setFailure] = useState<string>();
  const [busy, setBusy] = useState(false);

  const load = useCallback(async () => {
    const r = await api.GET(
      "/api/admin/clips/{id}/audio-plans/{plan_id}/publication",
      { params: { path: { id: clipId, plan_id: planId } } },
    );
    if (r.data) setPreview(r.data);
    else setFailure(errorCode(r.error) ?? "unknown");
  }, [clipId, planId]);

  useEffect(() => {
    void load();
  }, [load]);

  async function confirm() {
    if (!preview) return;
    setBusy(true);
    setFailure(undefined);
    try {
      const r = await api.POST(
        "/api/admin/clips/{id}/audio-plans/{plan_id}/publication",
        {
          params: {
            path: { id: clipId, plan_id: planId },
            header: mutation(session?.csrf_token),
          },
          body: { guard: preview.guard, confirm_losses: true },
        },
      );
      if (r.data) {
        setPreview(r.data);
        onConfirmed();
      } else {
        setFailure(errorCode(r.error) ?? "unknown");
        // The history moved again: show it as it is now.
        void load();
      }
    } catch {
      setFailure("unavailable");
    } finally {
      setBusy(false);
    }
  }

  return (
    <div
      role="group"
      aria-label={t("clip.analysis.publication.title")}
      className="flex flex-col gap-2 rounded-xl border border-accent bg-accent-soft p-3 text-sm"
    >
      <p className="m-0 font-medium">{t("clip.analysis.publication.title")}</p>
      {!preview && !failure && (
        <p className="m-0 text-mute">
          {t("clip.analysis.publication.loading")}
        </p>
      )}
      {preview && (
        <>
          <p className="m-0">
            {preview.losses.confirmation_required
              ? t("clip.analysis.publication.losses", {
                  corrections: preview.losses.correction_count,
                  validations: preview.losses.validated_voice_count,
                })
              : t("clip.analysis.publication.none")}
          </p>
          {preview.losses.replaced_track_indices.length > 0 && (
            <p className="m-0 text-xs text-mute">
              {t("clip.analysis.publication.tracks", {
                tracks: preview.losses.replaced_track_indices.join(", "),
              })}
            </p>
          )}
          {preview.confirmed ? (
            <p className="m-0 text-xs text-mute" role="status">
              {t("clip.analysis.publication.confirmed")}
            </p>
          ) : (
            <button
              type="button"
              disabled={busy}
              onClick={() => void confirm()}
              className={buttonClass()}
            >
              {t("clip.analysis.publication.confirm")}
            </button>
          )}
        </>
      )}
      {failure && <ErrorAlert>{error(failure)}</ErrorAlert>}
    </div>
  );
}
