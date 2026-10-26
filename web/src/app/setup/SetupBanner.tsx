import { X } from "lucide-react";
import { useEffect, useState } from "react";
import { Link } from "react-router";
import { buttonClass } from "../components/buttons";
import { useI18n } from "../i18n";
import { isAdministrator } from "../roles";
import { useSession } from "../session";
import {
  fetchSetupProgress,
  hasRemainingSteps,
  useSetupActions,
  type SetupProgress,
} from "./setupApi";

/**
 * "Finish the setup": shown to administrators while optional steps of the installation are
 * still to do, until they close it for good. The server keeps the progress and the closing.
 */
export function SetupBanner() {
  const { t } = useI18n();
  const { user } = useSession();
  const { dismiss } = useSetupActions();
  const [progress, setProgress] = useState<SetupProgress>();
  const administrator = user !== null && isAdministrator(user);

  useEffect(() => {
    if (!administrator) return;
    let current = true;
    void fetchSetupProgress().then((result) => {
      if (current && result.ok) setProgress(result.value);
    });
    return () => {
      current = false;
    };
  }, [administrator]);

  if (!progress || progress.dismissed || !hasRemainingSteps(progress)) {
    return null;
  }
  const left = progress.steps.filter((step) => step.state !== "done").length;

  return (
    <section
      aria-label={t("setup.banner.label")}
      className="mx-4 mt-4 flex flex-wrap items-center gap-x-4 gap-y-2 rounded-lg bg-accent-soft px-4 py-3 text-accent-ink md:mx-7"
    >
      <div className="min-w-0 flex-1">
        <p className="font-semibold">{t("setup.banner.title")}</p>
        <p>{t("setup.banner.body", { count: left })}</p>
      </div>
      <Link to="/setup" className={buttonClass("primary")}>
        {t("setup.banner.resume")}
      </Link>
      <button
        type="button"
        aria-label={t("setup.banner.dismiss")}
        title={t("setup.banner.dismiss")}
        onClick={() =>
          void dismiss().then((result) => {
            if (result.ok) setProgress(result.value);
          })
        }
        className="flex size-11 shrink-0 items-center justify-center rounded-md hover:bg-surface md:size-8"
      >
        <X className="size-4" />
      </button>
    </section>
  );
}
