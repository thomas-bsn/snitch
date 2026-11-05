import {
  ArrowLeft,
  ArrowRight,
  Check,
  CheckCircle2,
  ExternalLink,
  Pause,
  Play,
  SkipForward,
} from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Link, useNavigate } from "react-router";
import { useLoaded } from "../api/loaded";
import { ErrorAlert } from "../components/Alert";
import { buttonClass } from "../components/buttons";
import { VolumeBar } from "../components/VolumeBar";
import { getVolume, subscribeVolume } from "../components/volume";
import { useI18n } from "../i18n";
import { SegmentActions } from "../library/SegmentActions";
import { clipTitle, fetchClip, fetchClipVoice } from "../library/libraryApi";
import type { LocalSpeaker, TranscriptSegment } from "../library/libraryApi";
import { VoiceDecision } from "../voices/VoiceDecision";
import { useFailureText } from "./failureText";
import { claimPlayback } from "./ExcerptPlayer";
import { HandledNotice } from "./HandledNotice";
import {
  PeakBars,
  ProcessWaveform,
  clock,
  processColor,
} from "./ProcessWaveform";
import { LevelBadge } from "./LevelBadge";
import {
  publishToProcessCount,
  toProcessCount,
} from "../shell/useToProcessCount";
import { fetchToProcess } from "./peopleApi";

const EXCERPTS = 4;
const LIST_SIZE = 100;

/** A voice still waits for a human when it is unknown, ambiguous or only suggested. */
const isPending = (voice: LocalSpeaker) =>
  !voice.disposition &&
  (voice.decision_level === "unknown" ||
    voice.decision_level === "ambiguous" ||
    voice.decision_level === "suggestion");

/** The voices after the one at `index`, then those before it: where to go next. */
const following = (speakers: LocalSpeaker[], index: number) => [
  ...speakers.slice(index + 1),
  ...speakers.slice(0, index),
];

const moment = (seconds: number) =>
  `${Math.floor(seconds / 60)}:${String(Math.floor(seconds % 60)).padStart(2, "0")}`;

/**
 * The processing screen of one clip: the waveform of the whole conversation with the voice
 * in progress highlighted, the queue of voices, and the card "Who is speaking here?" whose
 * decisions are those of the clip sheet (VoiceDecision). Settling a voice moves on to the
 * next one; at the end, the next clip of the "To process" list is one button away.
 * `focusId` opens the screen on one voice (the address of a case).
 */
