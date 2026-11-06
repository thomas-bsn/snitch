import { useEffect, useState } from "react";
import type { components } from "../../api.gen";
import { api } from "../api/client";
import { useFormatDate } from "../admin/format";
import { useI18n } from "../i18n";
import { en } from "../i18n/en";
import type { MessageKey } from "../i18n/messages";

const known = en.clip.analysis.failure;

type JobEvent = components["schemas"]["JobEvent"];

/**
 * What an Administrator needs to act on a failed step: the step, when it failed, the cause
 * and the advised action. Everything comes from the recorded job history (the server decides
 * the cause and the action); the browser only translates the stable codes.
 */
export function StepFailure({
  jobId,
  step,
  version,
}: {
  jobId: number;
  step: string;
  /** Changes when the step does, so the detail follows a retry. */
  version: string;
}) {
  const { t } = useI18n();
  const formatDate = useFormatDate();
  const [event, setEvent] = useState<JobEvent | null>(null);

  useEffect(() => {
    let current = true;
    void api
      .GET("/api/admin/jobs/{id}/history", { params: { path: { id: jobId } } })
      .then((r) => {
        if (!current) return;
        // Only the current attempt cycle counts: a relaunch opens a new cycle, and its own
        // "admin_relaunched" event repeats the previous error, which is not a new failure.
        const items = r.data?.items ?? [];
        const cycle = items.reduce((max, e) => Math.max(max, e.cycle), 0);
        const failures = items.filter(
          (e) =>
            e.error_code && e.cycle === cycle && e.event !== "admin_relaunched",
        );
        setEvent(failures.length ? failures[failures.length - 1] : null);
      })
      .catch(() => current && setEvent(null));
    return () => {
      current = false;
    };
  }, [jobId, version]);

  if (!event) return null;
  const code = event.error_code ?? "";
  const action = event.recommended_action ?? "";
  return (
    <div
      role="group"
      aria-label={t("clip.analysis.failure.label")}
      className="mt-1 rounded-lg border border-line p-2 text-xs"
    >
      <dl className="m-0 grid grid-cols-[auto_1fr] gap-x-3 gap-y-1">
        <dt className="text-mute">{t("clip.analysis.failure.step")}</dt>
        <dd className="m-0">{step}</dd>
        <dt className="text-mute">{t("clip.analysis.failure.when")}</dt>
        <dd className="m-0">
          <time dateTime={event.created_at}>
            {formatDate(event.created_at)}
          </time>
        </dd>
        <dt className="text-mute">{t("clip.analysis.failure.cause")}</dt>
        <dd className="m-0">
          {Object.hasOwn(known.causes, code)
            ? t(`clip.analysis.failure.causes.${code}` as MessageKey)
            : t("clip.analysis.failure.unknownCause")}
        </dd>
        {action && (
          <>
            <dt className="text-mute">{t("clip.analysis.failure.action")}</dt>
            <dd className="m-0">
              {Object.hasOwn(known.actions, action)
                ? t(`clip.analysis.failure.actions.${action}` as MessageKey)
                : t("clip.analysis.failure.unknownAction")}
            </dd>
          </>
        )}
      </dl>
    </div>
  );
}
