import { SemanticSettings } from "../../admin/semantic/SemanticSettings";
import { useI18n } from "../../i18n";
import { StepHeader } from "../StepHeader";

export function SemanticStep({
  onDone,
  skip,
}: {
  onDone: () => Promise<void>;
  skip: React.ReactNode;
}) {
  const { t } = useI18n();
  return (
    <div className="flex max-w-xl flex-col gap-5">
      <StepHeader
        position={4}
        total={7}
        title={t("setup.semantic.title")}
        lead={t("setup.semantic.lead")}
      />
      <SemanticSettings onSaved={onDone} secondaryAction={skip} />
    </div>
  );
}
