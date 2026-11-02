import { useCallback, useEffect, useRef, useState } from "react";
import type { components } from "../../api.gen";
import { api, errorCode, mutation } from "../api/client";
import { ErrorAlert } from "../components/Alert";
import { buttonClass } from "../components/buttons";
import { useI18n } from "../i18n";
import type { MessageKey } from "../i18n/messages";
import { isAdministrator } from "../roles";
import { useSession } from "../session";

type Run = components["schemas"]["ClipAnalysisRun"];
type MediaTrack = components["schemas"]["MediaTrack"];

const pollMs = 3000;

/**
 * The primary action of a clip: one click starts a durable analysis on the server. The
 * state of each step, and why it waits, comes from the server, so it stays the same after
 * the page is closed and opened again. Administrators only.
 */
export function AnalyzeClip({
  clipId,
  hasTranscription,
  audioTracks,
  onChanged,
}: {
  clipId: number;
  hasTranscription: boolean;
  audioTracks: MediaTrack[];
  onChanged: () => void;
}) {
  const { t, error } = useI18n();
  const { user, session } = useSession();
  const admin = !!user && isAdministrator(user);
  const [run, setRun] = useState<Run | null>(null);
  const [loaded, setLoaded] = useState(false);
  const [failure, setFailure] = useState<string>();
  const [busy, setBusy] = useState(false);
  const [track, setTrack] = useState("");
  const wasActive = useRef(false);

  const apply = useCallback(
    (next: Run | null) => {
      setRun(next);
      const active = next?.status === "active";
      if (wasActive.current && !active) onChanged();
      wasActive.current = active;
    },
    [onChanged],
  );

  const refresh = useCallback(async () => {
    if (!admin) return;
    const r = await api.GET("/api/admin/clips/{id}/analysis", {
      params: { path: { id: clipId } },
    });
    if (r.data) apply(r.data.run ?? null);
    setLoaded(true);
  }, [admin, clipId, apply]);

  useEffect(() => {
    void refresh();
  }, [refresh]);
  const active = run?.status === "active";
  useEffect(() => {
    if (!active) return;
    const timer = setInterval(() => void refresh(), pollMs);
    return () => clearInterval(timer);
  }, [active, refresh]);

  if (!admin || !loaded) return null;
  const finished = run && !active;
  if (!run && hasTranscription) return null;
  if (finished && run.status === "completed" && hasTranscription) return null;

  async function start() {
    setBusy(true);
    setFailure(undefined);
    try {
      const r = await api.POST("/api/admin/clips/{id}/analysis", {
        params: {
          path: { id: clipId },
          header: mutation(session?.csrf_token),
        },
        body: {},
      });
      if (r.data) apply(r.data.run ?? null);
      else setFailure(errorCode(r.error) ?? "unknown");
    } catch {
      setFailure("unavailable");
    } finally {
      setBusy(false);
    }
  }

  async function chooseTrack() {
    setBusy(true);
    setFailure(undefined);
    try {
      const r = await api.PUT("/api/admin/clips/{id}/audio-track", {
        params: {
          path: { id: clipId },
          header: mutation(session?.csrf_token),
        },
        body: { audio_stream_index: Number(track) },
      });
      if (r.error) setFailure(errorCode(r.error) ?? "unknown");
      else await refresh();
    } catch {
      setFailure("unavailable");
    } finally {
      setBusy(false);
    }
  }

  const selection = run?.steps.some(
    (s) => s.reason === "audio_track_selection_required",
  );
  return (
    <section
      aria-label={t("clip.analysis.label")}
      className="flex flex-col gap-3 rounded-xl border border-line p-3"
    >
      {(!run || !active) && (
        <div>
          <button
            type="button"
            disabled={busy}
            onClick={() => void start()}
            className={buttonClass()}
          >
            {run ? t("clip.analysis.again") : t("clip.analysis.start")}
          </button>
        </div>
      )}
      {run && (
        <ol
          aria-label={t("clip.analysis.steps")}
          aria-live="polite"
          className="m-0 flex list-none flex-col gap-2 p-0"
        >
          {run.steps.map((step) => (
            <li key={step.step} className="text-sm">
              <span className="font-medium">
                {t(`clip.analysis.step.${step.step}`)}
              </span>
              {" · "}
              <span>{t(`clip.analysis.status.${step.status}`)}</span>
              <span className="block text-xs text-mute">
                {t(`clip.analysis.reason.${step.reason}` as MessageKey)}
              </span>
            </li>
          ))}
        </ol>
      )}
      {selection && (
        <div className="flex gap-2">
          <select
            aria-label={t("clip.analysis.track")}
            value={track}
            onChange={(e) => setTrack(e.target.value)}
            className="min-h-11 flex-1 rounded-md border border-line bg-bg px-3 text-sm"
          >
            <option value="">{t("clip.analysis.trackPlaceholder")}</option>
            {audioTracks.map((tr) => (
              <option key={tr.index} value={tr.index}>
                {tr.title ?? `#${tr.index}`}
                {tr.language ? ` (${tr.language})` : ""}
              </option>
            ))}
          </select>
          <button
            type="button"
            disabled={busy || track === ""}
            onClick={() => void chooseTrack()}
            className={buttonClass("secondary")}
          >
            {t("clip.analysis.trackConfirm")}
          </button>
        </div>
      )}
      {failure && <ErrorAlert>{error(failure)}</ErrorAlert>}
    </section>
  );
}
