import { WorkerCreateForm } from "../../admin/workers/WorkerCreateForm";
import { useI18n } from "../../i18n";
import { StepHeader } from "../StepHeader";

/**
 * Step 5 (optional): the first Worker, with the same form as the administration. Creating it
 * fixes the library's model and language (after the confirmation the form asks for) and
 * hands over the deployment files; there is no separate transcription step.
 */
export function WorkerStep({
  onDone,
  skip,
}: {
  onDone: () => Promise<void>;
  skip: React.ReactNode;
}) {
  const { t } = useI18n();
  return (
    <div className="flex flex-col gap-5">
      <StepHeader
        position={6}
        total={9}
        title={t("setup.worker.title")}
        lead={t("setup.worker.lead")}
      />
      <WorkerCreateForm onDone={() => void onDone()} secondaryAction={skip} />
    </div>
  );
}
