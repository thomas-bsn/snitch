import { useCallback } from "react";
import { api, errorCode } from "../api/client";
import { useLoaded } from "../api/loaded";
import { ErrorAlert } from "../components/Alert";
import { useFormatDate } from "../admin/format";
import { useI18n } from "../i18n";
export function ObsoleteHistory({
  clipId,
  jobId,
}: {
  clipId: number;
  jobId?: number;
}) {
  const { t, error } = useI18n();
  const format = useFormatDate();
  const { state } = useLoaded(
    useCallback(async () => {
      try {
        const r = await api.GET("/api/clips/{id}/obsolete-corrections", {
          params: { path: { id: clipId } },
        });
        return r.data
          ? { ok: true as const, value: r.data }
          : { ok: false as const, code: errorCode(r.error) ?? "unknown" };
      } catch {
        return { ok: false as const, code: "unavailable" };
      }
    }, [clipId, jobId]),
  );
  if (state.status === "loading") return null;
  if (state.status === "failed")
    return <ErrorAlert>{error(state.code)}</ErrorAlert>;
  if (!state.value.items.length) return null;
  return (
    <details className="m-3 rounded-lg border border-line p-3">
      <summary className="cursor-pointer">
        {t("segmentEdit.obsolete", { count: state.value.items.length })}
      </summary>
      <p>{t("segmentEdit.obsoleteHint")}</p>
      <ul className="flex flex-col gap-2">
        {state.value.items.map((item) => (
          <li key={`${item.transcription_job_id}-${item.index}`}>
            <p>
              {t("segmentEdit.obsoleteSegment", {
                index: item.index + 1,
                date: format(item.obsolete_at),
              })}
            </p>
            <p>{t("segmentEdit.original", { text: item.original_text })}</p>
            {item.text && <p>{item.text}</p>}
            <ol>
              {item.history.map((event) => (
                <li key={event.id}>
                  {t("clip.history.by", {
                    author: event.author.username,
                    date: format(event.created_at),
                  })}{" "}
                  ·{" "}
                  {event.kind === "speaker"
                    ? `${event.old_speaker_label} → ${event.new_speaker_label}`
                    : `${event.old_text ?? ""} → ${event.new_text ?? ""}`}
                </li>
              ))}
            </ol>
          </li>
        ))}
      </ul>
    </details>
  );
}
