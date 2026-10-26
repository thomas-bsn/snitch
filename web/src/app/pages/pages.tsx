import { Link, useSearchParams } from "react-router";
import { buttonClass } from "../components/buttons";
import { useDocumentTitle } from "../components/useDocumentTitle";
import { useI18n } from "../i18n";
import type { MessageKey } from "../i18n/messages";

/** The frame of a screen: its title as the page heading, then its content. */
export function Page({
  title,
  children,
}: {
  title: string;
  children?: React.ReactNode;
}) {
  useDocumentTitle(title);
  return (
    <section className="flex flex-col gap-4 px-4 py-6 md:px-7">
      <h1 className="text-[22px] font-semibold">{title}</h1>
      {children}
    </section>
  );
}

/** A screen whose content arrives in a later lot of the redesign. */
function Placeholder({ title }: { title: MessageKey }) {
  const { t } = useI18n();
  return (
    <Page title={t(title)}>
      <p className="max-w-xl text-mute">{t("pages.comingSoon")}</p>
      <div>
        <a href="/legacy/" className={buttonClass("secondary")}>
          {t("app.previousInterface")}
        </a>
      </div>
    </Page>
  );
}

export const ImportPage = () => <Placeholder title="pages.import.title" />;

export function SearchPage() {
  const { t } = useI18n();
  const [params] = useSearchParams();
  const query = params.get("q");
  return (
    <Page title={t("pages.search.title")}>
      {query && (
        <p className="text-mute">{t("pages.search.queryLabel", { query })}</p>
      )}
      <p className="max-w-xl text-mute">{t("pages.comingSoon")}</p>
      <div>
        <a href="/legacy/" className={buttonClass("secondary")}>
          {t("app.previousInterface")}
        </a>
      </div>
    </Page>
  );
}

export function Forbidden() {
  const { t } = useI18n();
  return (
    <Page title={t("pages.forbidden.title")}>
      <p className="text-mute">{t("pages.forbidden.body")}</p>
    </Page>
  );
}

export function NotFound() {
  const { t } = useI18n();
  return (
    <Page title={t("pages.notFound.title")}>
      <p className="text-mute">{t("pages.notFound.body")}</p>
      <div>
        <Link to="/library" className={buttonClass("secondary")}>
          {t("pages.notFound.back")}
        </Link>
      </div>
    </Page>
  );
}
