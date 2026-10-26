import { Plus } from "lucide-react";
import { Fragment, useState } from "react";
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
import { fetchPersons, usePersonActions, type Person } from "./peopleApi";

const smallButton = `${buttonClass("secondary")} md:h-8 px-3`;

/** The People section: the list, and the creation, merge, separation and deletion of Persons. */
export function PersonList() {
  const { t } = useI18n();
  const failureText = useFailureText();
  const actions = usePersonActions();
  const { state, reload } = useLoaded(fetchPersons);
  const [query, setQuery] = useState("");
  const [creating, setCreating] = useState(false);
  const [merging, setMerging] = useState<Person>();
  const [deleting, setDeleting] = useState<Person>();
  const [expanded, setExpanded] = useState<number>();
  const [notice, setNotice] = useState<string>();
  const [failure, setFailure] = useState<string>();
  const [busy, setBusy] = useState(false);
  const [changes, setChanges] = useState(0);

  function changed(message: string) {
    setFailure(undefined);
    setNotice(message);
    setChanges((n) => n + 1);
    reload();
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
    if (expanded === person.id) setExpanded(undefined);
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
      {failure && <ErrorAlert>{failureText(failure)}</ErrorAlert>}
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
        <div className="overflow-x-auto rounded-xl border border-line">
          <table
            aria-label={t("people.persons.listLabel")}
            className="w-full border-collapse text-left"
          >
            <thead className="bg-side text-xs text-mute">
              <tr>
                <th scope="col" className="px-3 py-2 font-medium">
                  {t("people.persons.name")}
                </th>
                <th scope="col" className="px-3 py-2 font-medium">
                  {t("people.persons.voices")}
                </th>
                <th scope="col" className="px-3 py-2 font-medium">
                  {t("people.persons.samples")}
                </th>
                <th scope="col" className="px-3 py-2 font-medium">
                  {t("people.persons.account")}
                </th>
                <th scope="col" className="px-3 py-2 font-medium">
                  {t("people.persons.actions")}
                </th>
              </tr>
            </thead>
            <tbody>
              {shown.map((person) => {
                const open = expanded === person.id;
                const locked = person.user_id !== null;
                return (
                  <Fragment key={person.id}>
                    <tr className="border-t border-line align-middle">
                      <th scope="row" className="px-3 py-2.5 font-medium">
                        {person.display_name}
                        {person.voice_profile_deleted_at !== null && (
                          <span className="ml-2 inline-block align-middle">
                            <StatusPill tone="idle">
                              {t("people.persons.profileDeleted")}
                            </StatusPill>
                          </span>
                        )}
                      </th>
                      <td className="px-3 py-2.5">
                        {person.local_speaker_count}
                      </td>
                      <td className="px-3 py-2.5">
                        {person.validated_sample_count}
                      </td>
                      <td className="px-3 py-2.5">
                        {locked ? (
                          <StatusPill tone="ok">
                            {t("people.persons.linked")}
                          </StatusPill>
                        ) : (
                          <span className="text-mute">
                            {t("people.persons.none")}
                          </span>
                        )}
                      </td>
                      <td className="px-3 py-2.5">
                        <div className="flex flex-wrap gap-2">
                          <button
                            type="button"
                            aria-expanded={open}
                            onClick={() =>
                              setExpanded(open ? undefined : person.id)
                            }
                            className={smallButton}
                          >
                            {t(
                              open
                                ? "people.persons.hideVoices"
                                : "people.persons.showVoices",
                              { name: person.display_name },
                            )}
                          </button>
                          <button
                            type="button"
                            disabled={
                              locked || person.voice_profile_deleted_at !== null
                            }
                            onClick={() => setMerging(person)}
                            className={smallButton}
                          >
                            {t("people.persons.merge", {
                              name: person.display_name,
                            })}
                          </button>
                          <button
                            type="button"
                            disabled={locked}
                            onClick={() => setDeleting(person)}
                            className={smallButton}
                          >
                            {t("people.persons.delete", {
                              name: person.display_name,
                            })}
                          </button>
                        </div>
                      </td>
                    </tr>
                    {open && (
                      <tr className="bg-side">
                        <td colSpan={5} className="px-3 py-3">
                          <PersonVoices person={person} onSeparated={changed} />
                        </td>
                      </tr>
                    )}
                  </Fragment>
                );
              })}
            </tbody>
          </table>
        </div>
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
      <MergeDialog
        person={merging}
        candidates={persons}
        onClose={() => setMerging(undefined)}
        onMerged={(source, target, summary) => {
          setMerging(undefined);
          if (expanded === source.id) setExpanded(undefined);
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
