import { Link } from "react-router";
import { useI18n } from "../i18n";
import { ExcerptPlayer } from "./ExcerptPlayer";
import { LevelBadge } from "./LevelBadge";
import type { Person, PersonVoice } from "./peopleApi";

/**
 * The voices confirmed for a Person that the caller can see, each with its excerpt to listen
 * to. Separating voices is an action of the record header, not of this list.
 */
export function PersonVoices({
  person,
  voices,
}: {
  person: Person;
  voices: PersonVoice[];
}) {
  const { t } = useI18n();
  return (
    <div
      role="group"
      aria-label={t("people.voicesPanel.title", { name: person.display_name })}
      className="flex flex-col gap-2"
    >
      <h3 className="text-base font-semibold">
        {t("people.voicesPanel.identifications")}
      </h3>
      {voices.length === 0 && (
        <p className="text-mute">{t("people.voicesPanel.empty")}</p>
      )}
      {voices.length > 0 && (
        <ul className="m-0 flex list-none flex-col gap-2 p-0">
          {voices.map((voice) => (
            <li
              key={voice.local_speaker_id}
              className="flex flex-wrap items-center gap-3 rounded-lg border border-line bg-surface p-2.5"
            >
              <ExcerptPlayer
                localSpeakerId={voice.local_speaker_id}
                label={voice.label}
                clip={voice.clip_name}
              />
              <div className="min-w-0 flex-1">
                <span className="flex flex-wrap items-center gap-2">
                  <b>{voice.label}</b>
                  <LevelBadge level="human" />
                </span>
                <span className="block truncate text-mute">
                  <Link
                    to={`/clips/${voice.clip_id}`}
                    aria-label={t("people.excerpt.openClip", {
                      clip: voice.clip_name,
                    })}
                    className="text-accent underline"
                  >
                    {voice.clip_name}
                  </Link>{" "}
                  ·{" "}
                  {t("people.voicesPanel.speech", {
                    seconds: voice.speaking_seconds.toFixed(1),
                  })}
                </span>
              </div>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
