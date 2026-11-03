import { Plus } from "lucide-react";
import { useState } from "react";
import { Link, useLocation } from "react-router";
import { useLoaded } from "../api/loaded";
import { ErrorAlert, StatusNotice } from "../components/Alert";
import { buttonClass } from "../components/buttons";
import { useI18n } from "../i18n";
import { CorrectionLog } from "./CorrectionLog";
import { useFailureText } from "./failureText";
import { NewPersonDialog } from "./PersonDialogs";
import { Avatar, ReferenceBadge, speechText } from "./PersonParts";
import { fetchPersons, type Person } from "./peopleApi";

/** A notice carried over from the page a Person was merged or deleted on. */
export type PeopleNotice = { notice?: string };

/** The People directory: a grid of cards, a search, and the creation of a Person. */
export function PersonList() {
  const { t } = useI18n();
  const failureText = useFailureText();
  const location = useLocation();
  const { state, reload } = useLoaded(fetchPersons);
  const [query, setQuery] = useState("");
  const [creating, setCreating] = useState(false);
  const [notice, setNotice] = useState<string | undefined>(
    (location.state as PeopleNotice | null)?.notice,
  );
  const [changes, setChanges] = useState(0);

  function changed(message: string) {
    setNotice(message);
    setChanges((n) => n + 1);
    reload();
  }

  const persons = state.status === "ready" ? state.value : [];
  const shown = persons.filter((person) =>
    person.display_name.toLowerCase().includes(query.trim().toLowerCase()),
  );

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-wrap items-center gap-3">
        <label className="flex-1 sm:max-w-xs">
          <span className="sr-only">{t("people.persons.searchLabel")}</span>
          <input
            type="search"
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            placeholder={t("people.persons.searchPlaceholder")}
            className="h-11 w-full rounded-lg border border-line-strong bg-surface px-3 text-ink md:h-10"
          />
        </label>
        <button
          type="button"
          onClick={() => setCreating(true)}
          className={`${buttonClass("primary")} ml-auto`}
        >
          <Plus className="size-4" aria-hidden="true" />
          {t("people.persons.new")}
        </button>
      </div>
      {notice && <StatusNotice>{notice}</StatusNotice>}
      {state.status === "loading" && (
        <p className="text-mute">{t("people.loading")}</p>
      )}
      {state.status === "failed" && (
        <ErrorAlert>{failureText(state.code)}</ErrorAlert>
      )}
      {state.status === "ready" && persons.length === 0 && (
        <p className="text-mute">{t("people.persons.empty")}</p>
      )}
      {state.status === "ready" && persons.length > 0 && shown.length === 0 && (
        <p className="text-mute">{t("people.persons.noMatch")}</p>
      )}
      {shown.length > 0 && (
        <ul
          aria-label={t("people.persons.listLabel")}
          className="m-0 grid list-none grid-cols-1 gap-3 p-0 sm:grid-cols-2 xl:grid-cols-3"
        >
          {shown.map((person) => (
            <PersonCard key={person.id} person={person} />
          ))}
        </ul>
      )}
      <CorrectionLog version={changes} />
      <NewPersonDialog
        open={creating}
        onOpenChange={setCreating}
        onCreated={(person) => {
          setCreating(false);
          changed(t("people.persons.created", { name: person.display_name }));
        }}
      />
    </div>
  );
}

function PersonCard({ person }: { person: Person }) {
  const { t } = useI18n();
  return (
    <li>
      <Link
        to={`/people/${person.id}`}
        className="flex h-full items-start gap-3 rounded-xl border border-line bg-surface p-4 text-ink hover:border-line-strong"
      >
        <Avatar name={person.display_name} />
        <span className="flex min-w-0 flex-1 flex-col gap-1.5">
          <span className="truncate font-semibold">
            {person.display_name}
            {person.user_id !== null && (
              <span className="ml-2 align-middle text-xs font-normal text-mute">
                {t("people.persons.linked")}
              </span>
            )}
          </span>
          <span className="text-sm text-mute">
            {t("people.persons.cardVideos", { count: person.clip_count })}
            {" · "}
            {t("people.persons.cardSpeech", {
              time: speechText(person.speaking_seconds, t),
            })}
          </span>
          <span>
            <ReferenceBadge samples={person.validated_sample_count} />
          </span>
        </span>
      </Link>
    </li>
  );
}
