import { Tabs } from "@base-ui/react/tabs";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import {
  Link,
  useLocation,
  useNavigate,
  useParams,
  useSearchParams,
} from "react-router";
import { useFormatDate } from "../admin/format";
import { useLoaded } from "../api/loaded";
import { ErrorAlert } from "../components/Alert";
import { useDocumentTitle } from "../components/useDocumentTitle";
import { useI18n } from "../i18n";
import { isAdministrator } from "../roles";
import { useSession } from "../session";
import { EditMetadata, MetadataDisclosure } from "./ClipMetadata";
import { ClipPlayer } from "./ClipPlayer";
import { ClipStatePill } from "./ClipStatePill";
import { HistoryTab } from "./HistoryTab";
import { Retranscribe } from "./Retranscribe";
import { ProgressBanner } from "./ProgressBanner";
import { clipSteps, legacyAnalysisState } from "./progress";
import { voiceToDecide } from "./speakerState";
import type { PillState } from "./AnalysisPill";
import { TranscriptTab } from "./TranscriptTab";
import { RelaunchVoice } from "./RelaunchVoice";
import { VoicesTab } from "./VoicesTab";
import { formatTimecode, parseMoment } from "./clipFormat";
import { clipTitle, fetchClip, fetchClipVoice, type Clip } from "./libraryApi";
import { AnalyzeClip, type Run } from "./AnalyzeClip";
import { ReanalyzeClip } from "./ReanalyzeClip";
import { buttonClass } from "../components/buttons";
import { backAddress, hasOrigin } from "./clipOrigin";
import {
  rememberLastSeen,
  useSequence,
  type Neighbour,
} from "./clipNeighbours";
import { ClipPager, ClipStrip, NextCountdown, useStripOpen } from "./ClipStrip";

// The clip opened by an arrow or by the chain starts by itself. Kept outside the
// history so that a reload or the back button never plays on its own.
let autoplayClip: number | null = null;

const AUTO_NEXT_KEY = "snitch.clip.autoNext";
function storedAutoNext() {
  try {
    return window.localStorage.getItem(AUTO_NEXT_KEY) === "on";
  } catch {
    return false;
  }
}

const tabClass =
  "-mb-px inline-flex h-11 shrink-0 cursor-pointer items-center gap-2 border-b-2 border-transparent px-3 font-medium whitespace-nowrap text-mute outline-none hover:text-ink data-[active]:border-accent data-[active]:text-ink md:h-10";

type TabName = "transcript" | "voices" | "history";

