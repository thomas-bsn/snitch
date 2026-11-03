import { useI18n } from "../i18n";
import { VoiceDecision } from "../voices/VoiceDecision";
import type { ClipVoice, TranscriptSegment } from "./libraryApi";
import { SPEAKER_LOOKS, speakerState } from "./speakerState";
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
  return (
    <div className="flex flex-col gap-4 p-3">
      <section
        aria-label={t("clip.voices.waveformLabel")}
        className="flex flex-col gap-2"
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
        className="m-0 flex list-none flex-col gap-2 p-0"
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
          return (
            <li
              key={speaker.id}
              className="flex flex-col gap-2 rounded-lg border border-line p-3"
            >
              <div className="flex flex-wrap items-center gap-2">
                <span
                  aria-hidden="true"
                  className="size-2.5 shrink-0 rounded-full"
                  style={{ backgroundColor: voiceColor(rank) }}
                />
                <span className="font-medium">
                  {speaker.person?.display_name ?? speaker.label}
                </span>
                {speaker.disposition ? (
                  <span className="inline-flex items-center rounded-full bg-side px-2 py-px text-[11px] font-medium text-ink">
                    {t(
                      speaker.disposition === "random"
                        ? "voiceDecision.status.random"
                        : "voiceDecision.status.ignored",
                    )}
                  </span>
                ) : (
                  look && (
                    <span
                      className={`inline-flex items-center gap-1 rounded-full px-2 py-px text-[11px] font-medium ${look.className}`}
                    >
                      <look.icon className="size-3" />
                      {t(look.label)}
                    </span>
                  )
                )}
              </div>
              <p className="m-0 text-xs text-mute">
                {`${speaker.label} · `}
                {t("clip.voices.speaking", {
                  seconds: Math.round(speaker.speaking_seconds),
                  turns: speaker.turn_count,
                })}
                {!speaker.person && ` · ${t("clip.voices.unidentified")}`}
              </p>
              <VoiceDecision
                speaker={speaker}
                clipName={clipName}
                onChanged={onChanged}
              />
            </li>
          );
        })}
      </ul>
    </div>
  );
}
