import { Tabs } from "@base-ui/react/tabs";
import { useCallback, useMemo, useRef, useState } from "react";
import { Link, useLocation, useParams, useSearchParams } from "react-router";
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
import { AnalyzeClip } from "./AnalyzeClip";
import { ReanalyzeClip } from "./ReanalyzeClip";
import { buttonClass } from "../components/buttons";

const tabClass =
  "-mb-px inline-flex h-11 shrink-0 cursor-pointer items-center gap-2 border-b-2 border-transparent px-3 font-medium whitespace-nowrap text-mute outline-none hover:text-ink data-[active]:border-accent data-[active]:text-ink md:h-10";

type TabName = "transcript" | "voices" | "history";

function ClipView({ clip, onChanged }: { clip: Clip; onChanged: () => void }) {
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
  const voice = useLoaded(
    useCallback(() => fetchClipVoice(clip.id), [clip.id]),
  );
  useDocumentTitle(clipTitle(clip));

  const back = (location.state as { from?: string } | null)?.from ?? "/library";
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
      <div className="flex flex-wrap items-center gap-x-3 gap-y-1 border-b border-line px-4 py-3 text-sm text-mute md:px-7">
        <Link
          to={back}
          className="inline-flex min-h-11 items-center gap-1 hover:text-ink md:min-h-8"
        >
          <span aria-hidden="true">‹</span>
          {t(back.startsWith("/search") ? "clip.backToSearch" : "clip.back")}
        </Link>
        {clip.status !== "ready" && <ClipStatePill status={clip.status} />}
      </div>
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
                legacy={legacy}
                summary={<ProgressBanner clip={clip} voice={voiceData} />}
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
            />
            <MetadataDisclosure clip={clip} />
            <ProgressBanner clip={clip} voice={voiceData} assistiveOnly />
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

  if (clip.state.status === "loading") {
    return (
      <p className="p-6 text-mute" role="status">
        {t("clip.loading")}
      </p>
    );
  }
  if (clip.state.status === "failed") {
    return (
      <section className="flex flex-col gap-4 px-4 py-6 md:px-7">
        <h1 className="text-[22px] font-semibold">{t("clip.back")}</h1>
        <ErrorAlert>{error(clip.state.code)}</ErrorAlert>
        <div>
          <Link to="/library" className="text-accent-ink underline">
            {t("clip.back")}
          </Link>
        </div>
      </section>
    );
  }
  return (
    <ClipView
      key={clip.state.value.id}
      clip={clip.state.value}
      onChanged={clip.reload}
    />
  );
}
