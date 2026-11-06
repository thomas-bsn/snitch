import { ChevronRight, CircleHelp } from "lucide-react";
import { useCallback, useState } from "react";
import { useLoaded } from "../api/loaded";
import { useAddressParams } from "../api/useAddressParams";
import { ErrorAlert, StatusNotice } from "../components/Alert";
import { buttonClass } from "../components/buttons";
import { useI18n } from "../i18n";
import { GroupDetail } from "./GroupDetail";
import { useFailureText } from "./failureText";
import { fetchVoiceGroups, type VoiceGroup } from "./peopleApi";

const PAGE_SIZE = 10;

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
  const { params, update } = useAddressParams();
  const [page, setPage] = useState(1);
  const load = useCallback(() => fetchVoiceGroups(page, PAGE_SIZE), [page]);
  const { state, reload } = useLoaded(load);
  const [notice, setNotice] = useState<string>();

  const data = state.status === "ready" ? state.value : undefined;
  const pages = Math.max(1, Math.ceil((data?.total ?? 0) / PAGE_SIZE));
  const voice = Number(params.get("voice"));
  const opened = Number.isInteger(voice) && voice > 0 ? voice : undefined;

  function open(group: VoiceGroup) {
    setNotice(undefined);
    update((next) =>
      next.set("voice", String(group.members[0].local_speaker_id)),
    );
  }

  // Leaving a voice out is saved as it happens, so the list is read back on every return.
  function close(message?: string) {
    update((next) => next.delete("voice"));
    if (message) setNotice(message);
    if (message && data && page > 1 && data.items.length === 1)
      setPage(page - 1);
    else reload();
  }

  return (
    <div className="flex flex-col gap-4">
      <p className="m-0 text-mute">{t("people.groups.lead")}</p>
      {opened !== undefined ? (
        <GroupDetail
          key={opened}
          voice={opened}
          onBack={() => close()}
          onNamed={close}
        />
      ) : (
        <>
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
              className="m-0 flex list-none flex-col gap-2 p-0"
            >
              {data.items.map((group) => (
                <li key={group.id}>
                  <button
                    type="button"
                    onClick={() => open(group)}
                    className="flex w-full items-center gap-3 rounded-lg border border-line px-3 py-2 text-left hover:bg-side"
                  >
                    <span
                      aria-hidden="true"
                      className="flex size-9 shrink-0 items-center justify-center rounded-full bg-side"
                    >
                      <CircleHelp className="size-4 text-mute" />
                    </span>
                    <span className="min-w-0 flex-1">
                      <span className="block font-medium">
                        {t("people.groups.title", {
                          count: group.member_count,
                        })}
                      </span>
                      <span className="block truncate text-xs text-mute">
                        {t("people.groups.heardIn", {
                          clips: group.clip_count,
                          names: clipNames(group),
                        })}
                      </span>
                    </span>
                    <ChevronRight
                      className="size-4 shrink-0 text-mute"
                      aria-hidden="true"
                    />
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
        </>
      )}
    </div>
  );
}
