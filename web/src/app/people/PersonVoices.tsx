import { Link } from "react-router";
import { useI18n } from "../i18n";
import { ExcerptPlayer } from "./ExcerptPlayer";
import { LevelBadge } from "./LevelBadge";
import { clock, shortDate } from "./PersonParts";
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

/** The videos where the Person is heard: one line per clip, with listen button, date, time and state. */
export function PersonVideos({
  person,
  voices,
}: {
  person: Person;
  voices: PersonVoice[];
}) {
  const { t, language } = useI18n();
  const clips = new Map<
    number,
    { name: string; seconds: number; recordedAt: string; voice: PersonVoice }
  >();
  for (const voice of voices) {
    const clip = clips.get(voice.clip_id) ?? {
      name: voice.clip_name,
      seconds: 0,
      recordedAt: voice.recorded_at,
      voice,
    };
    clip.seconds += voice.speaking_seconds;
    clips.set(voice.clip_id, clip);
  }
  const title = t("people.persons.videosTitle", { name: person.display_name });
  return (
    <section role="group" aria-label={title} className="flex flex-col">
      <h3 className="m-0 mb-2 text-sm font-semibold">
        {t("people.persons.heardIn")}
      </h3>
      {clips.size === 0 && (
        <p className="text-mute">{t("people.persons.videosEmpty")}</p>
      )}
      {clips.size > 0 && (
        <ul className="m-0 flex list-none flex-col p-0">
          {[...clips].map(([clipId, clip]) => (
            <li
              key={clipId}
              className="flex flex-wrap items-center gap-3 border-b border-line py-2 last:border-b-0"
            >
              <ExcerptPlayer
                compact
                localSpeakerId={clip.voice.local_speaker_id}
                label={clip.voice.label}
                clip={clip.name}
              />
              <Link
                to={`/clips/${clipId}`}
                className="min-w-0 flex-1 truncate hover:underline"
              >
                {clip.name}
              </Link>
              <span className="text-xs text-mute">
                {shortDate(clip.recordedAt, language)} · {clock(clip.seconds)}
              </span>
              <span className="inline-flex items-center rounded-full bg-ok-bg px-2 py-0.5 text-xs font-medium text-ok-fg">
                {t("people.persons.heardState")}
              </span>
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}
