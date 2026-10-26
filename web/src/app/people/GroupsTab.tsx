import { useCallback, useState } from "react";
import { useLoaded } from "../api/loaded";
import { ErrorAlert, StatusNotice } from "../components/Alert";
import { buttonClass } from "../components/buttons";
import { useI18n } from "../i18n";
import { tintOf } from "./ExcerptThumb";
import { GroupPanel } from "./GroupPanel";
import { useFailureText } from "./failureText";
import { fetchPersons, fetchVoiceGroups, type VoiceGroup } from "./peopleApi";

const PAGE_SIZE = 10;
const PREVIEW = 4;

/** The distinct clip names of a group, in order, for its summary line. */
function clipNames(group: VoiceGroup) {
  const names = [...new Set(group.members.map((m) => m.clip_name))];
  return names.length > 2
    ? `${names.slice(0, 2).join(", ")}…`
    : names.join(", ");
}

/**
 * The Voice groups section: unknown voices that sound alike, most frequent first. A group is
 * only a proposal; naming it, from the panel, is the one action that confirms it.
 */
export function GroupsTab() {
  const { t } = useI18n();
  const failureText = useFailureText();
  const [page, setPage] = useState(1);
  const load = useCallback(() => fetchVoiceGroups(page, PAGE_SIZE), [page]);
  const { state, reload } = useLoaded(load);
  const people = useLoaded(fetchPersons);
  const [opened, setOpened] = useState<VoiceGroup>();
  const [notice, setNotice] = useState<string>();

  const data = state.status === "ready" ? state.value : undefined;
  const pages = Math.max(1, Math.ceil((data?.total ?? 0) / PAGE_SIZE));

  // Leaving a voice out is saved as it happens, so the list is read back on every close.
  function closed(message?: string) {
    setOpened(undefined);
    if (message) setNotice(message);
    if (message && data && page > 1 && data.items.length === 1)
      setPage(page - 1);
    else reload();
    if (message) people.reload();
  }

  return (
    <div className="flex flex-col gap-4">
      <p className="max-w-2xl text-mute">{t("people.groups.lead")}</p>
      {notice && <StatusNotice>{notice}</StatusNotice>}
      {state.status === "loading" && (
        <p className="text-mute">{t("people.loading")}</p>
      )}
      {state.status === "failed" && (
        <ErrorAlert>{failureText(state.code)}</ErrorAlert>
      )}
      {data && data.items.length === 0 && (
        <p className="rounded-xl border border-dashed border-line-strong p-6 text-center text-mute">
          {t("people.groups.empty")}
        </p>
      )}
      {data && data.items.length > 0 && (
        <ul
          aria-label={t("people.groups.listLabel")}
          className="m-0 grid list-none grid-cols-1 gap-3 p-0 md:grid-cols-2"
        >
          {data.items.map((group) => (
            <li
              key={group.id}
              className="flex flex-col items-start gap-2 rounded-xl border border-line bg-surface p-4"
            >
              <div
                role="img"
                aria-label={t("people.groups.preview")}
                className="flex items-center"
              >
                {group.members.slice(0, PREVIEW).map((member, index) => (
                  <span
                    key={member.local_speaker_id}
                    className={`-ml-2 size-9 rounded-full border-2 border-surface first:ml-0 ${tintOf(member.local_speaker_id)}`}
                    style={{ zIndex: PREVIEW - index }}
                  />
                ))}
                {group.member_count > PREVIEW && (
                  <span className="-ml-2 flex size-9 items-center justify-center rounded-full border-2 border-surface bg-side text-xs font-medium text-mute">
                    +{group.member_count - PREVIEW}
                  </span>
                )}
              </div>
              <b>{t("people.groups.title", { count: group.member_count })}</b>
              <span className="text-mute">
                {t("people.groups.heardIn", {
                  clips: group.clip_count,
                  names: clipNames(group),
                })}
              </span>
              <span className="text-mute">{t("people.groups.never")}</span>
              <button
                type="button"
                onClick={() => setOpened(group)}
                aria-label={t("people.groups.openFor", {
                  count: group.member_count,
                  clip: group.members[0]?.clip_name ?? "",
                })}
                className={buttonClass("primary")}
              >
                {t("people.groups.open")}
              </button>
            </li>
          ))}
        </ul>
      )}
      {pages > 1 && (
        <div className="flex flex-wrap items-center gap-2">
          <button
            type="button"
            disabled={page <= 1}
            onClick={() => setPage(page - 1)}
            className={buttonClass("secondary")}
          >
            {t("people.groups.previous")}
          </button>
          <span className="text-mute">
            {t("people.groups.pageStatus", { page, pages })}
          </span>
          <button
            type="button"
            disabled={page >= pages}
            onClick={() => setPage(page + 1)}
            className={buttonClass("secondary")}
          >
            {t("people.groups.next")}
          </button>
        </div>
      )}
      <GroupPanel
        group={opened}
        persons={people.state.status === "ready" ? people.state.value : []}
        onClose={closed}
      />
    </div>
  );
}
