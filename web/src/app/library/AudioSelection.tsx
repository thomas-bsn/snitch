import { useCallback, useEffect, useRef, useState } from "react";
import type { components } from "../../api.gen";
import { api, errorCode, mutation } from "../api/client";
import { ErrorAlert, StatusNotice } from "../components/Alert";
import { buttonClass } from "../components/buttons";
import { useI18n } from "../i18n";
import type { MessageKey } from "../i18n/messages";
import { useSession } from "../session";

type Context = components["schemas"]["AudioSelectionContext"];
type Inventory = components["schemas"]["AudioTrackInventory"];
type Diagnostic = components["schemas"]["AudioSelectionDiagnostic"];
type Decision = components["schemas"]["AudioSelectionDecision"];
type Request = components["schemas"]["AudioSelectionRequest"];
type Preview = components["schemas"]["AudioSelectionPreview"];
export type AnalysisChoice = components["schemas"]["AudioAnalysisChoice"];
type Mode = Request["mode"];
type Action = Decision["action"];

/** What the person typed for one track; the request is derived from it, never stored. */
type TrackState = {
  picked: boolean;
  force: boolean;
  reason: string;
  decision?: Action;
};

const EMPTY: TrackState = { picked: false, force: false, reason: "" };

/** Codes after which the preview shown can no longer be trusted. */
const DROPS_PREVIEW = new Set([
  "audio_selection_stale",
  "audio_preview_expired",
  "audio_preview_changed",
]);

function diagnosticOf(ctx: Context | undefined, index: number) {
  return ctx?.diagnostics.find((d) => d.track_index === index);
}

/**
 * Builds the request and the reason it cannot be sent yet. The server validates everything
 * again; this only keeps the person from sending a choice that is known to be refused.
 * Only a current diagnostic constrains the choice: an out-of-date one is shown, not obeyed.
 */
export function buildSelection(
  ctx: Context,
  mode: Mode,
  tracks: Record<number, TrackState>,
  analysis: AnalysisChoice,
): {
  request: Request;
  blocker?: "needTrack" | "needDecision" | "forceNeeded";
  forceTrack?: number;
} {
  const decisions: Decision[] = [];
  const picked: number[] = [];
  let blocker: "needTrack" | "needDecision" | "forceNeeded" | undefined;
  let forceTrack: number | undefined;
  let kept = 0;
  for (const track of ctx.inventory) {
    const state = tracks[track.index] ?? {
      picked: false,
      force: false,
      reason: "",
    };
    const diagnostic = diagnosticOf(ctx, track.index);
    const current =
      diagnostic?.freshness === "current" ? diagnostic.status : undefined;
    if (mode === "manual") {
      if (state.picked) {
        picked.push(track.index);
        kept++;
      }
      if (current === "ambiguous")
        decisions.push({
          track_index: track.index,
          action: state.picked ? "include" : "exclude",
          forced: false,
          reason: state.reason,
        });
      else if (current === "no_speech" && state.picked) {
        if (state.force)
          decisions.push({
            track_index: track.index,
            action: "include",
            forced: true,
            reason: state.reason,
          });
        else if (!blocker) {
          blocker = "forceNeeded";
          forceTrack = track.index;
        }
      }
    } else if (current === "ambiguous") {
      if (state.decision === undefined) blocker ??= "needDecision";
      else {
        decisions.push({
          track_index: track.index,
          action: state.decision,
          forced: false,
          reason: state.reason,
        });
        if (state.decision === "include") kept++;
      }
    } else kept++;
  }
  if (kept === 0) blocker = "needTrack";
  return {
    request: {
      version: ctx.version,
      mode,
      track_indices: mode === "manual" ? picked : [],
      decisions,
      analysis,
    },
    blocker,
    forceTrack,
  };
}

