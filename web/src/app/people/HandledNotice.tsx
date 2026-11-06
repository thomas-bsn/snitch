import { Link } from "react-router";
import { buttonClass } from "../components/buttons";
import { useI18n } from "../i18n";
import type { LocalSpeaker } from "../library/libraryApi";

/**
 * Shown on the processing screen when a direct link (`?case=`) names a voice that is already
 * settled (ADR 0025): who attributed it, to whom, when, with a link to the Person. The
 * screen stays usable below it.
 */
export function HandledNotice({ voice }: { voice: LocalSpeaker }) {
  const { t, language } = useI18n();
  const attribution = voice.attribution;
  const person = voice.person ?? attribution?.person ?? null;
  const name = person?.display_name ?? "";
  let text: string;
  if (voice.disposition || !person) {
    text = t("people.verification.handledNoPerson", { label: voice.label });
  } else if (voice.decision_level === "human") {
    text = t("people.verification.handledBy", {
      label: voice.label,
      person: name,
      author: attribution?.author?.username ?? t("people.log.deletedUser"),
      date: new Date(attribution?.decided_at ?? "").toLocaleString(language),
    });
  } else {
    text = t("people.verification.handledAuto", {
      label: voice.label,
      person: name,
    });
  }
  return (
    <section
      aria-label={t("people.verification.directTitle")}
      className="flex flex-col items-start gap-3 rounded-xl border border-line p-4"
    >
      <h2 className="m-0 text-base font-semibold">
        {t("people.verification.handledTitle")}
      </h2>
      <p role="status" className="m-0">
        {text}
      </p>
      {person && !voice.disposition && (
        <Link to={`/people/${person.id}`} className={buttonClass("secondary")}>
          {t("people.verification.openPerson", { person: name })}
        </Link>
      )}
    </section>
  );
}
