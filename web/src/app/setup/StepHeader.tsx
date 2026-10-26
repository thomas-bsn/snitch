import { useEffect, useRef } from "react";
import { useI18n } from "../i18n";

/**
 * The heading of a wizard step. It takes the focus when the step appears after the first
 * one, so that keyboard and screen reader users land on the new step.
 */
export function StepHeader({
  position,
  total,
  title,
  lead,
  focus = true,
}: {
  position: number;
  total: number;
  title: string;
  lead?: string;
  focus?: boolean;
}) {
  const { t } = useI18n();
  const heading = useRef<HTMLHeadingElement>(null);
  useEffect(() => {
    if (focus) heading.current?.focus();
  }, [focus]);
  return (
    <div className="flex flex-col gap-1.5">
      <p className="text-xs font-semibold tracking-wide text-accent-ink uppercase">
        {t("setup.stepOf", { step: position, total })}
      </p>
      <h1
        ref={heading}
        tabIndex={-1}
        className="text-[22px] font-semibold outline-none"
      >
        {title}
      </h1>
      {lead && <p className="max-w-xl text-mute">{lead}</p>}
    </div>
  );
}