function ClipView({
  clip,
  onChanged,
  next,
  onNext,
}: {
  clip: Clip;
  onChanged: () => void;
  next?: Neighbour;
  onNext: () => void;
}) {
  const { t } = useI18n();
  const { user } = useSession();
  const admin = !!user && isAdministrator(user);
  const formatDate = useFormatDate();
  const location = useLocation();
  const [params, setParams] = useSearchParams();
  const videoRef = useRef<HTMLVideoElement | null>(null);
  const [startAt] = useState(() => parseMoment(params.get("t")));
  const [playhead, setPlayhead] = useState(startAt);
  const [tab, setTab] = useState<TabName>("transcript");
  const [autoPlay] = useState(() => {
    const yes = autoplayClip === clip.id;
    autoplayClip = null;
    return yes;
  });
  const [autoNext, setAutoNext] = useState(storedAutoNext);
  const [countdown, setCountdown] = useState(false);
  const voice = useLoaded(
    useCallback(() => fetchClipVoice(clip.id), [clip.id]),
  );
  // The analysis in progress, as the panel's stream last said it: the stepper reads its steps
  // from it, so that it moves with the text. It holds until the clip read after the change
  // lands (a finished analysis then speaks through the clip again), and only for a run seen
  // in progress on this page: an old finished run says nothing of the clip as it is now.
  const [live, setLive] = useState<{ run: Run; clip: Clip } | null>(null);
  const latestClip = useRef(clip);
  latestClip.current = clip;
  const followRun = useCallback((next: Run | null) => {
    setLive((current) =>
      next && (next.status === "active" || current)
        ? { run: next, clip: latestClip.current }
        : null,
    );
  }, []);
  const liveRun =
    live &&
    live.run.status !== "cancelled" &&
    (live.run.status === "active" || live.clip === clip)
      ? live.run
      : null;
  useDocumentTitle(clipTitle(clip));

  // The drawing follows the track the transcript and the voices come from; without peaks
  // (an analysis made before they existed) nothing is drawn.
  const waveform = useMemo(
    () =>
      clip.waveforms?.find(
        (entry) =>
          entry.audio_stream_index === clip.transcription?.audio_stream_index,
      ),
    [clip.waveforms, clip.transcription?.audio_stream_index],
  );
  const segments = useMemo(
    () => clip.transcription?.segments ?? [],
    [clip.transcription],
  );

  // The address keeps the moment being watched, so that sharing it shares that instant.
  const keepMoment = useCallback(
    (seconds: number) => {
      setParams(
        (current) => {
          const next = new URLSearchParams(current);
          if (seconds > 0) next.set("t", String(seconds));
          else next.delete("t");
          return next;
        },
        { replace: true, state: location.state },
      );
    },
    [setParams, location.state],
  );

  function seek(seconds: number) {
    const video = videoRef.current;
    if (video) {
      video.currentTime = seconds;
      void video.play().catch(() => undefined);
    }
    setPlayhead(seconds);
    keepMoment(Math.floor(seconds));
  }

  // A decision changes the voices and the transcript: both are read again.
  const reloadVoices = voice.reload;
  const refreshVoices = useCallback(() => {
    onChanged();
    reloadVoices();
  }, [onChanged, reloadVoices]);

  const date = clip.recorded_at
    ? formatDate(clip.recorded_at)
    : t("clip.speaker.unknown");
  const voiceData = voice.state.status === "ready" ? voice.state.value : null;

  // Without transcription, the step of the analysis has never run: the launch is the page's way forward.
  const neverAnalysed = !clip.transcription;
  // Until the voices are read, they are taken as detected: no step is announced as never run.
  const hasVoices = voiceData ? voiceData.status !== "none" : true;
  const [launching, setLaunching] = useState(false);
  // A clip processed step by step has no run: its header still tells what its steps say.
  const legacyKind = legacyAnalysisState(clipSteps(clip, voiceData));
  const failedVoiceStep = voiceData?.steps.find(
    (s) => s.status === "failed" || s.status === "dead",
  )?.step;
  const legacy: PillState | null =
    legacyKind === "failed"
      ? {
          kind: "failed",
          step: (["diarization", "fingerprints", "matching"] as const).find(
            (name) => failedVoiceStep?.includes(name.slice(0, 6)),
          ),
        }
      : legacyKind
        ? { kind: legacyKind }
        : null;
  const toDecide = voiceData?.speakers.filter(voiceToDecide).length ?? 0;

  const metaLine = [
    t("clip.meta.owner", {
      owner: clip.owner_deleted ? t("library.deletedOwner") : clip.owner,
    }),
    date,
    typeof clip.duration_seconds === "number"
      ? formatTimecode(clip.duration_seconds)
      : null,
    t(clip.visibility === "shared" ? "clip.meta.shared" : "clip.meta.private"),
  ]
    .filter(Boolean)
    .join(" · ");

  return (
    <>
      <section className="flex flex-col gap-4 px-4 py-4 md:px-7 md:py-5">
        <div className="flex flex-col gap-2">
          <div className="flex flex-wrap items-start gap-3">
            <header className="min-w-0 flex-1">
              <h1 className="m-0 text-[22px] font-semibold break-words">
                {clipTitle(clip)}
              </h1>
              <p className="mt-1 mb-0 text-xs text-mute">
                {clip.display_title
                  ? t("metadata.fileKept", { name: clip.original_name })
                  : t("clip.meta.file", { name: clip.original_name })}
              </p>
            </header>
            <div className="flex flex-wrap items-start gap-3">
              <EditMetadata clip={clip} onChanged={onChanged} />
              <AnalyzeClip
                clipId={clip.id}
                audioTracks={
                  clip.tracks?.filter((t) => t.type === "audio") ?? []
                }
                hasTranscription={!neverAnalysed}
                hasVoices={hasVoices}
                onChanged={refreshVoices}
                onRun={followRun}
                legacy={legacy}
                summary={
                  <ProgressBanner clip={clip} voice={voiceData} run={liveRun} />
                }
              />
            </div>
          </div>
          <p className="m-0 text-mute">{metaLine}</p>
          {clip.sources && clip.sources.length > 0 && (
            <p className="m-0 text-mute">
              {t("clip.meta.sources", { sources: clip.sources.join(", ") })}
            </p>
          )}
        </div>

        <div className="grid gap-5 lg:grid-cols-[minmax(0,1fr)_minmax(0,1fr)]">
          <div className="flex min-w-0 flex-col gap-3 lg:sticky lg:top-4 lg:self-start">
            <ClipPlayer
              clipId={clip.id}
              videoRef={videoRef}
              startAt={startAt}
              onMoment={keepMoment}
              onPlayhead={setPlayhead}
              autoPlay={autoPlay}
              onEnded={() => {
                if (autoNext && next) setCountdown(true);
              }}
              overlay={
                countdown &&
                next && (
                  <NextCountdown
                    next={next}
                    onGo={() => {
                      setCountdown(false);
                      onNext();
                    }}
                    onCancel={() => setCountdown(false)}
                  />
                )
              }
            />
            {next && (
              <label className="-mt-1 inline-flex min-h-11 cursor-pointer items-center gap-2 self-end text-sm text-mute md:min-h-8">
                <input
                  type="checkbox"
                  role="switch"
                  checked={autoNext}
                  onChange={(event) => {
                    setAutoNext(event.target.checked);
                    try {
                      window.localStorage.setItem(
                        AUTO_NEXT_KEY,
                        event.target.checked ? "on" : "off",
                      );
                    } catch {
                      // Without storage the choice lasts for this clip only.
                    }
                  }}
                  className="size-4 accent-[var(--accent-solid)]"
                />
                {t("clip.nav.autoNext")}
              </label>
            )}
            <MetadataDisclosure clip={clip} />
            <ProgressBanner
              clip={clip}
              voice={voiceData}
              run={liveRun}
              assistiveOnly
            />
            {neverAnalysed && (
              <div className="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-accent bg-accent-soft p-3">
                <p className="m-0 font-medium text-ink">
                  {t("clip.neverAnalysed")}
                </p>
                {admin && (
                  <button
                    type="button"
                    onClick={() => setLaunching(true)}
                    className={buttonClass()}
                  >
                    {t("clip.analysis.launch")}
                  </button>
                )}
              </div>
            )}
            {admin && (
              <ReanalyzeClip
                clipId={clip.id}
                hasTranscription={!neverAnalysed}
                hasVoices={hasVoices}
                open={launching}
                onOpenChange={setLaunching}
                onStarted={() => onChanged()}
              />
            )}
          </div>

          <Tabs.Root
            value={tab}
            onValueChange={(value) => setTab(value as TabName)}
            className="flex min-w-0 flex-col rounded-xl border border-line"
          >
            <Tabs.List
              aria-label={t("clip.tabs.label")}
              tabIndex={0}
              className="flex gap-1 overflow-x-auto border-b border-line px-3"
            >
              <Tabs.Tab value="transcript" className={tabClass}>
                {t("clip.tabs.transcript")}
                {segments.length > 0 && (
                  <span className="text-xs text-mute">{segments.length}</span>
                )}
              </Tabs.Tab>
              <Tabs.Tab value="voices" className={tabClass}>
                {t("clip.tabs.voices")}
                {voiceData && voiceData.speakers.length > 0 && (
                  <span className="text-xs text-mute">
                    {voiceData.speakers.length}
                  </span>
                )}
                {toDecide > 0 && (
                  <span className="rounded-full bg-warn-bg px-1.5 text-xs text-warn-fg">
                    {t("clip.tabs.todo", { count: toDecide })}
                  </span>
                )}
              </Tabs.Tab>
              <Tabs.Tab value="history" className={tabClass}>
                {t("clip.tabs.history")}
              </Tabs.Tab>
            </Tabs.List>
            <Tabs.Panel
              value="transcript"
              className="max-h-[70vh] overflow-y-auto outline-none"
            >
              {admin && clip.transcription && (
                <div className="border-b border-line p-3">
                  <Retranscribe
                    id={clip.id}
                    stream={clip.transcription.audio_stream_index}
                  />
                </div>
              )}
              <TranscriptTab
                clipId={clip.id}
                jobId={clip.transcription?.transcription_job_id}
                speakers={clip.transcription?.local_speakers ?? []}
                onChanged={onChanged}
                segments={segments}
                playhead={playhead}
                onSeek={seek}
              />
            </Tabs.Panel>
            <Tabs.Panel
              value="voices"
              className="max-h-[70vh] overflow-y-auto outline-none"
            >
              <VoicesTab
                voice={voiceData}
                segments={segments}
                peaks={waveform?.peaks}
                duration={clip.duration_seconds}
                playhead={playhead}
                onSeek={seek}
                clipName={clipTitle(clip)}
                onChanged={refreshVoices}
                coverage={
                  clip.transcription?.speaker_coverage ??
                  clip.transcription?.speaker_coverage_by_track?.flatMap(
                    (track) => track.coverage,
                  )
                }
              />
              {voiceData && (
                <div className="flex flex-col gap-3 p-3">
                  <RelaunchVoice
                    clipId={clip.id}
                    voice={voiceData}
                    onScheduled={voice.reload}
                  />
                </div>
              )}
            </Tabs.Panel>
            <Tabs.Panel
              value="history"
              className="max-h-[70vh] overflow-y-auto outline-none"
            >
              <HistoryTab
                clipId={clip.id}
                segments={segments}
                jobId={clip.transcription?.transcription_job_id}
                onChanged={onChanged}
              />
            </Tabs.Panel>
          </Tabs.Root>
        </div>
      </section>
    </>
  );
}

