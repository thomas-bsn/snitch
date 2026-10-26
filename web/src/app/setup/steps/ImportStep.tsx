import { Link } from "react-router";
import { buttonClass } from "../../components/buttons";
import { useI18n } from "../../i18n";
import { StepHeader } from "../StepHeader";

/**
 * Step 6 (optional): importing videos. The wizard only points to the Import page: nothing is
 * uploaded, scanned or processed unless the person does it there.
 */
export function ImportStep({
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
        position={6}
        total={7}
        title={t("setup.import.title")}
        lead={t("setup.import.lead")}
      />
      <p className="rounded-lg border border-line bg-side p-4 text-mute">
        {t("setup.import.nothing")}
      </p>
      <div>
        <Link to="/import" className={buttonClass()}>
          {t("setup.import.open")}
        </Link>
      </div>
      <div className="flex flex-wrap gap-2">
        {skip}
        <button
          type="button"
          onClick={() => void onDone()}
          className={buttonClass("primary")}
        >
          {t("setup.continue")}
        </button>
      </div>
    </div>
  );
}
