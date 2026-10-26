import { Check, Minus } from "lucide-react";
import { Brand } from "../components/Logo";
import { useI18n } from "../i18n";
import { LanguageSelect } from "../screens/AuthLayout";
import type { SetupStepState } from "./setupApi";
import { STEP_IDS, STEP_LABELS, type StepId } from "./steps";

/** How a step stands, for the list on the side. */
export type Standing = "done" | "skipped" | "current" | "todo";

/**
 * The side of the wizard: what it is for, the seven steps with where each one stands
 * (a word and a symbol, never the color alone) and the language choice.
 */
export function WizardSidebar({
  standing,
  optional,
  available,
  onSelect,
}: {
  standing: Record<StepId, Standing>;
  optional: ReadonlySet<StepId>;
  available: ReadonlySet<StepId>;
  onSelect: (id: StepId) => void;
}) {
  const { t } = useI18n();
  return (
    <aside className="flex flex-col gap-6 border-b border-line bg-side px-5 py-5 md:min-h-dvh md:w-[300px] md:shrink-0 md:border-r md:border-b-0 md:px-7 md:py-8">
      <Brand />
      <div className="flex flex-col gap-1.5">
        <p className="text-lg font-semibold">{t("setup.title")}</p>
        <p className="text-mute">{t("setup.lead")}</p>
      </div>
      <ol
        aria-label={t("setup.stepsLabel")}
        className="m-0 hidden list-none flex-col gap-1 p-0 md:flex"
      >
        {STEP_IDS.map((id, index) => {
          const state = standing[id];
          return (
            <li
              key={id}
              aria-current={state === "current" ? "step" : undefined}
              className={`flex items-center gap-3 rounded-md px-2 py-2 ${
                state === "current"
                  ? "bg-accent-soft font-semibold text-accent-ink"
                  : "text-nav"
              }`}
            >
              <span
                aria-hidden="true"
                className={`flex size-6 shrink-0 items-center justify-center rounded-full border text-xs font-semibold ${
                  state === "done"
                    ? "border-ok-fg bg-ok-bg text-ok-fg"
                    : state === "current"
                      ? "border-accent-solid bg-accent-solid text-accent-fg"
                      : "border-line-strong bg-surface text-mute"
                }`}
              >
                {state === "done" ? (
                  <Check className="size-3.5" />
                ) : state === "skipped" ? (
                  <Minus className="size-3.5" />
                ) : (
                  index + 1
                )}
              </span>
              {available.has(id) && state !== "current" ? (
                <button
                  type="button"
                  className="flex-1 text-left underline-offset-2 hover:underline focus-visible:outline-2 focus-visible:outline-accent-solid"
                  onClick={() => onSelect(id)}
                >
                  {t(STEP_LABELS[id])}
                </button>
              ) : (
                <span className="flex-1">{t(STEP_LABELS[id])}</span>
              )}
              <StandingTag state={state} optional={optional.has(id)} />
            </li>
          );
        })}
      </ol>
      <div className="md:mt-auto">
        <LanguageSelect />
      </div>
    </aside>
  );
}

function StandingTag({
  state,
  optional,
}: {
  state: Standing;
  optional: boolean;
}) {
  const { t } = useI18n();
  const label =
    state === "done"
      ? t("setup.tags.done")
      : state === "skipped"
        ? t("setup.tags.skipped")
        : optional
          ? t("setup.tags.optional")
          : null;
  if (!label) return null;
  return <span className="text-xs font-normal text-mute">{label}</span>;
}

export function standingOf(state: SetupStepState): Standing {
  return state === "todo" ? "todo" : state;
}
