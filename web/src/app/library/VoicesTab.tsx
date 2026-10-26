import { useI18n } from "../i18n";
import type { ClipVoice } from "./libraryApi";
import { SPEAKER_LOOKS, speakerState } from "./speakerState";

/** The voices of the clip: who speaks, for how long, and how settled their identity is. */
export function VoicesTab({ voice }: { voice: ClipVoice | null }) {
  const { t } = useI18n();
  if (!voice || voice.speakers.length === 0) {
    return <p className="p-4 text-mute">{t("clip.voices.empty")}</p>;
  }
  return (
    <ul
      aria-label={t("clip.voices.listLabel")}
      className="m-0 flex list-none flex-col gap-2 p-3"
    >
      {voice.speakers.map((speaker) => {
        const state =
          speakerState({
            local_speaker_id: speaker.id,
            local_speaker_label: speaker.label,
            local_speaker_decision_level: speaker.decision_level,
            local_speaker_corrected: false,
          }) ?? "unknown";
        const look = SPEAKER_LOOKS[state];
        return (
          <li
            key={speaker.id}
            className="flex flex-col gap-1 rounded-lg border border-line p-3"
          >
            <div className="flex flex-wrap items-center gap-2">
              <span className="font-medium">
                {speaker.person?.display_name ?? speaker.label}
              </span>
              <span
                className={`inline-flex items-center gap-1 rounded-full px-2 py-px text-[11px] font-medium ${look.className}`}
              >
                <look.icon className="size-3" />
                {t(look.label)}
              </span>
            </div>
            <p className="m-0 text-xs text-mute">
              {`${speaker.label} · `}
              {t("clip.voices.speaking", {
                seconds: Math.round(speaker.speaking_seconds),
                turns: speaker.turn_count,
              })}
              {!speaker.person && ` · ${t("clip.voices.unidentified")}`}
            </p>
          </li>
        );
      })}
    </ul>
  );
}
