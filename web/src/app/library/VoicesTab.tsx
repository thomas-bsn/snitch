import { TriangleAlert } from "lucide-react";
import { useI18n } from "../i18n";
import { ExcerptPlayer } from "../people/ExcerptPlayer";
import { DISPOSITION, VoiceDecision } from "../voices/VoiceDecision";
import type { ClipVoice, TranscriptSegment } from "./libraryApi";
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
}: {
  voice: ClipVoice | null;
  segments: TranscriptSegment[];
  peaks?: number[];
  duration?: number | null;
  playhead: number;
  onSeek: (seconds: number) => void;
  clipName: string;
  onChanged: () => void;
}) {
  const { t } = useI18n();
  if (!voice || voice.speakers.length === 0) {
    return <p className="p-4 text-mute">{t("clip.voices.empty")}</p>;
  }
  const todo = voice.speakers.filter(voiceToDecide).length;
  return (
    <div className="flex flex-col">
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
