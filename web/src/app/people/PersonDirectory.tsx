import { ChevronLeft, ChevronRight, Search, UserPlus } from "lucide-react";
import { useCallback, useState } from "react";
import { createPortal } from "react-dom";
import { Link, useNavigate, useOutletContext, useParams } from "react-router";
import type { PeopleOutletContext } from "./PeopleLayout";
import { useLoaded } from "../api/loaded";
import { ErrorAlert, StatusNotice } from "../components/Alert";
import { buttonClass } from "../components/buttons";
import { StatusPill } from "../components/StatusPill";
import { useI18n } from "../i18n";
import { CorrectionLog } from "./CorrectionLog";
import { useFailureText } from "./failureText";
import { NewPersonDialog } from "./PersonDialogs";
import { Avatar, ReferenceBadge, speechText } from "./PersonParts";
import { PersonRecord } from "./PersonRecord";
import { fetchPersonPage } from "./peopleApi";

const PAGE_SIZE = 25;

/**
 * The People section: a grid of cards searched and paged by the server, and the record of the
 * chosen Person as a full page at its own address (/people/:id).
 */
export function PersonDirectory() {
  const { t } = useI18n();
  const failureText = useFailureText();
  const navigate = useNavigate();
  const { id } = useParams();
  const { actionsSlot } = useOutletContext<PeopleOutletContext>();
  const [query, setQuery] = useState("");
  const [page, setPage] = useState(1);
  const [creating, setCreating] = useState(false);
  const [notice, setNotice] = useState<string>();
  const [changes, setChanges] = useState(0);

  const search = query.trim();
  const load = useCallback(
    () => fetchPersonPage(search, page, PAGE_SIZE),
    // `changes` makes the list load again after a creation, merge, separation or deletion.
    [search, page, changes],
  );
  const { state } = useLoaded(load);

  function changed(message: string) {
    setNotice(message);
    setChanges((n) => n + 1);
  }

  const selected = id !== undefined;
  const items = state.status === "ready" ? state.value.items : [];
  const total = state.status === "ready" ? state.value.total : 0;
  const pages = Math.max(1, Math.ceil(total / PAGE_SIZE));

  return (
    <div className="flex flex-col gap-4">
      {!selected &&
        actionsSlot &&
        createPortal(
          <>
            <label className="min-w-0">
              <span className="sr-only">{t("people.persons.searchLabel")}</span>
              <span className="flex h-11 items-center gap-2 rounded-lg border border-line-strong bg-surface px-3 md:h-9">
                <Search
                  className="size-4 shrink-0 text-mute"
                  aria-hidden="true"
                />
                <input
                  type="search"
                  value={query}
                  onChange={(event) => {
                    setQuery(event.target.value);
                    setPage(1);
                  }}
                  placeholder={t("people.persons.searchPlaceholder")}
                  className="min-w-0 flex-1 bg-transparent text-ink outline-none"
                />
              </span>
            </label>
            <button
              type="button"
              onClick={() => setCreating(true)}
              className={buttonClass("secondary")}
            >
              <UserPlus className="size-4" aria-hidden="true" />
              {t("people.persons.new")}
            </button>
          </>,
          actionsSlot,
        )}
      {notice && <StatusNotice>{notice}</StatusNotice>}
      {selected ? (
        <PersonRecord
          id={id}
          version={changes}
          onChanged={changed}
          onMerged={(target, message) => {
            navigate(`/people/${target.id}`);
            changed(message);
          }}
          onDeleted={(message) => {
            navigate("/people");
            changed(message);
          }}
        />
      ) : (
        <div className="flex min-w-0 flex-col gap-3">
          {state.status === "loading" && (
            <p className="text-mute">{t("people.loading")}</p>
          )}
          {state.status === "failed" && (
            <ErrorAlert>{failureText(state.code)}</ErrorAlert>
          )}
          {state.status === "ready" && total === 0 && search === "" && (
            <p className="text-mute">{t("people.persons.empty")}</p>
          )}
          {state.status === "ready" && total === 0 && search !== "" && (
            <p className="text-mute">{t("people.persons.noMatch")}</p>
          )}
          {items.length > 0 && (
            <ul
              aria-label={t("people.persons.listLabel")}
              className="m-0 grid list-none grid-cols-[repeat(auto-fill,minmax(200px,1fr))] gap-3 p-0"
            >
              {items.map((person) => (
                <li key={person.id}>
                  <Link
                    to={`/people/${person.id}`}
                    className="flex h-full min-h-11 w-full flex-col items-center gap-2 rounded-xl border border-line p-4 text-center hover:border-accent"
                  >
                    <Avatar name={person.display_name} size="lg" />
                    <span className="font-semibold break-words">
                      {person.display_name}
                    </span>
                    {person.user_id !== null && (
                      <StatusPill tone="ok">
                        {t("people.persons.account")}
                      </StatusPill>
                    )}
                    <span className="text-xs text-mute">
                      {t("people.persons.videoCount", {
                        count: person.clip_count,
                      })}
                      {" · "}
                      {speechText(person.speaking_seconds, t)}
                    </span>
                    <ReferenceBadge samples={person.validated_sample_count} />
                  </Link>
                </li>
              ))}
            </ul>
          )}
          {pages <= 1 && total > 0 && (
            <p className="m-0 text-center text-sm text-mute">
              {t("people.persons.pageStatus", { page, pages, total })}
            </p>
          )}
          {pages > 1 && (
            <nav
              aria-label={t("people.persons.pagesLabel")}
              className="flex flex-wrap items-center justify-between gap-2"
            >
              <button
                type="button"
                disabled={page <= 1}
                onClick={() => setPage(page - 1)}
                className={buttonClass("secondary")}
              >
                <ChevronLeft className="size-4" aria-hidden="true" />
                {t("people.persons.previousPage")}
              </button>
              <span className="text-sm text-mute">
                {t("people.persons.pageStatus", { page, pages, total })}
              </span>
              <button
                type="button"
                disabled={page >= pages}
                onClick={() => setPage(page + 1)}
                className={buttonClass("secondary")}
              >
                {t("people.persons.nextPage")}
                <ChevronRight className="size-4" aria-hidden="true" />
              </button>
            </nav>
          )}
        </div>
      )}
      {!selected && <CorrectionLog version={changes} />}
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