/** A readable name for a step the preview announces, such as "transcription.audio.7". */
export function stepLabel(
  t: ReturnType<typeof useI18n>["t"],
  step: string,
): string {
  const match =
    /^(audio\.diagnostic|transcription\.audio|voice\.diarization|voice\.fingerprint)\.(\d+)$/.exec(
      step,
    );
  if (match) {
    const kind = {
      "audio.diagnostic": "diagnostic",
      "transcription.audio": "transcription",
      "voice.diarization": "diarization",
      "voice.fingerprint": "fingerprint",
    }[match[1]!]!;
    return t(`clip.audioSelection.stepName.${kind}` as MessageKey, {
      index: match[2]!,
    });
  }
  const known = [
    "preparation",
    "transcription",
    "indexing",
    "diarization",
    "fingerprints",
    "matching",
  ];
  return known.includes(step)
    ? t(`clip.analysis.step.${step}` as MessageKey)
    : step;
}

export type Flow = ReturnType<typeof useAudioSelectionFlow>;

/**
 * The whole track-choice journey of one clip: reads the inventory and diagnostics, keeps the
 * choice, asks the server for the preview, then starts with that preview's token. A change
 * of choice goes back to the form and forgets the preview and its confirmation, so no
 * confirmation ever covers anything but the exact preview that was shown.
 */
export function useAudioSelectionFlow({
  clipId,
  enabled,
  analysis,
  onStarted,
}: {
  clipId: number;
  enabled: boolean;
  analysis: AnalysisChoice;
  onStarted: () => void | Promise<void>;
}) {
  const { session } = useSession();
  const [ctx, setCtx] = useState<Context>();
  const [loading, setLoading] = useState(false);
  const [loadFailed, setLoadFailed] = useState(false);
  const [mode, setMode] = useState<Mode>("auto");
  const [tracks, setTracks] = useState<Record<number, TrackState>>({});
  const [shown, setShown] = useState<{ preview: Preview; request: Request }>();
  const [confirmed, setConfirmed] = useState(false);
  const [busy, setBusy] = useState(false);
  const [failure, setFailure] = useState<string>();

  const load = useCallback(async () => {
    setLoading(true);
    setLoadFailed(false);
    try {
      const r = await api.GET("/api/admin/clips/{id}/audio-selection", {
        params: { path: { id: clipId } },
      });
      if (r.data) setCtx(r.data);
      else setLoadFailed(true);
    } catch {
      setLoadFailed(true);
    } finally {
      setLoading(false);
    }
  }, [clipId]);

  useEffect(() => {
    if (enabled) void load();
  }, [enabled, load]);

  const available = !!ctx && ctx.enabled && ctx.inventory.length > 0;
  const built =
    ctx && available ? buildSelection(ctx, mode, tracks, analysis) : undefined;

  function back() {
    setShown(undefined);
    setConfirmed(false);
    setFailure(undefined);
  }
  function edit(
    next: (current: Record<number, TrackState>) => Record<number, TrackState>,
  ) {
    back();
    setTracks(next);
  }
  const patch = (index: number, change: Partial<TrackState>) =>
    edit((current) => ({
      ...current,
      [index]: { ...(current[index] ?? EMPTY), ...change },
    }));

  async function failWith(error: unknown) {
    const code = errorCode(error) ?? "unknown";
    setFailure(code);
    if (code === "audio_selection_stale") await load();
    if (DROPS_PREVIEW.has(code)) {
      setShown(undefined);
      setConfirmed(false);
    }
  }

  async function requestPreview() {
    if (!built || built.blocker) return;
    setBusy(true);
    setFailure(undefined);
    try {
      const r = await api.POST(
        "/api/admin/clips/{id}/audio-selection/preview",
        {
          params: {
            path: { id: clipId },
            header: mutation(session?.csrf_token),
          },
          body: built.request,
        },
      );
      if (r.data) {
        setShown({ preview: r.data, request: built.request });
        setConfirmed(false);
      } else await failWith(r.error);
    } catch {
      setFailure("unavailable");
    } finally {
      setBusy(false);
    }
  }

  async function start() {
    if (!shown) return;
    setBusy(true);
    setFailure(undefined);
    try {
      const r = await api.POST("/api/admin/clips/{id}/audio-selection", {
        params: { path: { id: clipId }, header: mutation(session?.csrf_token) },
        body: {
          selection: shown.request,
          preview_token: shown.preview.preview_token,
          confirm_losses: confirmed,
        },
      });
      if (r.data) {
        setShown(undefined);
        setConfirmed(false);
        await onStarted();
      } else await failWith(r.error);
    } catch {
      setFailure("unavailable");
    } finally {
      setBusy(false);
    }
  }

  return {
    ctx,
    loading,
    loadFailed,
    available,
    load,
    mode,
    tracks,
    built,
    shown,
    confirmed,
    busy,
    failure,
    setMode: (next: Mode) => {
      back();
      setMode(next);
    },
    patch,
    back,
    requestPreview,
    start,
    setConfirmed,
  };
}

