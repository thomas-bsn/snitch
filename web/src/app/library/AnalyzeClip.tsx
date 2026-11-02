import { Menu } from "@base-ui/react/menu";
import { ChevronDown } from "lucide-react";
import { Link } from "react-router";
import { useCallback, useEffect, useRef, useState } from "react";
import type { components } from "../../api.gen";
import { api, errorCode, mutation } from "../api/client";
import { useEventStream } from "../api/useEventStream";
import { ErrorAlert } from "../components/Alert";
import { buttonClass } from "../components/buttons";
import { useI18n } from "../i18n";
import type { MessageKey } from "../i18n/messages";
import { IdentificationBadge } from "../admin/queue/AnalysisBoard";
import { isAdministrator } from "../roles";
import { ReanalyzeClip } from "./ReanalyzeClip";
import { useSession } from "../session";

type AdminRun = components["schemas"]["ClipAnalysisRun"];
type View = components["schemas"]["ClipAnalysisView"];
// Administrators get the full run; every other reader gets the redacted view.
type Run = AdminRun | View;
type MediaTrack = components["schemas"]["MediaTrack"];
type Scope = AdminRun["scope"];
type Step = AdminRun["steps"][number];

/**
 * The primary action of a clip: one click starts a durable analysis on the server. The
 * state of each step, and why it waits, comes from the server, so it stays the same after
 * the page is closed and opened again. Administrators drive it; every other user who can
 * open the clip follows it read-only (status and reason of each step, live).
 */
