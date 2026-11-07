import { TriangleAlert } from "lucide-react";
import { useI18n } from "../i18n";
import { ExcerptPlayer } from "../people/ExcerptPlayer";
import { DISPOSITION, VoiceDecision } from "../voices/VoiceDecision";
import type {
  ClipVoice,
  SpeakerCoverage,
  TranscriptSegment,
} from "./libraryApi";
import { formatTimecode } from "./clipFormat";
import { SPEAKER_LOOKS, speakerState, voiceToDecide } from "./speakerState";
import { VoiceWaveforms, voiceColor } from "./VoiceWaveforms";

/**
 * The voices of the clip: the waveform with one track per voice, then each voice with its colour,
 * how settled its identity is, and the decisions on it.
 */
export function VoicesTab({
  voice,
  segments,
  peaks,
  duration,
  playhead,
  onSeek,
  clipName,
  onChanged,
  coverage = [],
}: {
  voice: ClipVoice | null;
  segments: TranscriptSegment[];
  peaks?: number[];
  duration?: number | null;
  playhead: number;
  onSeek: (seconds: number) => void;
  clipName: string;
  onChanged: () => void;
  coverage?: SpeakerCoverage[];
}) {
  const { t } = useI18n();
  const tracks = voice?.tracks ?? [];
  if (!voice || (voice.speakers.length === 0 && tracks.length === 0)) {
    return <p className="p-4 text-mute">{t("clip.voices.empty")}</p>;
  }
  const todo = voice.speakers.filter(voiceToDecide).length;
  // A voice only shows its track when the clip has several: provenance, never a merge of voices.
  const showProvenance = tracks.length > 1;
  return (
    <div className="flex flex-col">
      {tracks.length > 0 && (
        <ul
          aria-label={t("clip.voices.trackStatesLabel")}
          className="m-0 flex list-none flex-col gap-1 border-b border-line p-0 px-4 py-3 text-sm"
        >
          {tracks.map((track) => (
            <li
              key={track.audio_stream_index}
              data-testid="voice-track"
              data-track={track.audio_stream_index}
              data-status={track.status}
              className="flex flex-wrap items-center gap-x-2"
            >
              <span className="font-medium">
                {t("clip.voices.trackLine", {
                  index: track.audio_stream_index,
                })}
              </span>
              {track.microphone && (
                <span className="rounded-full bg-side px-2 py-0.5 text-xs text-mute">
                  {t("clip.voices.microphoneTrack")}
                </span>
              )}
              <span
                className={
                  track.status === "failed" || track.cause
                    ? "text-warn-fg"
                    : "text-mute"
                }
              >
                {t(`clip.voices.trackStatus.${track.status}`)}
              </span>
              {track.cause && (
                <span className="text-xs text-mute">
                  {trackCauseLabel(t, track.cause)}
                </span>
              )}
            </li>
          ))}
        </ul>
      )}
      {todo > 0 && (
        <p className="m-0 flex items-center gap-2 border-b border-line bg-warn-bg px-4 py-2 text-sm text-warn-fg">
          <TriangleAlert className="size-4 shrink-0" aria-hidden="true" />
          {t("clip.voices.todo", { count: todo })}
        </p>
      )}
      <section
        aria-label={t("clip.voices.waveformLabel")}
        className="border-b border-line px-4 py-3"
      >
        <VoiceWaveforms
          speakers={voice.speakers}
          segments={segments}
          peaks={peaks}
          duration={duration}
          playhead={playhead}
          onSeek={onSeek}
        />
      </section>
      <ul
        aria-label={t("clip.voices.listLabel")}
        className="m-0 flex list-none flex-col p-0"
      >
        {voice.speakers.map((speaker, rank) => {
          const state = speaker.disposition
            ? null
            : (speakerState({
                local_speaker_id: speaker.id,
                local_speaker_label: speaker.label,
                local_speaker_decision_level: speaker.decision_level,
                local_speaker_corrected: false,
              }) ?? "unknown");
          const look = state ? SPEAKER_LOOKS[state] : null;
          const name = speaker.person?.display_name;
          const chip = speaker.disposition
            ? speaker.disposition === DISPOSITION.random
              ? {
                  text: t("voiceDecision.status.random"),
                  className: "bg-warn-bg text-warn-fg",
                }
              : null
            : name
              ? state === "suggestion"
                ? {
                    text: `${name} ?`,
                    className:
                      "border border-dashed border-accent text-accent-ink",
                  }
                : { text: name, className: "bg-accent-soft text-accent-ink" }
              : {
                  text: speaker.label,
                  className: "bg-side text-mute lowercase",
                };
          const covered = coverage.find(
            (c) => c.local_speaker_id === speaker.id,
          );
          const muted = !!speaker.disposition;
          return (
            <li
              key={speaker.id}
              className={`flex flex-col gap-2 border-b border-line px-4 py-3 ${muted ? "bg-side/60" : ""}`}
            >
              <div className="flex flex-wrap items-center gap-x-2 gap-y-1">
                <span
                  aria-hidden="true"
                  className="size-2.5 shrink-0 rounded-full"
                  style={{ backgroundColor: voiceColor(rank) }}
                />
                <span className="font-mono text-xs text-mute lowercase">
                  {speaker.label}
                </span>
                {showProvenance && (
                  <span
                    data-testid="voice-provenance"
                    data-track={speaker.audio_stream_index}
                    className="inline-flex w-fit rounded-full border border-line px-2 py-0.5 text-xs text-mute"
                  >
                    {t("clip.voices.fromTrack", {
                      index: speaker.audio_stream_index,
                    })}
                  </span>
                )}
                {chip ? (
                  <span
                    className={`inline-flex w-fit rounded-full px-2 py-0.5 text-xs font-medium ${chip.className}`}
                  >
                    {chip.text}
                  </span>
                ) : (
                  <span className="text-xs text-mute italic">
                    {t("voiceDecision.status.ignored")}
                  </span>
                )}
                <span className="text-xs text-mute">
                  {"· "}
                  {look && `${t(look.label)} · `}
                  {t("clip.voices.speaking", {
                    turns: speaker.turn_count,
                    duration: formatTimecode(speaker.speaking_seconds),
                  })}
                </span>
                {covered && !speaker.disposition && (
                  <span
                    data-testid="voice-coverage"
                    data-status={covered.coverage_status}
                    className={`inline-flex w-fit rounded-full px-2 py-0.5 text-xs ${covered.coverage_status === "words" ? "bg-side text-mute" : "bg-warn-bg text-warn-fg"}`}
                  >
                    {t(`clip.voices.coverage.${covered.coverage_status}`)}
                  </span>
                )}
                <span className="ml-auto">
                  <ExcerptPlayer
                    link
                    localSpeakerId={speaker.id}
                    label={speaker.label}
                    clip={clipName}
                  />
                </span>
              </div>
              <VoiceDecision
                speaker={speaker}
                clipName={clipName}
                onChanged={onChanged}
                excerpt={false}
              />
            </li>
          );
        })}
      </ul>
    </div>
  );
}

const KNOWN_CAUSES = [
  "voice_worker_unavailable",
  "microphone_direct_attribution",
  "no_speakers",
  "cancelled_by_admin",
  "fingerprint_missing",
] as const;

/** A failed step carries a redacted code of its own: show the generic message, not the code. */
function trackCauseLabel(
  t: ReturnType<typeof useI18n>["t"],
  cause: string,
): string {
  return (KNOWN_CAUSES as readonly string[]).includes(cause)
    ? t(`clip.voices.trackCause.${cause as (typeof KNOWN_CAUSES)[number]}`)
    : t("clip.voices.trackCause.step_failed");
}
