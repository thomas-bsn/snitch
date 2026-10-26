import { Link, useNavigate } from "react-router";
import { buttonClass } from "../../components/buttons";
import { StatusPill } from "../../components/StatusPill";
import { useI18n } from "../../i18n";
import { stepState, type SetupProgress } from "../setupApi";
import { StepHeader } from "../StepHeader";
import { OPTIONAL_STEPS, STEP_LABELS } from "../steps";

/**
 * Step 7: what is configured and, for what was skipped, where to finish it. It reads the
 * progress the server reports, which follows what is really configured.
 */
export function SummaryStep({ progress }: { progress: SetupProgress }) {
  const { t } = useI18n();
  const navigate = useNavigate();
  const remaining = progress.steps.filter((step) => step.state !== "done");
  return (
    <div className="flex max-w-xl flex-col gap-5">
      <StepHeader
        position={7}
        total={7}
        title={t("setup.summary.title")}
        lead={
          remaining.length === 0
            ? t("setup.summary.leadDone")
            : t("setup.summary.lead")
        }
      />
      <table className="w-full border-collapse overflow-hidden rounded-xl border border-line text-left">
        <caption className="sr-only">{t("setup.summary.caption")}</caption>
        <tbody>
          <tr>
            <th scope="row" className="px-3 py-2.5 font-medium">
              {t(STEP_LABELS.admin)}
            </th>
            <td className="px-3 py-2.5">
              <StatusPill tone="ok">{t("setup.tags.done")}</StatusPill>
            </td>
          </tr>
          {OPTIONAL_STEPS.map((step) => {
            const state = stepState(progress, step.server);
            return (
              <tr key={step.id} className="border-t border-line">
                <th scope="row" className="px-3 py-2.5 font-medium">
                  {t(STEP_LABELS[step.id])}
                </th>
                <td className="px-3 py-2.5">
                  {state === "done" ? (
                    <StatusPill tone="ok">{t("setup.tags.done")}</StatusPill>
                  ) : (
                    <span className="flex flex-wrap items-center gap-x-3 gap-y-1">
                      <StatusPill tone="idle">
                        {state === "skipped"
                          ? t("setup.tags.skipped")
                          : t("setup.tags.todo")}
                      </StatusPill>
                      <Link
                        to={step.where}
                        className="inline-flex min-h-11 items-center text-accent-ink underline-offset-2 hover:underline md:min-h-0"
                      >
                        {t("setup.summary.finishIn", {
                          where: t(step.whereLabel),
                        })}
                      </Link>
                    </span>
                  )}
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
      {remaining.length > 0 && (
        <p className="text-mute">{t("setup.summary.banner")}</p>
      )}
      <div>
        <button
          type="button"
          onClick={() => navigate("/library")}
          className={buttonClass("primary")}
        >
          {t("setup.summary.open")}
        </button>
      </div>
    </div>
  );
}
