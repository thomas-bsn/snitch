import { useCallback, useEffect, useMemo, useState } from "react";
import { Link, useOutletContext, useSearchParams } from "react-router";
import { useLoaded } from "../api/loaded";
import { buttonClass } from "../components/buttons";
import { ErrorAlert, StatusNotice } from "../components/Alert";
import { useI18n } from "../i18n";
import { useFailureText } from "./failureText";
import { LevelBadge } from "./LevelBadge";
import type { PeopleOutlet } from "./PeopleLayout";
import {
  fetchLocalSpeaker,
  fetchVerification,
  type LocalSpeakerRead,
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

/** The case a direct link points at, in the shape of a case of the queue. */
function toCase(read: LocalSpeakerRead): VerificationCase | undefined {
  if (read.decision_level !== "unknown" && read.decision_level !== "ambiguous")
    return undefined;
  return {
    local_speaker_id: read.local_speaker_id,
    clip_id: read.clip_id,
    clip_name: read.clip_name,
    label: read.label,
    speaking_seconds: read.speaking_seconds,
    decision_level: read.decision_level,
    candidates: read.attribution?.candidates ?? [],
    group_id: read.group_id ?? read.local_speaker_id,
    group_voice_count: read.group_voice_count ?? 1,
    group_clip_count: read.group_clip_count ?? 1,
  };
}

/**
 * Where a direct link to a voice that is no longer to decide lands: who attributed it
 * to whom and when, or the same message for a voice that is missing or hidden.
 */
function DirectLinkNotice({
  read,
  onContinue,
}: {
  read: LocalSpeakerRead | undefined;
  onContinue: () => void;
}) {
  const { t, language } = useI18n();
  const person = read?.attribution?.person;
  const author = read?.attribution?.author;
  let text = t("people.verification.unavailable");
  let title = t("people.verification.unavailableTitle");
  if (read) {
    title = t("people.verification.handledTitle");
    const name = person?.display_name ?? "";
    if (read.decision_level === "human") {
      text = t("people.verification.handledBy", {
        label: read.label,
        person: name,
        author: author?.username ?? t("people.log.deletedUser"),
        date: new Date(read.attribution?.decided_at ?? "").toLocaleString(
          language,
        ),
      });
    } else if (person) {
      text = t("people.verification.handledAuto", {
        label: read.label,
        person: name,
      });
    } else {
      text = t("people.verification.handledNoPerson", { label: read.label });
    }
  }
  return (
    <section
      aria-label={t("people.verification.directTitle")}
      className="flex flex-col items-start gap-3 rounded-xl border border-line p-4"
    >
      <h2 className="m-0 text-base font-semibold">{title}</h2>
      <p role="status" className="m-0">
        {text}
      </p>
      <div className="flex flex-wrap gap-2">
        {person && (
          <Link
            to={`/people/${person.id}`}
            className={buttonClass("secondary")}
          >
            {t("people.verification.openPerson", {
              person: person.display_name,
            })}
          </Link>
        )}
        <button
          type="button"
          onClick={onContinue}
          className={buttonClass("secondary")}
        >
          {t("people.verification.continueReview")}
        </button>
      </div>
    </section>
  );
}

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
  // A direct link (?case=<local speaker>) opens that voice first when it is to decide.
  const [params, setParams] = useSearchParams();
  const linked = Number(params.get("case"));
  const linkedId = Number.isInteger(linked) && linked > 0 ? linked : undefined;
  const loadLinked = useCallback(
    () =>
      linkedId === undefined
        ? Promise.resolve({ ok: false as const, code: "none" })
        : fetchLocalSpeaker(linkedId),
    [linkedId],
  );
  const linkedState = useLoaded(loadLinked).state;
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
  const pinned =
    linkedState.status === "ready" ? toCase(linkedState.value) : undefined;
  const queue = useMemo(() => {
    const items = [
      ...(pinned ? [pinned] : []),
      ...(data?.items ?? []).filter(
        (item) => item.local_speaker_id !== pinned?.local_speaker_id,
      ),
    ].filter((item) => !gone.includes(item.local_speaker_id));
    const rank = (item: VerificationCase) =>
      skipped.indexOf(item.local_speaker_id);
    return [
      ...items.filter((item) => rank(item) < 0),
      ...items
        .filter((item) => rank(item) >= 0)
        .sort((a, b) => rank(a) - rank(b)),
    ];
  }, [data, pinned, skipped, gone]);
  const current = queue[0];

  function leaveLink() {
    setParams({}, { replace: true });
  }

  function choose(next: VerificationFilter) {
    setFilter(next);
    setNotice(undefined);
  }

  function settled(item: VerificationCase, message: string) {
    if (item.local_speaker_id === linkedId) leaveLink();
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
      {linkedId !== undefined && linkedState.status === "loading" && (
        <p className="text-mute">{t("people.verification.directLoading")}</p>
      )}
      {linkedId !== undefined &&
        (linkedState.status === "failed"
          ? linkedState.code === "local_speaker_not_found"
          : linkedState.status === "ready" && !pinned) && (
          <DirectLinkNotice
            read={
              linkedState.status === "ready" ? linkedState.value : undefined
            }
            onContinue={leaveLink}
          />
        )}
      {linkedId !== undefined &&
        linkedState.status === "failed" &&
        linkedState.code !== "local_speaker_not_found" && (
          <ErrorAlert>{failureText(linkedState.code)}</ErrorAlert>
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