export function AnalyzeClip({
  clipId,
  audioTracks,
  onChanged,
}: {
  clipId: number;
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
  const [reanalyzing, setReanalyzing] = useState(false);
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
    if (!user) return;
    const r = admin
      ? await api.GET("/api/admin/clips/{id}/analysis", {
          params: { path: { id: clipId } },
        })
      : await api.GET("/api/clips/{id}/analysis", {
          params: { path: { id: clipId } },
        });
    if (r.data) apply(r.data.run ?? null);
    setLoaded(true);
  }, [user, admin, clipId, apply]);

  useEffect(() => {
    void refresh();
  }, [refresh]);
  const active = run?.status === "active";
  // The server pushes each change as a snapshot; polling only covers a stream that is down.
  const stream = useEventStream<{ run: Run | null }>(
    user
      ? admin
        ? `/api/admin/clips/${clipId}/analysis/events`
        : `/api/clips/${clipId}/analysis/events`
      : null,
    useCallback(
      (snapshot) => {
        apply(snapshot.run ?? null);
        setLoaded(true);
      },
      [apply],
    ),
    refresh,
  );

  if (!user || !loaded) return null;
  // A reader without a run has nothing to follow; the controls are administrators'.
  if (!admin && !run) return null;

  async function start(scope: Scope) {
    setBusy(true);
    setFailure(undefined);
    try {
      const r = await api.POST("/api/admin/clips/{id}/analysis", {
        params: {
          path: { id: clipId },
          header: mutation(session?.csrf_token),
        },
        body: { scope },
      });
      if (r.data) apply(r.data.run ?? null);
      else setFailure(errorCode(r.error) ?? "unknown");
    } catch {
      setFailure("unavailable");
    } finally {
      setBusy(false);
    }
  }

  async function act(
    request: () => Promise<{ data?: { run?: Run | null }; error?: unknown }>,
  ) {
    setBusy(true);
    setFailure(undefined);
    try {
      const r = await request();
      if (r.data) apply(r.data.run ?? null);
      else setFailure(errorCode(r.error) ?? "unknown");
    } catch {
      setFailure("unavailable");
    } finally {
      setBusy(false);
    }
  }

  const cancel = () =>
    act(() =>
      api.POST("/api/admin/clips/{id}/analysis/cancel", {
        params: {
          path: { id: clipId },
          header: mutation(session?.csrf_token),
        },
      }),
    );

  const retry = (step: Step["step"]) =>
    act(() =>
      api.POST("/api/admin/clips/{id}/analysis/steps/{step}/retry", {
        params: {
          path: { id: clipId, step },
          header: mutation(session?.csrf_token),
        },
      }),
    );

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

  const selection =
    admin &&
    run?.steps.some((s) => s.reason === "audio_track_selection_required");
  return (
    <section
      aria-label={t("clip.analysis.label")}
      className="flex flex-col gap-3 rounded-xl border border-line p-3"
    >
      {admin && active && (
        <div>
          <button
            type="button"
            disabled={busy}
            onClick={() => void cancel()}
            className={buttonClass("secondary")}
          >
            {t("clip.analysis.cancel")}
          </button>
        </div>
      )}
      {admin && !active && (
        <div className="flex items-center gap-1">
          <button
            type="button"
            disabled={busy}
            onClick={() => void start("full")}
            className={buttonClass()}
          >
            {t("clip.analysis.full")}
          </button>
          <Menu.Root>
            <Menu.Trigger
              aria-label={t("clip.analysis.more")}
              disabled={busy}
              className={buttonClass("secondary")}
            >
              <ChevronDown aria-hidden="true" className="size-4" />
            </Menu.Trigger>
            <Menu.Portal>
              <Menu.Positioner align="start" sideOffset={6} className="z-50">
                <Menu.Popup className="w-56 rounded-xl border border-line bg-surface p-1.5 text-ink shadow-pop outline-none">
                  <Menu.Item
                    onClick={() => void start("transcription")}
                    className="flex min-h-11 cursor-pointer items-center rounded-lg px-3 text-sm outline-none data-[highlighted]:bg-side"
                  >
                    {t("clip.analysis.transcription")}
                  </Menu.Item>
                  <Menu.Item
                    onClick={() => void start("voice")}
                    className="flex min-h-11 cursor-pointer items-center rounded-lg px-3 text-sm outline-none data-[highlighted]:bg-side"
                  >
                    {t("clip.analysis.voice")}
                  </Menu.Item>
                  <Menu.Item
                    onClick={() => setReanalyzing(true)}
                    className="flex min-h-11 cursor-pointer items-center rounded-lg px-3 text-sm outline-none data-[highlighted]:bg-side"
                  >
                    {t("clip.reanalyze.menu")}
                  </Menu.Item>
                </Menu.Popup>
              </Menu.Positioner>
            </Menu.Portal>
          </Menu.Root>
        </div>
      )}
      <p
        role={stream === "forbidden" ? "alert" : "status"}
        hidden={stream === "connecting"}
        className={stream === "live" ? "sr-only" : "m-0 text-xs text-mute"}
      >
        {stream === "live"
          ? t("clip.analysis.live")
          : stream === "forbidden"
            ? t("clip.analysis.liveLost")
            : t("clip.analysis.liveOff")}
      </p>
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
              {"forced" in step && step.forced && (
                <span className="ml-1 text-xs text-mute">
                  {t("clip.analysis.forced")}
                </span>
              )}
              <span className="block text-xs text-mute">
                {t(`clip.analysis.reason.${step.reason}` as MessageKey)}
              </span>
              {"profile" in step && step.profile && (
                <span className="block text-xs text-mute">
                  {t("clip.analysis.profile", { profile: step.profile })}
                </span>
              )}
              {"job_id" in step &&
                (step.job_id !== null || step.status === "failed") && (
                  <span className="mt-1 flex flex-wrap gap-2">
                    {step.job_id !== null && (
                      <Link
                        to="/admin/queue"
                        className="text-xs text-accent underline"
                      >
                        {t("clip.analysis.viewTask", { id: step.job_id })}
                      </Link>
                    )}
                    {step.status === "failed" && run.status !== "cancelled" && (
                      <button
                        type="button"
                        disabled={busy}
                        onClick={() => void retry(step.step)}
                        className={buttonClass("secondary")}
                      >
                        {t("clip.analysis.retry", {
                          step: t(`clip.analysis.step.${step.step}`),
                        })}
                      </button>
                    )}
                  </span>
                )}
            </li>
          ))}
        </ol>
      )}
      {run && <IdentificationBadge run={run} />}
      {run?.status === "cancelled" && (
        <p className="m-0 text-sm text-mute">{t("clip.analysis.cancelled")}</p>
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
      {admin && (
        <ReanalyzeClip
          clipId={clipId}
          open={reanalyzing}
          onOpenChange={setReanalyzing}
          onStarted={apply}
        />
      )}
    </section>
  );
}
