import { useDocumentTitle } from "../../components/useDocumentTitle";
import { useI18n } from "../../i18n";
import { QueueTab } from "./QueueTab";

/** The Task queue, an entry of the sidebar of its own (administrators only; the route guards it). */
export function QueuePage() {
  const { t } = useI18n();
  useDocumentTitle(t("nav.queue"));
  return (
    <section className="flex flex-col gap-4 px-4 py-6 md:px-7">
      <QueueTab />
    </section>
  );
}
