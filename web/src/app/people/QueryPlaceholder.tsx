import { useI18n } from "../i18n";

/**
 * Stands in for a screen not built yet (case processing, group naming) when its address carries
 * a query: the address keeps its target visible instead of silently ignoring it.
 */
export function QueryPlaceholder({
  kind,
  value,
}: {
  kind: "case" | "voice";
  value: string;
}) {
  const { t } = useI18n();
  const title =
    kind === "case"
      ? t("people.placeholder.caseTitle", { case: value })
      : t("people.placeholder.voiceTitle", { voice: value });
  const lead =
    kind === "case"
      ? t("people.placeholder.caseLead")
      : t("people.placeholder.voiceLead");
  return (
    <section
      data-testid={`people-${kind}-placeholder`}
      className="flex flex-col gap-1 rounded-xl border border-dashed border-line-strong bg-surface p-4"
    >
      <b>{title}</b>
      <p className="m-0 text-mute">{lead}</p>
    </section>
  );
}
