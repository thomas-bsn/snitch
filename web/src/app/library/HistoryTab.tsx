import { ChevronDown, ChevronRight } from "lucide-react";
import { useCallback, useId, useState } from "react";
import { useLoaded } from "../api/loaded";
import { useFormatDate } from "../admin/format";
import { ErrorAlert } from "../components/Alert";
import { useI18n } from "../i18n";
import type { MessageKey } from "../i18n/messages";
import { formatTimecode } from "./clipFormat";
import {
  fetchSegmentHistory,
  type SegmentHistory,
  type TranscriptSegment,
} from "./libraryApi";

const kinds: Record<string, MessageKey> = {
  text: "clip.history.kindText",
  speaker: "clip.history.kindSpeaker",
  restore: "clip.history.kindRestore",
};

function Entries({ history }: { history: SegmentHistory }) {
  const { t } = useI18n();
  const formatDate = useFormatDate();
  return (
    <div className="flex flex-col gap-2">
      <p className="m-0 text-xs text-mute">
        {t("clip.history.original", { text: history.original_text })}
      </p>
      {history.items.length === 0 ? (
        <p className="m-0 text-mute">{t("clip.history.none")}</p>
      ) : (
        <ol className="m-0 flex list-none flex-col gap-2 p-0">
          {history.items.map((entry) => (
            <li
              key={entry.id}
              className="rounded-md border border-line p-2.5 text-sm"
            >
              <p className="m-0 font-medium">
                {t(kinds[entry.kind] ?? "clip.history.kindText")}
              </p>
              <p className="m-0 text-xs text-mute">
                {t("clip.history.by", {
                  author: entry.author.username,
                  date: formatDate(entry.created_at),
                })}
              </p>
              {entry.reason && (
                <p className="m-0 text-xs text-mute">
                  {t("clip.history.reason", { reason: entry.reason })}
                </p>
              )}
              {entry.kind !== "speaker" && (
                <dl className="m-0 mt-1.5 grid grid-cols-[auto_1fr] gap-x-2 gap-y-0.5">
                  <dt className="text-xs text-mute">
                    {t("clip.history.before")}
                  </dt>
                  <dd className="m-0">{entry.old_text}</dd>
                  <dt className="text-xs text-mute">
                    {t("clip.history.after")}
                  </dt>
                  <dd className="m-0">{entry.new_text}</dd>
                </dl>
              )}
              {entry.kind === "speaker" && (
                <p className="m-0 mt-1.5">
                  {entry.old_speaker_label} → {entry.new_speaker_label}
                </p>
              )}
            </li>
          ))}
        </ol>
      )}
    </div>
  );
}

function LoadedHistory({ clipId, index }: { clipId: number; index: number }) {
  const { t, error } = useI18n();
  const { state } = useLoaded(
    useCallback(() => fetchSegmentHistory(clipId, index), [clipId, index]),
  );
  if (state.status === "loading") {
    return (
      <p className="m-0 text-mute" role="status">
        {t("clip.history.loading")}
      </p>
    );
  }
  if (state.status === "failed") {
    return <ErrorAlert>{error(state.code)}</ErrorAlert>;
  }
  return <Entries history={state.value} />;
}

function CorrectedSegment({
  clipId,
  segment,
}: {
  clipId: number;
  segment: TranscriptSegment;
}) {
  const { t } = useI18n();
  const [open, setOpen] = useState(false);
  const panel = useId();
  const Chevron = open ? ChevronDown : ChevronRight;
  return (
    <li className="rounded-lg border border-line p-3">
      <div className="flex items-start gap-3">
        <div className="min-w-0 flex-1">
          <p className="m-0 text-xs font-medium text-mute">
            {t("clip.history.segment", { time: formatTimecode(segment.start) })}
          </p>
          <p className="m-0 mt-0.5">{segment.text}</p>
        </div>
        <button
          type="button"
          aria-expanded={open}
          aria-controls={panel}
          onClick={() => setOpen(!open)}
          className="inline-flex min-h-11 shrink-0 items-center gap-1 rounded-md px-2 text-sm font-medium text-accent-ink md:min-h-8"
        >
          <Chevron className="size-4" aria-hidden="true" />
          {open ? t("clip.history.hide") : t("clip.history.show")}
          <span className="sr-only">
            {" "}
            (
            {t("clip.history.segment", { time: formatTimecode(segment.start) })}
            )
          </span>
        </button>
      </div>
      <div id={panel} hidden={!open} className="mt-3">
        {open && <LoadedHistory clipId={clipId} index={segment.index ?? 0} />}
      </div>
    </li>
  );
}

/** The history of the clip, read only: the corrected segments, each with its changes. */
export function HistoryTab({
  clipId,
  segments,
}: {
  clipId: number;
  segments: TranscriptSegment[];
}) {
  const { t } = useI18n();
  const corrected = segments.filter(
    (segment) =>
      segment.index !== undefined &&
      (segment.corrected || segment.local_speaker_corrected),
  );
  if (corrected.length === 0) {
    return <p className="p-4 text-mute">{t("clip.history.empty")}</p>;
  }
  return (
    <ul
      aria-label={t("clip.history.listLabel")}
      className="m-0 flex list-none flex-col gap-2 p-3"
    >
      {corrected.map((segment) => (
        <CorrectedSegment
          key={segment.index}
          clipId={clipId}
          segment={segment}
        />
      ))}
    </ul>
  );
}
