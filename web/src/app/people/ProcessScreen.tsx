import { Pause, Play } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Link, useNavigate } from "react-router";
import { useLoaded } from "../api/loaded";
import { ErrorAlert } from "../components/Alert";
import { buttonClass } from "../components/buttons";
import { useI18n } from "../i18n";
import { clipTitle, fetchClip, fetchClipVoice } from "../library/libraryApi";
import type { LocalSpeaker, TranscriptSegment } from "../library/libraryApi";
import { VoiceWaveforms, voiceColor } from "../library/VoiceWaveforms";
import { VoiceDecision } from "../voices/VoiceDecision";
import { useFailureText } from "./failureText";
import { claimPlayback } from "./ExcerptPlayer";
import { HandledNotice } from "./HandledNotice";
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

  const clip = useLoaded(useCallback(() => fetchClip(clipId), [clipId])).state;
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
  const suggested =
    current?.person && current.attribution?.validation_state !== "validated";
  const confidence = current?.attribution?.confidence;

  return (
    <section
      aria-label={t("people.process.label")}
      className="flex flex-col gap-4"
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
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div className="flex min-w-0 flex-col">
          <h2 className="m-0 truncate text-lg font-semibold">
            {t("people.process.title", { clip: name })}
          </h2>
          <Link
            to="/people/verification"
            className="inline-flex min-h-11 items-center text-accent-ink underline md:min-h-8"
          >
            {t("people.process.backToList")}
          </Link>
        </div>
        <Link to={`/clips/${clipId}`} className={buttonClass("secondary")}>
          {t("people.process.openClip")}
        </Link>
      </div>

      <VoiceWaveforms
        speakers={speakers}
        segments={segments}
        peaks={waveform?.peaks}
        duration={clipData.duration_seconds}
        playhead={playhead}
        onSeek={(seconds) => playFrom("wave", seconds, Infinity)}
        activeId={current?.id}
      />

      <div className="grid gap-4 md:grid-cols-[14rem_minmax(0,1fr)]">
        <ul
          aria-label={t("people.process.queueLabel")}
          className="m-0 flex list-none flex-col gap-2 p-0"
        >
          {speakers.map((speaker, index) => {
            const waiting = isPending(speaker);
            return (
              <li key={speaker.id}>
                <button
                  type="button"
                  aria-current={speaker.id === current?.id ? "true" : undefined}
                  onClick={() => setSelected(speaker.id)}
                  className={`flex min-h-11 w-full items-center gap-2 rounded-lg border px-3 text-left ${
                    speaker.id === current?.id
                      ? "border-accent bg-accent-soft text-accent-ink"
                      : "border-line bg-surface hover:bg-side"
                  }`}
                >
                  <span
                    aria-hidden="true"
                    className="size-2.5 shrink-0 rounded-full"
                    style={{ backgroundColor: voiceColor(index) }}
                  />
                  <span className="min-w-0 flex-1 truncate">
                    {speaker.person?.display_name ?? speaker.label}
                  </span>
                  <span className="text-xs text-mute">
                    {speaker.disposition
                      ? t(
                          speaker.disposition === "random"
                            ? "voiceDecision.status.random"
                            : "voiceDecision.status.ignored",
                        )
                      : waiting
                        ? t("people.process.waiting")
                        : t("people.process.settled")}
                  </span>
                </button>
              </li>
            );
          })}
        </ul>

        <div className="flex min-w-0 flex-col gap-4">
          {done && (
            <div
              role="status"
              className="flex flex-col items-start gap-2 rounded-xl border border-line bg-surface p-4"
            >
              <p className="m-0 font-medium">{t("people.process.done")}</p>
              <button
                type="button"
                disabled={!next}
                onClick={() =>
                  next && navigate(`/people/verification?clip=${next}`)
                }
                className={buttonClass("primary")}
              >
                {t("people.process.nextVideo")}
              </button>
              {next === null && (
                <p className="m-0 text-mute">{t("people.process.noNext")}</p>
              )}
            </div>
          )}
          {current && (
            <section
              aria-label={t("people.process.cardLabel")}
              className="flex flex-col gap-3 rounded-xl border border-line bg-surface p-4"
            >
              <div className="flex flex-wrap items-center gap-2">
                <span
                  aria-hidden="true"
                  className="size-3 rounded-full"
                  style={{ backgroundColor: voiceColor(rank) }}
                />
                <h2 className="m-0 text-lg font-semibold">{current.label}</h2>
                <LevelBadge level={current.decision_level} />
                <span className="text-mute">
                  {t("people.process.position", {
                    n: Math.max(1, pending.indexOf(current) + 1),
                    total: pending.length,
                  })}
                </span>
              </div>
              <p className="m-0 font-medium">{t("people.process.cardLabel")}</p>
              {suggested && current.person && (
                <p className="m-0">
                  {t("people.process.suggestion", {
                    person: current.person.display_name,
                  })}
                  {typeof confidence === "number" &&
                    ` ${t("voiceDecision.proximity", {
                      percent: Math.round(confidence * 100),
                    })}`}
                </p>
              )}
              <ul
                aria-label={t("people.process.excerptsLabel")}
                className="m-0 flex list-none flex-col gap-1 p-0"
              >
                {excerpts.map((segment) => {
                  const key = `s${segment.start}`;
                  return (
                    <li
                      key={key}
                      className="flex items-center gap-2 rounded-lg border border-line px-2 py-1"
                    >
                      <button
                        type="button"
                        aria-pressed={playing === key}
                        aria-label={t("people.process.playFrom", {
                          time: moment(segment.start),
                        })}
                        onClick={() =>
                          playFrom(key, segment.start, segment.end)
                        }
                        className="flex size-11 shrink-0 items-center justify-center rounded-full border border-line bg-surface hover:bg-side md:size-9"
                      >
                        {playing === key ? (
                          <Pause className="size-4" aria-hidden="true" />
                        ) : (
                          <Play className="size-4" aria-hidden="true" />
                        )}
                      </button>
                      <span className="text-xs text-mute">
                        {moment(segment.start)}
                      </span>
                      <span className="min-w-0 flex-1">{segment.text}</span>
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
              />
              <div className="flex flex-wrap items-center justify-between gap-2">
                <span className="text-mute">
                  {t("people.process.shortcuts")}
                </span>
                <button
                  type="button"
                  disabled={
                    pending.filter((p) => p.id !== current.id).length === 0
                  }
                  onClick={skip}
                  className={buttonClass("secondary")}
                >
                  {t("people.process.skip")}
                </button>
              </div>
            </section>
          )}
        </div>
      </div>
    </section>
  );
}