function seconds(value: number) {
  return String(Math.round(value * 100) / 100);
}

function TrackMeta({ track }: { track: Inventory }) {
  const { t } = useI18n();
  const parts = [
    track.codec && t("clip.audioSelection.codec", { codec: track.codec }),
    track.channels > 0 &&
      t("clip.audioSelection.channels", { count: track.channels }),
    track.language &&
      t("clip.audioSelection.language", { language: track.language }),
    track.start_seconds !== 0 &&
      t("clip.audioSelection.offset", {
        seconds: seconds(track.start_seconds),
      }),
    track.duration_seconds != null &&
      t("clip.audioSelection.duration", {
        seconds: seconds(track.duration_seconds),
      }),
  ].filter(Boolean);
  return <>{parts.join(" · ")}</>;
}

function DiagnosticLine({ diagnostic }: { diagnostic?: Diagnostic }) {
  const { t } = useI18n();
  if (!diagnostic) return <>{t("clip.audioSelection.noDiagnostic")}</>;
  const text = `${t(`clip.audioSelection.diagnostic.${diagnostic.status}` as MessageKey)} · ${t(`clip.audioSelection.freshness.${diagnostic.freshness}` as MessageKey)}`;
  return (
    <>
      <strong className="font-medium text-ink">{text}</strong>
      {diagnostic.cause &&
        ` · ${t("clip.audioSelection.cause", { cause: diagnostic.cause })}`}
    </>
  );
}

/**
 * The "Audio tracks" field: Auto, or any number of tracks by their real indices, each with
 * what is known about it and its diagnostic. No position stands for a role.
 */
