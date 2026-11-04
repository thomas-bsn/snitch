import { useI18n } from "../../i18n";

/**
 * How to use the one-command installation: where to run it (WSL2 or Linux), what must be ready,
 * and what to do when the invitation no longer works. The command itself is shown by the caller.
 */
export function InstallGuide() {
  const { t } = useI18n();
  return (
    <div className="flex flex-col gap-3">
      <h4 className="font-medium">{t("workers.guide.whereTitle")}</h4>
      <ul className="m-0 flex list-disc flex-col gap-1 pl-5">
        <li>{t("workers.guide.wsl")}</li>
        <li>{t("workers.guide.linux")}</li>
      </ul>
      <h4 className="font-medium">{t("workers.guide.prerequisitesTitle")}</h4>
      <ul className="m-0 flex list-disc flex-col gap-1 pl-5">
        <li>{t("workers.guide.docker")}</li>
        <li>{t("workers.guide.gpu")}</li>
        <li>{t("workers.guide.network")}</li>
      </ul>
      <p className="text-mute">{t("workers.guide.models")}</p>
      <p className="text-mute">{t("workers.guide.expired")}</p>
    </div>
  );
}
