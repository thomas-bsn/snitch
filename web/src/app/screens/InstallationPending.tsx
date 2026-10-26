import { buttonClass } from "../components/buttons";
import { useDocumentTitle } from "../components/useDocumentTitle";
import { useI18n } from "../i18n";
import { AuthLayout } from "./AuthLayout";

/**
 * Until the guided installation exists in this interface, an instance without an
 * administrator is finished in the previous interface.
 */
export function InstallationPending() {
  const { t } = useI18n();
  useDocumentTitle(t("installation.title"));
  return (
    <AuthLayout title={t("installation.title")} lead={t("installation.body")}>
      <a href="/legacy/" className={`${buttonClass("primary")} w-full`}>
        {t("installation.open")}
      </a>
    </AuthLayout>
  );
}
