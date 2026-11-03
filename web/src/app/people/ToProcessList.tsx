import { useCallback, useEffect, useState } from "react";
import { Link } from "react-router";
import { useLoaded } from "../api/loaded";
import { buttonClass } from "../components/buttons";
import { ErrorAlert } from "../components/Alert";
import { useI18n } from "../i18n";
import { useFailureText } from "./failureText";
import { fetchToProcess, type ToProcessFilter } from "./peopleApi";

const PAGE_SIZE = 100;

const filterClass = (pressed: boolean) =>
  `inline-flex min-h-11 items-center gap-2 rounded-lg border px-3 text-sm font-medium md:min-h-9 ${
    pressed
      ? "border-accent bg-accent-soft text-accent-ink"
      : "border-line-strong bg-surface text-ink hover:bg-side"
  }`;

/**
 * The "To process" list: one row per clip that still has a voice to settle, with what is
 * left in it. Numbers and filters come from the server. `version` changes when the caller
 * settled something, so the list is read again; `onToDecide` receives the number of voices
 * to decide (unknown + ambiguous) for the counter of the tab.
 */
export function ToProcessList({
  version,
  onToDecide,
}: {
  version: number;
  onToDecide?: (count: number) => void;
}) {
  const { t } = useI18n();
  const failureText = useFailureText();
  const [filter, setFilter] = useState<ToProcessFilter>("all");
  const load = useCallback(() => {
    void version;
    return fetchToProcess(filter, 1, PAGE_SIZE);
  }, [filter, version]);
  const { state } = useLoaded(load);
  const data = state.status === "ready" ? state.value : undefined;
  const totals = data?.totals;
  const toDecide = totals ? totals.unknown + totals.ambiguous : undefined;
  useEffect(() => {
    if (toDecide !== undefined) onToDecide?.(toDecide);
  }, [toDecide, onToDecide]);

  const filters: { value: ToProcessFilter; label: string; count?: number }[] = [
    {
      value: "all",
      label: t("people.verification.byClip.all"),
      count: totals?.clips,
    },
    {
      value: "to_confirm",
      label: t("people.verification.byClip.toConfirm"),
      count: totals?.to_confirm,
    },
    {
      value: "unknown_ambiguous",
      label: t("people.verification.byClip.unknownAmbiguous"),
      count: totals ? totals.unknown + totals.ambiguous : undefined,
    },
    {
      value: "group",
      label: t("people.verification.byClip.withGroup"),
      count: totals?.groups,
    },
  ];

  return (
    <section
      aria-label={t("people.verification.byClip.label")}
      className="flex flex-col gap-3"
    >
      <p className="max-w-2xl text-mute">
        {t("people.verification.byClip.lead")}
      </p>
      <div
        role="group"
        aria-label={t("people.verification.byClip.filtersLabel")}
        className="flex flex-wrap gap-2"
      >
        {filters.map((item) => (
          <button
            key={item.value}
            type="button"
            aria-pressed={filter === item.value}
            onClick={() => setFilter(item.value)}
            className={filterClass(filter === item.value)}
          >
            {item.label}
            {item.count !== undefined && <b>{item.count}</b>}
          </button>
        ))}
      </div>
      {state.status === "failed" && (
        <ErrorAlert>{failureText(state.code)}</ErrorAlert>
      )}
      {data && data.items.length === 0 && (
        <p className="rounded-xl border border-dashed border-line-strong p-4 text-center text-mute">
          {t("people.verification.byClip.empty")}
        </p>
      )}
      {data && data.items.length > 0 && (
        <ul className="m-0 flex list-none flex-col gap-2 p-0">
          {data.items.map((clip) => (
            <li
              key={clip.clip_id}
              className="flex flex-wrap items-center gap-x-4 gap-y-2 rounded-xl border border-line p-3"
            >
              <b className="min-w-0 flex-1 basis-48 break-words">
                {clip.clip_name}
              </b>
              <span className="flex flex-wrap gap-x-3 text-sm text-mute">
                <span>
                  {t("people.verification.byClip.toConfirmCount", {
                    count: clip.to_confirm,
                  })}
                </span>
                <span>
                  {t("people.verification.byClip.ambiguousCount", {
                    count: clip.ambiguous,
                  })}
                </span>
                <span>
                  {t("people.verification.byClip.unknownCount", {
                    count: clip.unknown,
                  })}
                </span>
                <span>
                  {t("people.verification.byClip.groupsCount", {
                    count: clip.groups,
                  })}
                </span>
              </span>
              <Link
                to={`/people/verification?clip=${clip.clip_id}`}
                aria-label={t("people.verification.byClip.processLabel", {
                  clip: clip.clip_name,
                })}
                className={buttonClass("secondary")}
              >
                {t("people.verification.byClip.process")}
              </Link>
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}
