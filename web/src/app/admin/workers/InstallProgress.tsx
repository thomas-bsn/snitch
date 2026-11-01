import { useMemo, type ReactNode } from "react";
import { StatusPill } from "../../components/StatusPill";
import { useI18n } from "../../i18n";
import { useFormatDate } from "../format";
import type { WorkerDetail } from "./workersApi";
import { installCause, installStatus, transcriptionState } from "./workerView";

type Install = NonNullable<WorkerDetail["install"]>;
type Phase = NonNullable<Install["events"]>[number]["phase"];

const PHASES: readonly Phase[] = [
  "prerequisites",
  "registration",
  "image",
  "permissions",
  "gpu",
  "voice-models",
  "voice-test",
  "service",
];

function isPhase(value: string): value is Phase {
  return (PHASES as readonly string[]).includes(value);
}

/** Installation progress of a Worker: state, latest event, absence of news, transcription and voice, apart. */
export function InstallProgress({
  install,
  transcriptionConfig,
  createdAt,
}: {
  install: Install;
  transcriptionConfig: WorkerDetail["transcription_config"];
  createdAt: string;
}) {
  const { t } = useI18n();
  const formatDate = useFormatDate();
  const { tone, label } = installStatus(install);
  const last = install.last_event;
  const cause = installCause(install.cause_code);
  const transcription = transcriptionState(transcriptionConfig);
  const phaseName = (phase: string) =>
    isPhase(phase) ? t(`workers.install.phase.${phase}`) : phase;
  const newestFirst = useMemo(
    () => [...(install.events ?? [])].reverse(),
    [install.events],
  );
  const causeName = (code: string) =>
    t(`workers.install.cause.${installCause(code)}`);

  return (
    <div className="flex flex-col gap-3">
      <div className="flex flex-wrap items-center gap-2">
        <StatusPill tone={tone}>{t(label)}</StatusPill>
        <span className="text-mute">
          {t(`workers.install.stateHint.${install.state}`)}
        </span>
      </div>

      {install.state === "failed" && (
        <div
          role="alert"
          className="rounded-lg bg-bad-bg px-3 py-2.5 text-bad-fg"
        >
          <p className="font-medium">
            {t("workers.install.failedAt", {
              phase: phaseName(install.failed_phase),
              cause: causeName(install.cause_code),
            })}
          </p>
          <p>
            <strong>{t("workers.install.recovery_label")}.</strong>{" "}
            {t(`workers.install.recovery.${cause}`)}
          </p>
        </div>
      )}

      <dl className="grid grid-cols-[max-content_1fr] gap-x-6 gap-y-1.5">
        <Row label={t("workers.install.lastEvent")}>
          {last ? (
            <>
              {t("workers.install.lastEventValue", {
                phase: phaseName(last.phase),
                status: t(`workers.install.status.${last.status}`),
              })}
              {last.status === "failed" && last.cause_code && (
                <span className="ml-1.5 text-mute">
                  ({causeName(last.cause_code)})
                </span>
              )}
              <span className="ml-1.5 text-mute">
                {t("workers.install.occurredAt", {
                  date: formatDate(last.occurred_at),
                })}
              </span>
              {last.received_later && (
                <span className="ml-1.5 rounded-full bg-side px-2 py-0.5 text-xs text-mute">
                  {t("workers.install.receivedLater")}
                </span>
              )}
            </>
          ) : (
            <span className="text-mute">{t("workers.install.noEvent")}</span>
          )}
        </Row>
        <Row label={t("workers.install.news")}>
          {install.no_news
            ? t(
                install.last_activity_at
                  ? "workers.install.noNews"
                  : "workers.install.noNewsNever",
                {
                  date: formatDate(install.last_activity_at ?? createdAt),
                },
              )
            : t("workers.install.newsOk")}
        </Row>
        <Row label={t("workers.install.transcription")}>
          {t(`workers.install.transcriptionState.${transcription}`)}
          {transcriptionConfig?.error_code && (
            <span className="ml-1.5 text-mute">
              {t(
                `workers.install.transcriptionError.${transcriptionConfig.error_code}`,
              )}
            </span>
          )}
        </Row>
        <Row label={t("workers.install.voice")}>
          {install.voice.state === "unavailable"
            ? t("workers.install.voiceState.unavailable", {
                cause: causeName(install.voice.cause_code),
              })
            : t(`workers.install.voiceState.${install.voice.state}`)}
        </Row>
      </dl>

      {newestFirst.length > 0 && (
        <details>
          <summary className="cursor-pointer text-accent-ink">
            {t("workers.install.history")}
          </summary>
          <ul className="mt-2 flex flex-col gap-1">
            {newestFirst.map((event) => (
              <li
                key={`${event.received_at}|${event.occurred_at}|${event.phase}|${event.status}`}
                className="text-mute"
              >
                {formatDate(event.occurred_at)} · {phaseName(event.phase)} ·{" "}
                {t(`workers.install.status.${event.status}`)}
                {event.cause_code && ` · ${causeName(event.cause_code)}`}
                {event.received_later &&
                  ` · ${t("workers.install.receivedLater")}`}
              </li>
            ))}
          </ul>
        </details>
      )}
    </div>
  );
}

function Row({ label, children }: { label: string; children: ReactNode }) {
  return (
    <>
      <dt className="text-mute">{label}</dt>
      <dd className="m-0 break-words">{children}</dd>
    </>
  );
}
