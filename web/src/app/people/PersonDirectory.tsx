import { ChevronLeft, ChevronRight, Plus } from "lucide-react";
import { useCallback, useId, useState } from "react";
import { Link, NavLink, useNavigate, useParams } from "react-router";
import { useLoaded } from "../api/loaded";
import { ErrorAlert, StatusNotice } from "../components/Alert";
import { buttonClass } from "../components/buttons";
import { ConfirmDialog } from "../components/ConfirmDialog";
import { StatusPill } from "../components/StatusPill";
import { useI18n } from "../i18n";
import { CorrectionLog } from "./CorrectionLog";
import { useFailureText } from "./failureText";
import { MergeDialog, NewPersonDialog } from "./PersonDialogs";
import { PersonVoices } from "./PersonVoices";
import {
  fetchPerson,
  fetchPersonPage,
  usePersonActions,
  type Person,
} from "./peopleApi";

const PAGE_SIZE = 25;

const itemClass = ({ isActive }: { isActive: boolean }) =>
  `flex min-h-11 flex-col gap-0.5 rounded-lg border px-3 py-2 ${
    isActive
      ? "border-accent bg-accent-soft"
      : "border-line bg-surface hover:bg-side"
  }`;

/** What a Person has, as the server counts it: voices and validated samples (the reference). */
function Counts({ person }: { person: Person }) {
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

/**
 * The People section: a compact list searched and paged by the server, and the record of the
 * chosen Person at its own address (/people/:id). On a phone they are two screens in turn.
 */
export function PersonDirectory() {
  const { t } = useI18n();
  const failureText = useFailureText();
  const actions = usePersonActions();
  const navigate = useNavigate();
  const { id } = useParams();
  const [query, setQuery] = useState("");
  const [page, setPage] = useState(1);
  const [creating, setCreating] = useState(false);
  const [merging, setMerging] = useState<Person>();
  const [deleting, setDeleting] = useState<Person>();
  const [notice, setNotice] = useState<string>();
  const [failure, setFailure] = useState<string>();
  const [busy, setBusy] = useState(false);
  const [changes, setChanges] = useState(0);

  const search = query.trim();
  const load = useCallback(
    () => fetchPersonPage(search, page, PAGE_SIZE),
    // `changes` makes the list load again after a creation, merge, separation or deletion.
    [search, page, changes],
  );
  const { state } = useLoaded(load);

  function changed(message: string) {
    setFailure(undefined);
    setNotice(message);
    setChanges((n) => n + 1);
  }

  async function remove(person: Person) {
    setBusy(true);
    const result = await actions.remove(person.id);
    setBusy(false);
    setDeleting(undefined);
    if (!result.ok) {
      setNotice(undefined);
      setFailure(result.code);
      return;
    }
    navigate("/people");
    changed(
      t("people.delete.done", {
        name: person.display_name,
        summary: t("people.summary", {
          attributions: result.value.attributions,
          samples: result.value.samples,
          rules: result.value.microphone_rules,
        }),
      }),
    );
  }

  const selected = id !== undefined;
  const items = state.status === "ready" ? state.value.items : [];
  const total = state.status === "ready" ? state.value.total : 0;
  const pages = Math.max(1, Math.ceil(total / PAGE_SIZE));

  return (
    <div className="flex flex-col gap-4">
      {notice && <StatusNotice>{notice}</StatusNotice>}
      {failure && <ErrorAlert>{failureText(failure)}</ErrorAlert>}
      <div className="grid gap-6 min-[821px]:grid-cols-[minmax(280px,360px)_1fr]">
        <div
          className={`flex min-w-0 flex-col gap-3 ${selected ? "max-[820px]:hidden" : ""}`}
        >
          <div className="flex flex-wrap items-center gap-3">
            <label className="min-w-0 flex-1">
              <span className="sr-only">{t("people.persons.searchLabel")}</span>
              <input
                type="search"
                value={query}
                onChange={(event) => {
                  setQuery(event.target.value);
                  setPage(1);
                }}
                placeholder={t("people.persons.searchPlaceholder")}
                className="h-11 w-full rounded-lg border border-line-strong bg-surface px-3 text-ink md:h-10"
              />
            </label>
            <button
              type="button"
              onClick={() => setCreating(true)}
              className={buttonClass("primary")}
            >
              <Plus className="size-4" aria-hidden="true" />
              {t("people.persons.new")}
            </button>
          </div>
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
              className="m-0 flex list-none flex-col gap-1.5 p-0"
            >
              {items.map((person) => (
                <li key={person.id}>
                  <NavLink to={`/people/${person.id}`} className={itemClass}>
                    <span className="flex flex-wrap items-center gap-2">
                      <span className="font-medium">{person.display_name}</span>
                      {person.user_id !== null && (
                        <StatusPill tone="ok">
                          {t("people.persons.account")}
                        </StatusPill>
                      )}
                    </span>
                    <span className="text-sm text-mute">
                      <Counts person={person} />
                    </span>
                  </NavLink>
                </li>
              ))}
            </ul>
          )}
          {total > 0 && (
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
        <div className={`min-w-0 ${selected ? "" : "max-[820px]:hidden"}`}>
          {selected ? (
            <PersonRecord
              id={id}
              version={changes}
              onMerge={setMerging}
              onDelete={setDeleting}
              onSeparated={changed}
            />
          ) : (
            <p className="text-mute">{t("people.persons.choose")}</p>
          )}
        </div>
      </div>
      <CorrectionLog version={changes} />
      <NewPersonDialog
        open={creating}
        onOpenChange={setCreating}
        onCreated={(person) => {
          setCreating(false);
          changed(t("people.persons.created", { name: person.display_name }));
        }}
      />
      <MergeDialog
        person={merging}
        onClose={() => setMerging(undefined)}
        onMerged={(source, target, summary) => {
          setMerging(undefined);
          navigate(`/people/${target.id}`);
          changed(
            t("people.merge.done", {
              name: source.display_name,
              target: target.display_name,
              summary,
            }),
          );
        }}
      />
      <ConfirmDialog
        open={deleting !== undefined}
        onOpenChange={(open) => !open && setDeleting(undefined)}
        title={t("people.delete.title", { name: deleting?.display_name ?? "" })}
        cancelLabel={t("people.delete.cancel")}
        confirmLabel={t("people.delete.submit")}
        danger
        busy={busy}
        onConfirm={() => deleting && void remove(deleting)}
      >
        <p>{t("people.delete.body", { name: deleting?.display_name ?? "" })}</p>
      </ConfirmDialog>
    </div>
  );
}

/** The record of one Person: its real counts, its account, its confirmed voices. */
function PersonRecord({
  id,
  version,
  onMerge,
  onDelete,
  onSeparated,
}: {
  id: string;
  version: number;
  onMerge: (person: Person) => void;
  onDelete: (person: Person) => void;
  onSeparated: (message: string) => void;
}) {
  const { t } = useI18n();
  const failureText = useFailureText();
  const headingId = useId();
  const valid = /^[1-9]\d{0,14}$/.test(id);
  const load = useCallback(
    () =>
      valid
        ? fetchPerson(Number(id))
        : Promise.resolve({ ok: false as const, code: "person_not_found" }),
    // `version` reloads the record after a change made from the list or the record.
    [id, valid, version],
  );
  const { state } = useLoaded(load);

  const back = (
    <Link
      to="/people"
      className="inline-flex min-h-11 items-center gap-1 text-accent underline min-[821px]:hidden"
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
          className="inline-flex min-h-11 items-center text-accent underline max-[820px]:hidden"
        >
          {t("people.persons.back")}
        </Link>
      </div>
    );
  }

  const person = state.value;
  const locked = person.user_id !== null;
  return (
    <section
      aria-labelledby={headingId}
      className="flex flex-col gap-4 rounded-xl border border-line p-4"
    >
      {back}
      <div className="flex flex-wrap items-center gap-2">
        <h2 id={headingId} className="text-lg font-semibold">
          {person.display_name}
        </h2>
        {locked && (
          <StatusPill tone="ok">{t("people.persons.linkedAccount")}</StatusPill>
        )}
        {person.voice_profile_deleted_at !== null && (
          <StatusPill tone="idle">
            {t("people.persons.profileDeleted")}
          </StatusPill>
        )}
      </div>
      <p>
        <Counts person={person} />
      </p>
      <div className="flex flex-wrap gap-2">
        <button
          type="button"
          disabled={locked || person.voice_profile_deleted_at !== null}
          onClick={() => onMerge(person)}
          className={buttonClass("secondary")}
        >
          {t("people.persons.merge", { name: person.display_name })}
        </button>
        <button
          type="button"
          disabled={locked}
          onClick={() => onDelete(person)}
          className={buttonClass("secondary")}
        >
          {t("people.persons.delete", { name: person.display_name })}
        </button>
      </div>
      <PersonVoices person={person} onSeparated={onSeparated} />
    </section>
  );
}