export function ProcessScreen({
  clipId,
  focusId,
}: {
  clipId: number;
  focusId?: number;
}) {
  const { t } = useI18n();
  const failureText = useFailureText();
  const navigate = useNavigate();

  const clipLoad = useLoaded(useCallback(() => fetchClip(clipId), [clipId]));
  const clip = clipLoad.state;
  const voiceLoad = useLoaded(
    useCallback(() => fetchClipVoice(clipId), [clipId]),
  );
  const voice =
    voiceLoad.state.status === "ready" ? voiceLoad.state.value : undefined;
  const clipData = clip.status === "ready" ? clip.value : undefined;
  const name = clipData ? clipTitle(clipData) : "";

  const speakers = useMemo(() => voice?.speakers ?? [], [voice]);
  const pending = speakers.filter(isPending);
  const [selected, setSelected] = useState<number | undefined>(focusId);
  const current =
    speakers.find((speaker) => speaker.id === selected) ?? pending[0];
  // A direct link to a voice already settled says so, once, on arrival.
  const [arrival, setArrival] = useState<LocalSpeaker>();
  const arrived = useRef(false);
  useEffect(() => {
    if (arrived.current || !voice) return;
    arrived.current = true;
    const target = speakers.find((speaker) => speaker.id === focusId);
    if (target && !isPending(target)) setArrival(target);
  }, [voice, speakers, focusId]);
  // The voices that waited when the screen opened, for the progress "0 / 4 voices".
  const [initial, setInitial] = useState<Set<number>>(new Set());
  useEffect(() => {
    if (voice)
      setInitial((known) =>
        known.size > 0 ? known : new Set(pending.map((p) => p.id)),
      );
  }, [voice, pending]);
  // The voice just decided, until the voices have been read again.
  const decided = useRef<number | undefined>(undefined);

  // After a decision the voices are read again; the next voice to process takes the place.
  useEffect(() => {
    if (decided.current === undefined || !voice) return;
    const index = speakers.findIndex(
      (speaker) => speaker.id === decided.current,
    );
    decided.current = undefined;
    const stillPending = speakers[index] && isPending(speakers[index]);
    if (stillPending) return;
    setSelected(following(speakers, index).find(isPending)?.id);
  }, [voice, speakers]);

  // The other clips with something to process, for "Next video"; the tab counter follows.
  const [next, setNext] = useState<number | null | undefined>();
  const done = !!voice && pending.length === 0;
  const refreshList = useCallback(() => {
    void fetchToProcess("all", 1, LIST_SIZE).then((result) => {
      if (!result.ok) return;
      publishToProcessCount(toProcessCount(result.value.totals));
      setNext(
        result.value.items.find((item) => item.clip_id !== clipId)?.clip_id ??
          null,
      );
    });
  }, [clipId]);
  useEffect(() => {
    refreshList();
  }, [refreshList, voice]);

  function changed() {
    decided.current = current?.id;
    voiceLoad.reload();
  }

  function skip() {
    if (!current) return;
    const other = following(speakers, speakers.indexOf(current)).find(
      isPending,
    );
    if (other) setSelected(other.id);
  }

  // One media element plays the excerpts and the clicks in the waveform.
  const media = useRef<HTMLAudioElement>(null);
  const stopAt = useRef(Infinity);
  useEffect(() => {
    const apply = () => {
      if (media.current) media.current.volume = getVolume();
    };
    apply();
    return subscribeVolume(apply);
  }, [clipData]);
  const [playhead, setPlayhead] = useState(0);
  const [playing, setPlaying] = useState<string>();
  function playFrom(key: string, start: number, end: number) {
    const element = media.current;
    if (!element) return;
    if (playing === key) {
      element.pause();
      return;
    }
    claimPlayback(() => element.pause());
    element.currentTime = start;
    stopAt.current = end;
    setPlayhead(start);
    setPlaying(key);
    void element.play().catch(() => setPlaying(undefined));
  }

  const segments: TranscriptSegment[] = useMemo(
    () => clipData?.transcription?.segments ?? [],
    [clipData],
  );
  const waveform = clipData?.waveforms?.find(
    (entry) =>
      entry.audio_stream_index === clipData.transcription?.audio_stream_index,
  );
  const excerpts = current
    ? segments
        .filter(
          (segment) =>
            segment.local_speaker_id === current.id && segment.text.trim(),
        )
        .slice(0, EXCERPTS)
    : [];

  if (clip.status === "failed" || voiceLoad.state.status === "failed") {
    const code =
      clip.status === "failed"
        ? clip.code
        : voiceLoad.state.status === "failed"
          ? voiceLoad.state.code
          : "";
    return <ErrorAlert>{failureText(code)}</ErrorAlert>;
  }
  if (!clipData || !voice) {
    return <p className="text-mute">{t("people.loading")}</p>;
  }

  const rank = current ? speakers.indexOf(current) : -1;
  // The end of the last turn bounds the drawing when the recorded duration is shorter.
  const total = Math.max(
    clipData.duration_seconds ?? 0,
    ...segments.map((segment) => segment.end),
  );
  const colors = new Map(
    speakers
      .filter((speaker) => speaker.disposition !== "ignored")
      .map(
        (speaker) =>
          [speaker.id, processColor(speakers.indexOf(speaker))] as const,
      ),
  );
  const spoken = (speaker: LocalSpeaker) =>
    segments
      .filter((segment) => segment.local_speaker_id === speaker.id)
      .reduce((sum, segment) => sum + (segment.end - segment.start), 0);
  const turns = current
    ? segments.filter((segment) => segment.local_speaker_id === current.id)
    : [];
  const resolved = [...initial].filter(
    (id) => !pending.some((speaker) => speaker.id === id),
  ).length;
  const steps = Math.max(initial.size, resolved + pending.length);

  return (
    <section
      aria-label={t("people.process.label")}
      className="flex flex-col gap-5"
    >
      <audio
        ref={media}
        preload="none"
        src={`/api/clips/${clipId}/media`}
        onPause={() => setPlaying(undefined)}
        onEnded={() => setPlaying(undefined)}
        onTimeUpdate={(event) => {
          const time = event.currentTarget.currentTime;
          setPlayhead(time);
          if (time >= stopAt.current) event.currentTarget.pause();
        }}
      />
      {arrival && <HandledNotice voice={arrival} />}
      <div className="flex flex-wrap items-center gap-3">
        <Link
          to="/people/verification"
          className="inline-flex min-h-11 items-center gap-1 text-mute hover:text-ink md:min-h-8"
        >
          <ArrowLeft className="size-4" aria-hidden="true" />
          {t("people.process.backToList")}
        </Link>
        <span aria-hidden="true" className="hidden h-4 w-px bg-line md:block" />
        <div className="min-w-0 flex-1">
          <h1 className="m-0 truncate text-lg font-semibold">
            {t("people.process.title", { clip: name })}
          </h1>
          {total > 0 && <p className="m-0 text-xs text-mute">{clock(total)}</p>}
        </div>
        <div className="flex items-center gap-2 text-sm">
          <span className="text-mute">
            {t("people.process.progress", { done: resolved, total: steps })}
          </span>
          <div aria-hidden="true" className="flex gap-1">
            {Array.from({ length: steps }, (_, index) => (
              <span
                key={index}
                className={`h-1.5 w-6 rounded-full ${index < resolved ? "bg-ok-fg" : "bg-line"}`}
              />
            ))}
          </div>
        </div>
        <Link
          to={`/clips/${clipId}`}
          className="inline-flex min-h-11 items-center gap-1 text-sm text-accent-ink hover:underline md:min-h-8"
        >
          {t("people.process.openClip")}
          <ExternalLink className="size-3.5" aria-hidden="true" />
        </Link>
      </div>

      <VolumeBar />

      <ProcessWaveform
        speakers={speakers}
        segments={segments}
        peaks={waveform?.peaks}
        duration={total}
        playhead={playhead}
        playing={playing === "wave"}
        onToggle={() => playFrom("wave", playhead, Infinity)}
        onSeek={(seconds) => playFrom("wave", seconds, Infinity)}
        activeId={current?.id}
      />

      <div className="grid gap-5 lg:grid-cols-[260px_minmax(0,1fr)]">
        <ul
          aria-label={t("people.process.queueLabel")}
          className="m-0 flex list-none flex-col gap-1 p-0"
        >
          {speakers.map((speaker, index) => {
            const waiting = isPending(speaker);
            const status = speaker.disposition
              ? t(
                  speaker.disposition === "random"
                    ? "voiceDecision.status.random"
                    : "voiceDecision.status.ignored",
                )
              : waiting
                ? t("people.process.waiting")
                : t("people.process.settled");
            return (
              <li key={speaker.id}>
                <button
                  type="button"
                  aria-current={speaker.id === current?.id ? "true" : undefined}
                  onClick={() => setSelected(speaker.id)}
                  className={`flex min-h-11 w-full items-center gap-3 rounded-xl px-3 py-2.5 text-left ${
                    speaker.id === current?.id
                      ? "bg-accent-soft"
                      : "hover:bg-side"
                  }`}
                >
                  <span
                    aria-hidden="true"
                    className="size-2.5 shrink-0 rounded-full"
                    style={{
                      backgroundColor:
                        speaker.disposition === "ignored"
                          ? "var(--line-strong)"
                          : processColor(index),
                    }}
                  />
                  <span className="min-w-0 flex-1">
                    <span className="block text-sm font-medium">
                      {speaker.person?.display_name ?? (
                        <span className="lowercase">{speaker.label}</span>
                      )}
                    </span>
                    <span className="block text-xs text-mute">
                      {t("people.process.speaking", {
                        time: clock(spoken(speaker)),
                      })}
                    </span>
                  </span>
                  <span className="sr-only">{status}</span>
                  {waiting ? (
                    <span
                      aria-hidden="true"
                      className="size-2 shrink-0 rounded-full bg-warn-fg"
                    />
                  ) : (
                    <Check
                      className="size-4 shrink-0 text-ok-fg"
                      aria-hidden="true"
                    />
                  )}
                </button>
              </li>
            );
          })}
        </ul>

        <div className="flex min-w-0 flex-col gap-5">
          {done && (
            <div
              role="status"
              className="flex flex-col items-center gap-3 rounded-2xl border border-line p-10 text-center"
            >
              <CheckCircle2 className="size-10 text-ok-fg" aria-hidden="true" />
              <p className="m-0 text-lg font-semibold">
                {t("people.process.done")}
              </p>
              <button
                type="button"
                disabled={!next}
                onClick={() =>
                  next && navigate(`/people/verification?clip=${next}`)
                }
                className={buttonClass("primary")}
              >
                {t("people.process.nextVideo")}
                <ArrowRight className="size-4" aria-hidden="true" />
              </button>
              {next === null && (
                <p className="m-0 text-mute">{t("people.process.noNext")}</p>
              )}
            </div>
          )}
          {current && (
            <section
              aria-label={t("people.process.cardLabel")}
              className="flex flex-col gap-5 rounded-2xl border border-line p-5"
              style={{ borderTop: `4px solid ${processColor(rank)}` }}
            >
              <div className="flex flex-wrap items-baseline gap-x-3 gap-y-1">
                <p className="m-0 text-xl font-semibold">
                  {t("people.process.cardLabel")}
                </p>
                <h2 className="m-0 font-mono text-sm font-normal text-mute lowercase">
                  {current.label}
                </h2>
                <span className="text-sm text-mute">
                  ·{" "}
                  {t("people.process.turns", {
                    count: turns.length,
                    time: clock(spoken(current)),
                  })}
                </span>
                <LevelBadge level={current.decision_level} />
                <span className="text-sm text-mute">
                  {t("people.process.position", {
                    n: Math.max(1, pending.indexOf(current) + 1),
                    total: pending.length,
                  })}
                </span>
              </div>
              <ul
                aria-label={t("people.process.excerptsLabel")}
                className="m-0 flex list-none flex-col gap-2 p-0"
              >
                {excerpts.map((segment) => {
                  const key = `s${segment.start}`;
                  const on = playing === key;
                  return (
                    <li
                      key={key}
                      className={`group relative flex items-center gap-3 rounded-xl border px-3 py-2 pr-14 ${
                        on ? "border-accent bg-accent-soft" : "border-line"
                      }`}
                    >
                      <button
                        type="button"
                        aria-pressed={on}
                        aria-label={t("people.process.playFrom", {
                          time: moment(segment.start),
                        })}
                        onClick={() =>
                          playFrom(key, segment.start, segment.end)
                        }
                        className="flex size-11 shrink-0 items-center justify-center rounded-full border border-line-strong hover:bg-side md:size-9"
                      >
                        {on ? (
                          <Pause className="size-4" aria-hidden="true" />
                        ) : (
                          <Play
                            className="size-4 translate-x-px"
                            aria-hidden="true"
                          />
                        )}
                      </button>
                      <div className="hidden w-28 shrink-0 sm:block">
                        <PeakBars
                          peaks={waveform?.peaks}
                          segments={segments}
                          colors={colors}
                          total={total}
                          from={segment.start}
                          to={segment.end}
                          focusId={current.id}
                          bars={40}
                          height={28}
                        />
                      </div>
                      <div className="min-w-0 flex-1">
                        <p className="m-0 text-sm">« {segment.text} »</p>
                        <SegmentActions
                          clipId={clipId}
                          jobId={clipData.transcription?.transcription_job_id}
                          segment={segment}
                          speakers={[]}
                          onChanged={clipLoad.reload}
                        />
                      </div>
                      <span className="text-xs text-mute tabular-nums">
                        {moment(segment.start)}
                      </span>
                    </li>
                  );
                })}
              </ul>
              <VoiceDecision
                key={current.id}
                speaker={current}
                clipName={name}
                onChanged={changed}
                shortcuts
                showCandidates
                triage
                trailing={
                  <button
                    type="button"
                    disabled={
                      pending.filter((p) => p.id !== current.id).length === 0
                    }
                    onClick={skip}
                    className="inline-flex min-h-11 items-center gap-1 text-sm text-mute hover:text-ink disabled:opacity-60 md:min-h-8"
                  >
                    {t("people.process.skip")}
                    <SkipForward className="size-4" aria-hidden="true" />
                  </button>
                }
              />
              <p className="m-0 text-xs text-mute">
                {t("people.process.shortcuts")}
              </p>
            </section>
          )}
        </div>
      </div>
    </section>
  );
}