/** The page of one clip: the player and the transcript side by side, with the clip's progress. */
export function ClipPage() {
  const { t, error } = useI18n();
  const { id } = useParams();
  const location = useLocation();
  const navigate = useNavigate();
  const clipId = /^[1-9]\d*$/.test(id ?? "") ? Number(id) : 0;
  const clip = useLoaded(
    useCallback(
      () =>
        clipId > 0
          ? fetchClip(clipId)
          : Promise.resolve({ ok: false as const, code: "invalid_clip_id" }),
      [clipId],
    ),
  );
  useDocumentTitle(t("clip.back"));

  // The list the clip was opened from, to go to its neighbours without it.
  const back = backAddress(location.state);
  const openedFromHere = hasOrigin(location.state);
  const sequence = useSequence(openedFromHere ? back : null);
  const [stripOpen, toggleStrip] = useStripOpen();
  // The tile the arrows moved to stays the current one, even if the list names the
  // same clip twice (two paths of one video).
  const chosen = useRef<number | null>(null);
  const index = useMemo(() => {
    if (!sequence) return -1;
    const kept = chosen.current;
    if (kept !== null && sequence.items[kept]?.clipId === clipId) return kept;
    return sequence.items.findIndex((item) => item.clipId === clipId);
  }, [sequence, clipId]);
  const go = useCallback(
    (target: number) => {
      const item = sequence?.items[target];
      if (!item) return;
      chosen.current = target;
      autoplayClip = item.clipId;
      // Only a clip reached by the arrows is remembered: a clip opened from the list and
      // closed keeps the scroll position the list restores.
      rememberLastSeen(back, item.clipId);
      // The history keeps one entry for the fiche: Back returns to the list in one step.
      navigate(
        `/clips/${item.clipId}${item.at !== undefined ? `?t=${item.at}` : ""}`,
        { replace: true, state: location.state },
      );
    },
    [sequence, navigate, location.state, back],
  );
  useEffect(() => {
    if (index < 0) return;
    const onKey = (event: KeyboardEvent) => {
      const target = event.target as HTMLElement;
      if (target.closest("input, textarea, select, [contenteditable]")) return;
      if (!event.shiftKey || event.metaKey || event.ctrlKey || event.altKey)
        return;
      const key = event.key.toLowerCase();
      if (key === "n" || key === "p") {
        event.preventDefault();
        go(key === "n" ? index + 1 : index - 1);
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [go, index]);

  const shown = clip.state.status === "ready" ? clip.state.value : null;
  const stale = !!shown && shown.id !== clipId;
  const current = sequence && index >= 0 ? sequence.items[index] : null;
  const topBar = (
    <>
      <div className="flex min-h-14 flex-wrap items-center gap-x-3 gap-y-1 border-b border-line px-4 py-2 text-sm text-mute md:px-7">
        <Link
          to={back}
          // Coming from the list or the search, "back" is the history's back: the
          // entries stay [list, clip] and the native button behaves the same.
          onClick={(event) => {
            if (!openedFromHere || event.metaKey || event.ctrlKey) return;
            event.preventDefault();
            navigate(-1);
          }}
          className="inline-flex min-h-11 items-center gap-1 hover:text-ink md:min-h-8"
        >
          <span aria-hidden="true">‹</span>
          {t(back.startsWith("/search") ? "clip.backToSearch" : "clip.back")}
        </Link>
        {shown && shown.status !== "ready" && (
          <ClipStatePill status={shown.status} />
        )}
        {sequence && index >= 0 && sequence.items.length > 1 && (
          <ClipPager
            sequence={sequence}
            index={index}
            open={stripOpen}
            onToggle={toggleStrip}
            onGo={go}
          />
        )}
      </div>
      {sequence && index >= 0 && sequence.items.length > 1 && stripOpen && (
        <ClipStrip sequence={sequence} index={index} onGo={go} />
      )}
      {current && sequence && (
        <p className="sr-only" aria-live="polite">
          {t("clip.nav.announce", {
            index: index + 1,
            total: sequence.items.length,
            name: current.title,
          })}
        </p>
      )}
    </>
  );

  if (clip.state.status === "loading") {
    return (
      <>
        {topBar}
        <p className="p-6 text-mute" role="status">
          {t("clip.loading")}
        </p>
      </>
    );
  }
  if (clip.state.status === "failed") {
    return (
      <>
        {topBar}
        <section className="flex flex-col gap-4 px-4 py-6 md:px-7">
          <h1 className="text-[22px] font-semibold">{t("clip.back")}</h1>
          <ErrorAlert>{error(clip.state.code)}</ErrorAlert>
          <div>
            <Link to="/library" className="text-accent-ink underline">
              {t("clip.back")}
            </Link>
          </div>
        </section>
      </>
    );
  }
  return (
    <>
      {topBar}
      {/* While the next clip loads, the one still shown is dimmed and out of reach: the strip
          already names the new one, and the old fiche must not pass for it. */}
      <div
        aria-busy={stale || undefined}
        inert={stale}
        className={`transition-opacity motion-reduce:transition-none ${stale ? "opacity-40" : ""}`}
      >
        <ClipView
          key={clip.state.value.id}
          clip={clip.state.value}
          onChanged={clip.reload}
          next={sequence && index >= 0 ? sequence.items[index + 1] : undefined}
          onNext={() => go(index + 1)}
        />
      </div>
    </>
  );
}
