import type { components } from "../../../api.gen";
import { useI18n } from "../../i18n";
import type { MessageKey } from "../../i18n/messages";
import { Diagnosis } from "./AnalysisBoard";

type Event = components["schemas"]["AnalysisStepEvent"];

/**
 * The persisted events of the job a step followed, oldest first: what happened, when, on which
 * Worker and model, which attempt and cycle, and for a failure its catalogue cause with the advised
 * action. Nothing here is a raw error text or a path.
 */
export function StepTimeline({
  step,
  events,
}: {
  step: string;
  events: Event[];
}) {
  const { t, language } = useI18n();
  if (events.length === 0) return null;
  return (
    <ol
      data-timeline
      aria-label={t("queue.board.timeline.label", {
        step: t(`clip.analysis.step.${step}` as MessageKey),
      })}
      className="m-0 mt-1 flex list-none flex-col gap-1 border-l border-line p-0 pl-3"
    >
      {events.map((event) => {
        const worker =
          event.actor === "worker"
            ? event.worker_name || event.actor_id
            : undefined;
        const facts = [
          worker && t("queue.board.timeline.worker", { name: worker }),
          event.model &&
            t("queue.board.timeline.model", { model: event.model }),
          event.attempt != null &&
            t("queue.board.attempt", { n: event.attempt }),
          t("queue.board.timeline.cycle", { n: event.cycle }),
        ].filter(Boolean);
        return (
          <li
            key={event.id}
            data-timeline-event={event.event}
            className="text-xs"
          >
            <strong>{t(`queue.event.${event.event}`)}</strong>{" "}
            <time dateTime={event.created_at} className="text-mute">
              {new Date(event.created_at).toLocaleString(language)}
            </time>
            <span className="text-mute"> · {facts.join(" · ")}</span>
            {event.error_code && (
              <Diagnosis
                code={event.error_code}
                action={event.recommended_action}
                failed={false}
              />
            )}
          </li>
        );
      })}
    </ol>
  );
}
