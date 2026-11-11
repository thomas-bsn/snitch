import { LoaderCircle } from "lucide-react";
import { useCallback, useEffect, useRef, useState } from "react";
import type { components } from "../../api.gen";
import { api } from "../api/client";
import { useEventStream } from "../api/useEventStream";
import { useI18n } from "../i18n";

export type VoiceMatchingStatus = components["schemas"]["VoiceMatchingStatus"];

/**
 * Where voice matching stands (#392): a validation answers at once and matching follows in the background, so
 * the suggestions shown may be about to change. Live through the server-sent stream, polled while it is down.
 * `onFinished` runs when a pass ends after the hook mounted, for a screen to read its suggestions again.
 */
export function useVoiceMatchingStatus(onFinished?: () => void) {
  const [status, setStatus] = useState<VoiceMatchingStatus>();
  const finished = useRef<string | null | undefined>(undefined);
  const callback = useRef(onFinished);
  useEffect(() => {
    callback.current = onFinished;
  });
  const accept = useCallback((next: VoiceMatchingStatus) => {
    setStatus(next);
    const previous = finished.current;
    finished.current = next.last_finished_at;
    if (previous !== undefined && next.last_finished_at !== previous)
      callback.current?.();
  }, []);
  useEventStream<VoiceMatchingStatus>(
    "/api/voice-matching/status/events",
    accept,
    () =>
      void api
        .GET("/api/voice-matching/status")
        .then((r) => r.data && accept(r.data))
        .catch(() => undefined),
    10_000,
  );
  return status;
}

function useClock() {
  const { language } = useI18n();
  return useCallback(
    (iso: string) => {
      const at = new Date(iso);
      const today = new Date().toDateString() === at.toDateString();
      return new Intl.DateTimeFormat(
        language,
        today
          ? { timeStyle: "short" }
          : { dateStyle: "medium", timeStyle: "short" },
      ).format(at);
    },
    [language],
  );
}

/** One line saying whether matching is in progress, and when the last pass ended. */
export function MatchingStatusLine({
  status,
  className = "",
}: {
  status: VoiceMatchingStatus | undefined;
  className?: string;
}) {
  const { t } = useI18n();
  const clock = useClock();
  if (!status) return null;
  const inProgress = status.pending || status.running;
  let text: string;
  if (inProgress && status.last_failed_at)
    text = t("matchingStatus.failed", { time: clock(status.last_failed_at) });
  else if (inProgress) text = t("matchingStatus.inProgress");
  else if (status.last_finished_at)
    text = t("matchingStatus.upToDate", {
      time: clock(status.last_finished_at),
    });
  else text = t("matchingStatus.never");
  return (
    <p
      aria-live="polite"
      title={t("matchingStatus.label")}
      data-testid="matching-status"
      data-state={inProgress ? "in_progress" : "idle"}
      className={`m-0 flex items-center gap-1.5 text-[13px] text-mute ${className}`}
    >
      {inProgress && (
        <LoaderCircle aria-hidden size={14} className="animate-spin" />
      )}
      <span>{text}</span>
    </p>
  );
}

/** The live status line, for a screen that only shows it. */
export function MatchingStatus({ className }: { className?: string }) {
  return (
    <MatchingStatusLine
      status={useVoiceMatchingStatus()}
      className={className}
    />
  );
}
