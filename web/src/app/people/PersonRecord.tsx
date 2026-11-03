import { ChevronLeft } from "lucide-react";
import { useCallback, useId } from "react";
import { Link } from "react-router";
import { useLoaded } from "../api/loaded";
import { ErrorAlert } from "../components/Alert";
import { StatusPill } from "../components/StatusPill";
import { useI18n } from "../i18n";
import { useFailureText } from "./failureText";
import { PersonActions } from "./PersonActions";
import {
  Avatar,
  ReferenceBadge,
  referenceLevel,
  speechText,
} from "./PersonParts";
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
  const { t } = useI18n();
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
      className="inline-flex min-h-11 items-center gap-1 text-accent underline"
    >
      <ChevronLeft className="size-4" aria-hidden="true" />
      {t("people.persons.back")}
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
        <Link
          to="/people"
          className="inline-flex min-h-11 items-center text-accent underline"
        >
          {t("people.persons.back")}
        </Link>
      </div>
    );
  }

  const { person, impact, voices } = state.value;
  return (
    <section
      aria-labelledby={headingId}
      className="flex flex-col gap-4 rounded-xl border border-line p-4"
    >
      {back}
      <div className="flex flex-wrap items-center gap-3">
        <Avatar name={person.display_name} size="lg" />
        <h2 id={headingId} className="text-lg font-semibold">
          {person.display_name}
        </h2>
        {person.user_id !== null && (
          <StatusPill tone="ok">{t("people.persons.linkedAccount")}</StatusPill>
        )}
        {person.voice_profile_deleted_at !== null && (
          <StatusPill tone="idle">
            {t("people.persons.profileDeleted")}
          </StatusPill>
        )}
      </div>
      <div
        role="group"
        aria-label={t("people.persons.figuresLabel", {
          name: person.display_name,
        })}
        className="flex flex-wrap items-center gap-x-3 gap-y-1"
      >
        <span>
          <Counts person={person} />
        </span>
        <span className="text-mute">
          {t("people.persons.videoCount", { count: person.clip_count })}
          {" · "}
          {speechText(person.speaking_seconds, t)}
        </span>
        <ReferenceBadge samples={person.validated_sample_count} />
      </div>
      {referenceLevel(person.validated_sample_count) !== "good" && (
        <div role="note" className="rounded-lg bg-warn-bg p-3 text-warn-fg">
          <b className="block">{t("people.persons.referenceWarningTitle")}</b>
          {t(
            referenceLevel(person.validated_sample_count) === "none"
              ? "people.persons.referenceWarningNone"
              : "people.persons.referenceWarningWeak",
            { name: person.display_name },
          )}
        </div>
      )}
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
      <PersonVideos person={person} voices={voices} />
      <PersonVoices person={person} voices={voices} />
    </section>
  );
}
