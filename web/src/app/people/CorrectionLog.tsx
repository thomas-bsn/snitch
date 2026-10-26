import { useState } from "react";
import { useLoaded } from "../api/loaded";
import { ErrorAlert } from "../components/Alert";
import { buttonClass } from "../components/buttons";
import { useI18n } from "../i18n";
import { useFailureText } from "./failureText";
import { fetchPersonEvents } from "./peopleApi";

function Events() {
  const { t, language } = useI18n();
  const failureText = useFailureText();
  const { state } = useLoaded(fetchPersonEvents);
  if (state.status === "loading")
    return <p className="text-mute">{t("people.loading")}</p>;
  if (state.status === "failed")
    return <ErrorAlert>{failureText(state.code)}</ErrorAlert>;
  if (state.value.length === 0)
    return <p className="text-mute">{t("people.log.empty")}</p>;
  return (
    <ul
      aria-label={t("people.log.label")}
      className="m-0 flex list-none flex-col gap-1.5 p-0"
    >
      {state.value.map((event) => (
        <li key={event.id} className="rounded-lg border border-line p-2.5">
          <b>
            {t(`people.log.${event.action}`, {
              person: event.person.display_name,
              target: event.target?.display_name ?? "",
            })}
          </b>
          <span className="block text-mute">
            {t("people.log.detail", {
              count: event.attribution_count,
              author: event.author?.username ?? t("people.log.deletedUser"),
              date: new Date(event.created_at).toLocaleString(language),
            })}
          </span>
        </li>
      ))}
    </ul>
  );
}

/** The immutable trail of merges, separations and deletions, shown on demand. */
export function CorrectionLog({ version }: { version: number }) {
  const { t } = useI18n();
  const [open, setOpen] = useState(false);
  return (
    <div className="flex flex-col gap-2">
      <div>
        <button
          type="button"
          aria-expanded={open}
          onClick={() => setOpen(!open)}
          className={buttonClass("secondary")}
        >
          {open ? t("people.log.hide") : t("people.log.show")}
        </button>
      </div>
      {/* A new key loads the trail again after a correction. */}
      {open && <Events key={version} />}
    </div>
  );
}
