import { useCallback, useEffect, useMemo, useState } from "react";
import { useOutletContext } from "react-router";
import { useLoaded } from "../api/loaded";
import { ErrorAlert, StatusNotice } from "../components/Alert";
import { useI18n } from "../i18n";
import { useFailureText } from "./failureText";
import { LevelBadge } from "./LevelBadge";
import type { PeopleOutlet } from "./PeopleLayout";
import {
  fetchVerification,
  type VerificationCase,
  type VerificationFilter,
} from "./peopleApi";
import { VerificationCaseCard } from "./VerificationCaseCard";

// The most a page of the server holds: settled cases leave, so the next ones come in.
const QUEUE_SIZE = 100;

const filterClass = (pressed: boolean) =>
  `inline-flex min-h-11 items-center gap-2 rounded-lg border px-3 text-sm font-medium md:min-h-9 ${
    pressed
      ? "border-accent bg-accent-soft text-accent-ink"
      : "border-line-strong bg-surface text-ink hover:bg-side"
  }`;

/**
 * The Verification section: one case at a time, the most frequent voices first. The
 * counters filter the queue. "Skip" only reorders it for this session; settling a case is
 * the ordinary human validation of the voice, then the next case comes by itself.
 */
export function VerificationTab() {
  const { t } = useI18n();
  const failureText = useFailureText();
  const outlet = useOutletContext<PeopleOutlet | null>();
  const [filter, setFilter] = useState<VerificationFilter>("all");
  const load = useCallback(
    () => fetchVerification(filter, 1, QUEUE_SIZE),
    [filter],
  );
  const { state, reload } = useLoaded(load);
  // Local to the session: nothing of this is written to the server.
  const [skipped, setSkipped] = useState<number[]>([]);
  const [gone, setGone] = useState<number[]>([]);
  const [notice, setNotice] = useState<string>();

  const data = state.status === "ready" ? state.value : undefined;
  const counts = data?.counts;
  const toDecide = counts ? counts.unknown + counts.ambiguous : undefined;
  const report = outlet?.reportToDecide;
  useEffect(() => {
    if (toDecide !== undefined) report?.(toDecide);
  }, [toDecide, report]);

  // The queue: what was not skipped first, then the skipped ones in the order they were.
  const queue = useMemo(() => {
    const items = (data?.items ?? []).filter(
      (item) => !gone.includes(item.local_speaker_id),
    );
    const rank = (item: VerificationCase) =>
      skipped.indexOf(item.local_speaker_id);
    return [
      ...items.filter((item) => rank(item) < 0),
      ...items
        .filter((item) => rank(item) >= 0)
        .sort((a, b) => rank(a) - rank(b)),
    ];
  }, [data, skipped, gone]);
  const current = queue[0];

  function choose(next: VerificationFilter) {
    setFilter(next);
    setNotice(undefined);
  }

  function settled(item: VerificationCase, message: string) {
    setGone((ids) => [...ids, item.local_speaker_id]);
    setNotice(message);
    reload();
  }

  function skip(item: VerificationCase) {
    setSkipped((ids) => [
      ...ids.filter((id) => id !== item.local_speaker_id),
      item.local_speaker_id,
    ]);
    setNotice(undefined);
  }

  return (
    <div className="flex flex-col gap-4">
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
                <LevelBadge
                  level={level}
                  count={counts[level]}
                  interactive={false}
                />
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
      {data && !current && (
        <p className="rounded-xl border border-dashed border-line-strong p-6 text-center text-mute">
          {t("people.verification.empty")}
        </p>
      )}
      {current && (
        <VerificationCaseCard
          // Mounted per case, so a new case starts without a choice.
          key={current.local_speaker_id}
          item={current}
          total={queue.length}
          canSkip={queue.length > 1}
          onSkip={() => skip(current)}
          onSettled={(message) => settled(current, message)}
          onTaken={(message) => {
            // Someone else settled it meanwhile: nothing was written, the queue is read again.
            settled(current, message);
          }}
        />
      )}
    </div>
  );
}
