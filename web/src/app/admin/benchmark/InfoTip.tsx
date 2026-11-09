import { useId } from "react";
import { Tooltip } from "@base-ui/react/tooltip";
import { useI18n } from "../../i18n";
import type { MessageKey } from "../../i18n/messages";

/**
 * A small "i" that explains a measure: a short definition on hover, on keyboard focus and on a tap. The
 * popup is portaled, so a scrolling table never clips it.
 */
export function InfoTip({ metric }: { metric: Metric }) {
  const { t } = useI18n();
  const name = t(`benchmark.metrics.${metric}.name` as MessageKey);
  const help = t(`benchmark.metrics.${metric}.help` as MessageKey);
  const descriptionId = useId();
  return (
    <Tooltip.Provider delay={100}>
      <Tooltip.Root>
        <Tooltip.Trigger
          type="button"
          aria-label={t("benchmark.results.infoLabel", { metric: name })}
          aria-describedby={descriptionId}
          className="ml-1.5 inline-flex size-[18px] shrink-0 cursor-help items-center justify-center rounded-full border border-line-strong bg-surface align-middle font-serif text-[11px] font-bold text-mute italic normal-case hover:border-accent hover:bg-accent-soft hover:text-accent-ink focus-visible:border-accent focus-visible:bg-accent-soft focus-visible:text-accent-ink"
        >
          i
        </Tooltip.Trigger>
        <Tooltip.Portal>
          <Tooltip.Positioner sideOffset={8} className="z-[70]">
            <Tooltip.Popup
              role="tooltip"
              className="max-w-72 rounded-lg bg-ink px-3 py-2 text-xs leading-snug font-normal tracking-normal text-canvas normal-case shadow-pop"
            >
              <strong className="block">{name}</strong>
              {help}
            </Tooltip.Popup>
          </Tooltip.Positioner>
        </Tooltip.Portal>
      </Tooltip.Root>
      {/* The definition is the description of the button for a screen reader, without opening the bubble; hidden, so it does not lengthen the name of a table header. */}
      <span id={descriptionId} hidden>
        {help}
      </span>
    </Tooltip.Provider>
  );
}

export type Metric =
  | "wer"
  | "cer"
  | "eer"
  | "thr"
  | "top1"
  | "precision"
  | "recall"
  | "fp"
  | "fn"
  | "coverage"
  | "elapsed";
