import { useCallback, useState } from "react";
import { useSearchParams } from "react-router";
import { useLoaded } from "../api/loaded";
import { ErrorAlert, StatusNotice } from "../components/Alert";
import { buttonClass } from "../components/buttons";
import { useI18n } from "../i18n";
import { DecidePanel } from "./DecidePanel";
import { ExcerptDialog, ExcerptThumb } from "./ExcerptThumb";
import { useFailureText } from "./failureText";
import { LevelBadge } from "./LevelBadge";
import { QueryPlaceholder } from "./QueryPlaceholder";
import {
  fetchPersons,
  fetchVerification,
  type VerificationCase,
  type VerificationFilter,
} from "./peopleApi";

const PAGE_SIZE = 10;

const filterClass = (pressed: boolean) =>
  `inline-flex min-h-11 items-center gap-2 rounded-lg border px-3 text-sm font-medium md:min-h-9 ${
    pressed
      ? "border-accent bg-accent-soft text-accent-ink"
      : "border-line-strong bg-surface text-ink hover:bg-side"
  }`;

/**
 * The Verification section: unknown and ambiguous voices, most frequent first. The counters
 * filter the list; settling a case is the ordinary human validation of the voice.
 */
export function VerificationTab() {
  const { t } = useI18n();
  const failureText = useFailureText();
  // The address of a verification case; its processing screen is not built yet.
  const caseId = useSearchParams()[0].get("case");
  const [filter, setFilter] = useState<VerificationFilter>("all");
  const [page, setPage] = useState(1);
  const load = useCallback(
    () => fetchVerification(filter, page, PAGE_SIZE),
    [filter, page],
  );
  const { state, reload } = useLoaded(load);
  const people = useLoaded(fetchPersons);
  const [previewing, setPreviewing] = useState<VerificationCase>();
  const [deciding, setDeciding] = useState<VerificationCase>();
  const [notice, setNotice] = useState<string>();

  const data = state.status === "ready" ? state.value : undefined;
  const counts = data?.counts;
  const pages = Math.max(1, Math.ceil((data?.total ?? 0) / PAGE_SIZE));

  function choose(next: VerificationFilter) {
    setFilter(next);
    setPage(1);
  }

  function settled(message: string) {
    setDeciding(undefined);
    setPreviewing(undefined);
    setNotice(message);
    // The counters and the list are read back from the server; a page that is now past
    // the end goes back to the last one.
    if (data && page > 1 && data.items.length === 1) setPage(page - 1);
    else reload();
    people.reload();
  }

  const open = (item: VerificationCase) => {
    setPreviewing(undefined);
    setDeciding(item);
  };

  return (
    <div className="flex flex-col gap-4">
      {caseId && <QueryPlaceholder kind="case" value={caseId} />}
      <p className="max-w-2xl text-mute">{t("people.verification.lead")}</p>
      {counts && (
        <ul
          aria-label={t("people.verification.countersLabel")}
          className="m-0 flex list-none flex-wrap gap-2 p-0"
        >
          <li>
            <button
              type="button"
              aria-pressed={filter === "all"}
              onClick={() => choose("all")}
              className={filterClass(filter === "all")}
            >
              {t("people.verification.toDecide")}
              <b>{counts.unknown + counts.ambiguous}</b>
            </button>
          </li>
          {(["unknown", "ambiguous"] as const).map((level) => (
            <li key={level}>
              <button
                type="button"
                aria-pressed={filter === level}
                onClick={() => choose(level)}
                className={filterClass(filter === level)}
              >
                <LevelBadge level={level} count={counts[level]} />
              </button>
            </li>
          ))}
          {(["human", "automatic", "suggestion"] as const).map((level) => (
            <li key={level} className="flex min-h-9 items-center px-1">
              <LevelBadge level={level} count={counts[level]} />
            </li>
          ))}
        </ul>
      )}
      {notice && <StatusNotice>{notice}</StatusNotice>}
      {state.status === "loading" && (
        <p className="text-mute">{t("people.loading")}</p>
      )}
      {state.status === "failed" && (
        <ErrorAlert>{failureText(state.code)}</ErrorAlert>
      )}
      {data && data.items.length === 0 && (
        <p className="rounded-xl border border-dashed border-line-strong p-6 text-center text-mute">
          {t("people.verification.empty")}
        </p>
      )}
      {data && data.items.length > 0 && (
        <ul
          aria-label={t("people.verification.listLabel")}
          className="m-0 flex list-none flex-col gap-2 p-0"
        >
          {data.items.map((item) => (
            <li
              key={item.local_speaker_id}
              className="flex flex-wrap items-center gap-3 rounded-xl border border-line bg-surface p-3"
            >
              <ExcerptThumb item={item} onOpen={() => setPreviewing(item)} />
              <div className="flex min-w-0 flex-1 basis-60 flex-col gap-1">
                <div className="flex flex-wrap items-center gap-2">
                  <b>{item.label}</b>
                  <LevelBadge level={item.decision_level} />
                </div>
                <span className="truncate text-mute">
                  {t("people.verification.clipLine", {
                    clip: item.clip_name,
                    speech: t("people.verification.speech", {
                      seconds: item.speaking_seconds.toFixed(1),
                    }),
                  })}
                </span>
                <span className="text-mute">
                  {item.group_voice_count > 1
                    ? t("people.verification.heardIn", {
                        clips: item.group_clip_count,
                        voices: item.group_voice_count,
                      })
                    : t("people.verification.single")}
                </span>
                {item.candidates.length > 0 && (
                  <ul
                    aria-label={t("people.verification.candidates")}
                    className="m-0 flex list-none flex-wrap gap-1.5 p-0"
                  >
                    {item.candidates.map((candidate) => (
                      <li
                        key={candidate.person.id}
                        className="rounded-full bg-accent-soft px-2.5 py-0.5 text-xs font-medium text-accent-ink"
                      >
                        {candidate.person.display_name}{" "}
                        {Math.round(candidate.score * 100)}%
                      </li>
                    ))}
                  </ul>
                )}
              </div>
              <button
                type="button"
                aria-label={t("people.verification.decideFor", {
                  label: item.label,
                })}
                onClick={() => open(item)}
                className={buttonClass("primary")}
              >
                {t("people.verification.decide")}
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
            {t("people.verification.previous")}
          </button>
          <span className="text-mute">
            {t("people.verification.pageStatus", { page, pages })}
          </span>
          <button
            type="button"
            disabled={page >= pages}
            onClick={() => setPage(page + 1)}
            className={buttonClass("secondary")}
          >
            {t("people.verification.next")}
          </button>
        </div>
      )}
      <ExcerptDialog
        item={previewing}
        onClose={() => setPreviewing(undefined)}
        onDecide={open}
      />
      <DecidePanel
        item={deciding}
        persons={people.state.status === "ready" ? people.state.value : []}
        onClose={() => setDeciding(undefined)}
        onSettled={settled}
      />
    </div>
  );
}
