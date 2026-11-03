import { useCallback, useState } from "react";
import type { components } from "../../api.gen";
import { useEventStream } from "../api/useEventStream";
import { useI18n } from "../i18n";

type JobLogs = components["schemas"]["JobLogs"];

/**
 * The last lines of a failed job, for an Administrator: pushed by the server as they arrive
 * (at most 50 lines, already masked by the Worker).
 */
export function JobLogsPanel({ jobId }: { jobId: number }) {
  const { t } = useI18n();
  const [logs, setLogs] = useState<JobLogs | null>(null);
  const status = useEventStream<JobLogs>(
    `/api/admin/jobs/${jobId}/logs/events`,
    useCallback((snapshot: JobLogs) => setLogs(snapshot), []),
    // The stream is the only source: there is no JSON read of the lines to fall back on.
    useCallback(() => undefined, []),
  );
  const lines = logs?.lines ?? [];
  return (
    <section
      aria-label={t("clip.analysis.logs.title")}
      className="mt-1 rounded-lg border border-line p-2"
    >
      <h3 className="m-0 text-xs font-medium">
        {t("clip.analysis.logs.title")}
      </h3>
      {lines.length > 0 ? (
        <ol className="m-0 mt-1 flex max-h-60 list-none flex-col gap-0.5 overflow-y-auto p-0 font-mono text-[11px] leading-relaxed">
          {lines.map((line) => (
            <li key={line.seq} className="break-all">
              {line.text}
            </li>
          ))}
        </ol>
      ) : (
        <p className="m-0 mt-1 text-xs text-mute">
          {t("clip.analysis.logs.empty")}
        </p>
      )}
      {status !== "live" && status !== "connecting" && (
        <p className="m-0 mt-1 text-xs text-mute">
          {t("clip.analysis.logs.liveOff")}
        </p>
      )}
    </section>
  );
}
