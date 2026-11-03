import { ArrowRight } from "lucide-react";
import { useCallback, useEffect, useState, type ReactNode } from "react";
import { Link } from "react-router";
import { useLoaded } from "../api/loaded";
import { buttonClass } from "../components/buttons";
import { ErrorAlert } from "../components/Alert";
import { useI18n } from "../i18n";
import { useFailureText } from "./failureText";
import {
  publishToProcessCount,
  toProcessCount,
} from "../shell/useToProcessCount";
import { fetchToProcess, type ToProcessFilter } from "./peopleApi";

const PAGE_SIZE = 100;

const filterClass = (pressed: boolean) =>
  `inline-flex min-h-11 items-center rounded-full border px-3 py-1 text-sm md:min-h-0 ${
    pressed
      ? "border-accent bg-accent-soft text-accent-ink"
      : "border-line text-mute hover:text-ink"
  }`;

type Tone = "accent" | "warn" | "idle";

function CountBadge({ tone, children }: { tone: Tone; children: ReactNode }) {
  const c = {
    accent: "bg-accent-soft text-accent-ink",
    warn: "bg-warn-bg text-warn-fg",
    idle: "bg-side text-mute",
  }[tone];
  return (
    <span
      className={`inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium ${c}`}
    >
      {children}
    </span>
  );
}

/**
 * The "To process" list: one row per clip that still has a voice to settle, with what is
 * left in it. Numbers and filters come from the server; the number of voices to settle goes
 * to the counters of the tab and of the sidebar.
 */
export function ToProcessList() {
  const { t } = useI18n();
  const failureText = useFailureText();
  const [filter, setFilter] = useState<ToProcessFilter>("all");
  const load = useCallback(
    () => fetchToProcess(filter, 1, PAGE_SIZE),
    [filter],
  );
  const { state } = useLoaded(load);
  const data = state.status === "ready" ? state.value : undefined;
  const totals = data?.totals;
  const toSettle = totals ? toProcessCount(totals) : undefined;
  useEffect(() => {
    if (toSettle !== undefined) publishToProcessCount(toSettle);
  }, [toSettle]);

  const filters: { value: ToProcessFilter; label: string }[] = [
    {
      value: "all",
      label: t("people.verification.byClip.all"),
    },
    {
      value: "to_confirm",
      label: t("people.verification.byClip.toConfirm"),
    },
    {
      value: "unknown_ambiguous",
      label: t("people.verification.byClip.unknownAmbiguous"),
    },
    {
      value: "group",
      label: t("people.verification.byClip.withGroup"),
    },
  ];

  return (
    <section
      aria-label={t("people.verification.byClip.label")}
      className="flex flex-col gap-3"
    >
      <div
        role="group"
        aria-label={t("people.verification.byClip.filtersLabel")}
        className="flex flex-wrap items-center gap-2"
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
          </button>
        ))}
        {totals && (
          <span className="ml-auto text-sm text-mute">
            {t("people.verification.byClip.summary", {
              voices: toSettle ?? 0,
              clips: totals.clips,
            })}
          </span>
        )}
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
        <ul className="m-0 flex list-none flex-col rounded-xl border border-line p-0">
          {data.items.map((clip) => (
            <li
              key={clip.clip_id}
              className="flex flex-wrap items-center gap-3 border-b border-line px-4 py-3 last:border-b-0"
            >
              <span className="min-w-0 flex-1 basis-48 font-medium break-words">
                {clip.clip_name}
              </span>
              <span className="flex flex-wrap gap-1.5">
                {clip.to_confirm > 0 && (
                  <CountBadge tone="accent">
                    {t("people.verification.byClip.toConfirmCount", {
                      count: clip.to_confirm,
                    })}
                  </CountBadge>
                )}
                {clip.ambiguous > 0 && (
                  <CountBadge tone="warn">
                    {t("people.verification.byClip.ambiguousCount", {
                      count: clip.ambiguous,
                    })}
                  </CountBadge>
                )}
                {clip.unknown > 0 && (
                  <CountBadge tone="idle">
                    {t("people.verification.byClip.unknownCount", {
                      count: clip.unknown,
                    })}
                  </CountBadge>
                )}
                {clip.groups > 0 && (
                  <CountBadge tone="idle">
                    {t("people.verification.byClip.groupsCount", {
                      count: clip.groups,
                    })}
                  </CountBadge>
                )}
              </span>
              <Link
                to={`/people/verification?clip=${clip.clip_id}`}
                className={buttonClass("primary")}
              >
                <span aria-hidden="true">
                  {t("people.verification.byClip.process")}{" "}
                  {clip.to_confirm + clip.ambiguous + clip.unknown}
                </span>
                <span className="sr-only">
                  {t("people.verification.byClip.processLabel", {
                    clip: clip.clip_name,
                  })}
                </span>
                <ArrowRight className="size-4" aria-hidden="true" />
              </Link>
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}