export function AudioTrackChoice({
  flow,
  idPrefix,
}: {
  flow: Flow;
  idPrefix: string;
}) {
  const { t } = useI18n();
  const { ctx, mode, tracks, built } = flow;
  if (flow.loading && !ctx)
    return (
      <p role="status" className="m-0 text-sm text-mute">
        {t("clip.audioSelection.loading")}
      </p>
    );
  if (flow.loadFailed)
    return (
      <div className="flex flex-col gap-2">
        <ErrorAlert>{t("clip.audioSelection.loadFailed")}</ErrorAlert>
        <button
          type="button"
          onClick={() => void flow.load()}
          className={buttonClass("secondary")}
        >
          {t("clip.audioSelection.reload")}
        </button>
      </div>
    );
  if (!ctx || !flow.available) return null;
  const blocker = built?.blocker;
  return (
    <fieldset
      className="m-0 flex flex-col gap-2 border-0 p-0"
      aria-busy={flow.busy}
    >
      <legend className="mb-1 text-sm font-medium">
        {t("clip.audioSelection.legend")}
      </legend>
      {(["auto", "manual"] as Mode[]).map((value) => (
        <label key={value} className="flex min-h-11 items-center gap-2">
          <input
            type="radio"
            name={`${idPrefix}-audio-mode`}
            checked={mode === value}
            onChange={() => flow.setMode(value)}
          />
          {t(
            value === "auto"
              ? "clip.audioSelection.modeAuto"
              : "clip.audioSelection.modeManual",
          )}
        </label>
      ))}
      {mode === "auto" && (
        <p className="m-0 text-sm text-mute">
          {t("clip.audioSelection.autoHint")}
        </p>
      )}
      <ul
        aria-label={t("clip.audioSelection.tracksLabel")}
        className="m-0 flex list-none flex-col gap-2 p-0"
      >
        {ctx.inventory.map((track) => {
          const state = tracks[track.index];
          const diagnostic = diagnosticOf(ctx, track.index);
          const ambiguous =
            diagnostic?.freshness === "current" &&
            diagnostic.status === "ambiguous";
          const noSpeech =
            diagnostic?.freshness === "current" &&
            diagnostic.status === "no_speech";
          const id = `${idPrefix}-track-${track.index}`;
          const name = `${t("clip.audioSelection.track", { index: track.index })} — ${track.title || t("clip.audioSelection.untitled")}`;
          return (
            <li key={track.index} className="rounded-xl border border-line p-3">
              <div className="flex items-start gap-2">
                {mode === "manual" ? (
                  <input
                    id={id}
                    type="checkbox"
                    className="mt-1"
                    checked={!!state?.picked}
                    aria-describedby={`${id}-info`}
                    onChange={(e) =>
                      flow.patch(track.index, { picked: e.target.checked })
                    }
                  />
                ) : null}
                <div className="flex min-w-0 flex-1 flex-col gap-1">
                  {mode === "manual" ? (
                    <label htmlFor={id} className="min-h-6 font-medium">
                      {name}
                    </label>
                  ) : (
                    <span className="font-medium">{name}</span>
                  )}
                  <span id={`${id}-info`} className="text-xs text-mute">
                    <TrackMeta track={track} />
                    <span className="block">
                      <DiagnosticLine diagnostic={diagnostic} />
                    </span>
                  </span>
                  {mode === "auto" && ambiguous && (
                    <div
                      role="radiogroup"
                      aria-label={t("clip.audioSelection.decisionLabel", {
                        index: track.index,
                      })}
                      className="flex flex-wrap gap-4"
                    >
                      {(["include", "exclude"] as Action[]).map((action) => (
                        <label
                          key={action}
                          className="flex min-h-11 items-center gap-2 text-sm"
                        >
                          <input
                            type="radio"
                            name={`${id}-decision`}
                            checked={state?.decision === action}
                            onChange={() =>
                              flow.patch(track.index, { decision: action })
                            }
                          />
                          {t(
                            action === "include"
                              ? "clip.audioSelection.include"
                              : "clip.audioSelection.exclude",
                          )}
                        </label>
                      ))}
                    </div>
                  )}
                  {mode === "manual" && noSpeech && state?.picked && (
                    <label className="flex min-h-11 items-center gap-2 text-sm">
                      <input
                        type="checkbox"
                        checked={!!state.force}
                        onChange={(e) =>
                          flow.patch(track.index, { force: e.target.checked })
                        }
                      />
                      {t("clip.audioSelection.force")}
                    </label>
                  )}
                  {((mode === "manual" &&
                    ((noSpeech && state?.picked && state.force) ||
                      ambiguous)) ||
                    (mode === "auto" && ambiguous && state?.decision)) && (
                    <label className="flex flex-col gap-1 text-sm">
                      {t("clip.audioSelection.forceReason")}
                      <input
                        type="text"
                        value={state?.reason ?? ""}
                        maxLength={500}
                        onChange={(e) =>
                          flow.patch(track.index, { reason: e.target.value })
                        }
                        className="min-h-11 rounded-md border border-line bg-bg px-3"
                      />
                    </label>
                  )}
                </div>
              </div>
            </li>
          );
        })}
      </ul>
      {blocker && (
        <p role="note" className="m-0 text-sm text-mute">
          {blocker === "forceNeeded"
            ? t("clip.audioSelection.forceNeeded", {
                index: built?.forceTrack ?? 0,
              })
            : t(`clip.audioSelection.${blocker}` as MessageKey)}
        </p>
      )}
    </fieldset>
  );
}

