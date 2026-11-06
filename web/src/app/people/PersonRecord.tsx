import { ChevronLeft } from "lucide-react";
import { useCallback, useId } from "react";
import { Link } from "react-router";
import { useLoaded } from "../api/loaded";
import { ErrorAlert } from "../components/Alert";
import { StatusPill } from "../components/StatusPill";
import { useI18n } from "../i18n";
import { useFailureText } from "./failureText";
import { PersonActions } from "./PersonActions";
import { Avatar, referenceLevel, shortDate, speechShort } from "./PersonParts";
import { PersonVideos, PersonVoices } from "./PersonVoices";
import {
  fetchPerson,
  fetchPersonImpact,
  fetchPersonVoices,
  type Person,
  type PersonImpact,
  type PersonVoice,
} from "./peopleApi";

/** What a Person has, as the server counts it: voices and validated samples (the reference). */
export function Counts({ person }: { person: Person }) {
  const { t } = useI18n();
  return (
    <>
      {t("people.persons.voiceCount", { count: person.local_speaker_count })}
      {" · "}
      {person.validated_sample_count > 0
        ? t("people.persons.sampleCount", {
            count: person.validated_sample_count,
          })
        : t("people.persons.noReference")}
    </>
  );
}

type Loaded = { person: Person; impact: PersonImpact; voices: PersonVoice[] };

/**
 * The record of one Person: its real counts, its account, its actions in the header and its
 * confirmed voices.
 */
export function PersonRecord({
  id,
  version,
  onChanged,
  onMerged,
  onDeleted,
}: {
  id: string;
  version: number;
  onChanged: (message: string) => void;
  onMerged: (target: Person, message: string) => void;
  onDeleted: (message: string) => void;
}) {
  const { t, language } = useI18n();
  const failureText = useFailureText();
  const headingId = useId();
  const valid = /^[1-9]\d{0,14}$/.test(id);
  const load = useCallback(async () => {
    if (!valid) return { ok: false as const, code: "person_not_found" };
    const number = Number(id);
    const [person, impact, voices] = await Promise.all([
      fetchPerson(number),
      fetchPersonImpact(number),
      fetchPersonVoices(number),
    ]);
    if (!person.ok) return person;
    if (!impact.ok) return impact;
    if (!voices.ok) return voices;
    return {
      ok: true as const,
      value: {
        person: person.value,
        impact: impact.value,
        voices: voices.value,
      } satisfies Loaded,
    };
    // `version` reloads the record after a change made from the list or the record.
  }, [id, valid, version]);
  const { state, reload } = useLoaded(load);

  const back = (
    <Link
      to="/people"
      aria-label={t("people.persons.back")}
      className="inline-flex min-h-11 w-fit items-center gap-1 text-mute hover:text-ink md:min-h-0"
    >
      <ChevronLeft className="size-4" aria-hidden="true" />
      {t("people.persons.backShort")}
    </Link>
  );

  if (state.status === "loading") {
    return (
      <div className="flex flex-col gap-2">
        {back}
        <p className="text-mute">{t("people.loading")}</p>
      </div>
    );
  }
  if (state.status === "failed") {
    if (state.code !== "person_not_found") {
      return (
        <div className="flex flex-col gap-2">
          {back}
          <ErrorAlert>{failureText(state.code)}</ErrorAlert>
        </div>
      );
    }
    return (
      <div className="flex flex-col gap-2">
        {back}
        <h2 className="text-lg font-semibold">
          {t("people.persons.notFound")}
        </h2>
        <p className="text-mute">{t("people.persons.notFoundBody")}</p>
      </div>
    );
  }

  const { person, impact, voices } = state.value;
  const level = referenceLevel(person.validated_sample_count);
  const lastHeard = voices.reduce<string | undefined>(
    (latest, voice) =>
      latest === undefined || voice.recorded_at > latest
        ? voice.recorded_at
        : latest,
    undefined,
  );
  const account =
    person.user_id === null
      ? t("people.persons.noAccount")
      : person.account_username
        ? t("people.persons.linkedTo", { username: person.account_username })
        : t("people.persons.linkedNoName");
  const tiles: [string, string, boolean][] = [
    [t("people.persons.tiles.videos"), String(person.clip_count), false],
    [
      t("people.persons.tiles.speech"),
      speechShort(person.speaking_seconds),
      false,
    ],
    [
      t("people.persons.tiles.samples"),
      String(person.validated_sample_count),
      false,
    ],
    [
      t("people.persons.tiles.quality"),
      t(`people.reference.${level}`),
      level !== "good",
    ],
  ];
  return (
    <section aria-labelledby={headingId} className="flex flex-col gap-4">
      {back}
      <div className="flex flex-wrap items-center gap-3">
        <Avatar
          personId={person.user_id !== null ? person.id : undefined}
          name={person.display_name}
          size="record"
        />
        <div className="min-w-0 flex-1">
          <div className="flex flex-wrap items-center gap-2">
            <h2 id={headingId} className="m-0 text-lg font-semibold">
              {person.display_name}
            </h2>
            {person.voice_profile_deleted_at !== null && (
              <StatusPill tone="idle">
                {t("people.persons.profileDeleted")}
              </StatusPill>
            )}
          </div>
          <p className="m-0 text-sm text-mute">
            {account}
            {lastHeard !== undefined &&
              ` · ${t("people.persons.lastHeard", { date: shortDate(lastHeard, language) })}`}
          </p>
        </div>
        <PersonActions
          key={person.id}
          person={person}
          impact={impact}
          voices={voices}
          onChanged={onChanged}
          onMerged={onMerged}
          onDeleted={onDeleted}
          onRefused={reload}
        />
      </div>
      <div
        role="group"
        aria-label={t("people.persons.figuresLabel", {
          name: person.display_name,
        })}
      >
        <dl className="m-0 grid grid-cols-2 gap-3 sm:grid-cols-4">
          {tiles.map(([label, value, warn]) => (
            <div key={label} className="rounded-lg border border-line p-3">
              <dt className="text-xs text-mute">{label}</dt>
              <dd
                className={`m-0 mt-1 text-base font-semibold ${warn ? "text-warn-fg" : ""}`}
              >
                {value}
              </dd>
            </div>
          ))}
        </dl>
      </div>
      {level !== "good" && (
        <div
          role="note"
          className="rounded-lg bg-warn-bg px-3 py-2 text-sm text-warn-fg"
        >
          <b className="block">{t("people.persons.referenceWarningTitle")}</b>
          {t(
            level === "none"
              ? "people.persons.referenceWarningNone"
              : "people.persons.referenceWarningWeak",
            { name: person.display_name },
          )}
        </div>
      )}
      <PersonVideos person={person} voices={voices} />
      <PersonVoices person={person} voices={voices} />
    </section>
  );
}
