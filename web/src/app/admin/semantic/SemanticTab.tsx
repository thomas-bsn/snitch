import { useI18n } from "../../i18n";
import { SemanticSettings } from "./SemanticSettings";

export function SemanticTab() {
  const { t } = useI18n();
  return (
    <section className="flex flex-col gap-4">
      <h2 className="text-lg font-semibold">{t("admin.tabs.semantic")}</h2>
      <SemanticSettings />
    </section>
  );
}