/** The effects of the choice, the losses to confirm, and the start bound to this preview. */
export function AudioSelectionPreviewView({ flow }: { flow: Flow }) {
  const { t, error } = useI18n();
  const heading = useRef<HTMLHeadingElement>(null);
  const shown = flow.shown;
  useEffect(() => {
    if (shown) heading.current?.focus();
  }, [shown]);
  if (!shown) return null;
  const { effects } = shown.preview;
  const losses = effects.losses;
  const needsConfirm = losses.confirmation_required;
  const list = (items: string[], empty: string) =>
    items.length === 0 ? (
      <p className="m-0 text-sm text-mute">{empty}</p>
    ) : (
      <ul className="m-0 flex list-disc flex-col gap-1 pl-5 text-sm">
        {items.map((item) => (
          <li key={item}>{stepLabel(t, item)}</li>
        ))}
      </ul>
    );
  return (
    <section
      aria-labelledby="audio-preview-title"
      className="flex flex-col gap-3"
    >
      <h3
        id="audio-preview-title"
        ref={heading}
        tabIndex={-1}
        className="m-0 text-base font-semibold"
      >
        {t("clip.audioSelection.previewTitle")}
      </h3>
      <div className="flex flex-col gap-1">
        <h4 className="m-0 text-sm font-medium">
          {t("clip.audioSelection.tracksConcerned")}
        </h4>
        <p className="m-0 text-sm">
          {effects.track_indices
            .map((i) => t("clip.audioSelection.track", { index: i }))
            .join(", ")}
        </p>
      </div>
      <div className="flex flex-col gap-1">
        <h4 className="m-0 text-sm font-medium">
          {t("clip.audioSelection.added")}
        </h4>
        {list(effects.added_steps, t("clip.audioSelection.nothingAdded"))}
      </div>
      <div className="flex flex-col gap-1">
        <h4 className="m-0 text-sm font-medium">
          {t("clip.audioSelection.invalidated")}
        </h4>
        {list(
          effects.invalidated_steps,
          t("clip.audioSelection.nothingInvalidated"),
        )}
      </div>
      <div className="flex flex-col gap-1">
        <h4 className="m-0 text-sm font-medium">
          {t("clip.audioSelection.lossesTitle")}
        </h4>
        {needsConfirm ? (
          <ul className="m-0 flex list-disc flex-col gap-1 pl-5 text-sm">
            {losses.correction_count > 0 && (
              <li>
                {t("clip.audioSelection.lossCorrections", {
                  count: losses.correction_count,
                })}
              </li>
            )}
            {losses.validated_voice_count > 0 && (
              <li>
                {t("clip.audioSelection.lossValidations", {
                  count: losses.validated_voice_count,
                })}
              </li>
            )}
            {losses.replaced_track_indices.length > 0 && (
              <li>
                {t("clip.audioSelection.lossTracks", {
                  tracks: losses.replaced_track_indices.join(", "),
                })}
              </li>
            )}
          </ul>
        ) : (
          <p className="m-0 text-sm text-mute">
            {t("clip.audioSelection.noLoss")}
          </p>
        )}
      </div>
      {!effects.worker_compatible && (
        <StatusNotice>{t("clip.audioSelection.workerWait")}</StatusNotice>
      )}
      {needsConfirm && (
        <div className="flex flex-col gap-1">
          <label className="flex min-h-11 items-center gap-2 text-sm font-medium">
            <input
              type="checkbox"
              checked={flow.confirmed}
              onChange={(e) => flow.setConfirmed(e.target.checked)}
            />
            {t("clip.audioSelection.confirmLosses")}
          </label>
          <p className="m-0 text-xs text-mute">
            {t("clip.audioSelection.confirmScope")}
          </p>
        </div>
      )}
      {flow.failure && <ErrorAlert>{error(flow.failure)}</ErrorAlert>}
      <div className="flex justify-end gap-2">
        <button
          type="button"
          disabled={flow.busy}
          onClick={flow.back}
          className={buttonClass("secondary")}
        >
          {t("clip.audioSelection.change")}
        </button>
        <button
          type="button"
          disabled={flow.busy || (needsConfirm && !flow.confirmed)}
          onClick={() => void flow.start()}
          className={buttonClass(needsConfirm ? "danger" : undefined)}
        >
          {flow.busy
            ? t("clip.audioSelection.starting")
            : t("clip.audioSelection.start")}
        </button>
      </div>
    </section>
  );
}

/** The preview button and its failure, shown under the choice while no preview exists. */
export function AudioPreviewAction({ flow }: { flow: Flow }) {
  const { t, error } = useI18n();
  return (
    <>
      {flow.failure && <ErrorAlert>{error(flow.failure)}</ErrorAlert>}
      <button
        type="button"
        disabled={flow.busy || !flow.built || !!flow.built.blocker}
        onClick={() => void flow.requestPreview()}
        className={buttonClass()}
      >
        {flow.busy
          ? t("clip.audioSelection.previewing")
          : t("clip.audioSelection.preview")}
      </button>
    </>
  );
}
