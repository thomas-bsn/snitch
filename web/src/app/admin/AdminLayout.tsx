import { NavLink, Outlet, useLocation } from "react-router";
import { RenderBoundary } from "../components/RenderBoundary";
import { useDocumentTitle } from "../components/useDocumentTitle";
import { useI18n } from "../i18n";
import { ADMIN_TABS } from "./tabs";

const tabClass = ({ isActive }: { isActive: boolean }) =>
  `-mb-px inline-flex h-11 shrink-0 items-center border-b-2 px-3 font-medium whitespace-nowrap md:h-10 ${
    isActive
      ? "border-accent text-ink"
      : "border-transparent text-mute hover:text-ink"
  }`;

const pillClass = ({ isActive }: { isActive: boolean }) =>
  `inline-flex h-11 items-center rounded-full border px-3.5 text-sm font-medium md:h-8 ${
    isActive
      ? "border-accent bg-accent-soft text-accent-ink"
      : "border-line-strong text-nav hover:bg-side"
  }`;

/**
 * The administration: a horizontal tab bar, optional pills for the sections of a tab, and the
 * screen of the tab below. Reserved to administrators (the route guards it).
 */
export function AdminLayout() {
  const { t } = useI18n();
  const { pathname } = useLocation();
  useDocumentTitle(t("admin.title"));
  const current = ADMIN_TABS.find(
    (tab) =>
      pathname === `/admin/${tab.path}` ||
      pathname.startsWith(`/admin/${tab.path}/`),
  );

  return (
    <section className="flex flex-col gap-4 px-4 py-6 md:px-7">
      <h1 className="text-[22px] font-semibold">{t("admin.title")}</h1>
      <nav
        aria-label={t("admin.tabsLabel")}
        tabIndex={0}
        className="-mx-4 overflow-x-auto border-b border-line px-4 md:mx-0 md:px-0"
      >
        <ul className="m-0 flex list-none gap-1 p-0">
          {ADMIN_TABS.map((tab) => (
            <li key={tab.path}>
              <NavLink
                to={`/admin/${tab.path}`}
                className={tabClass}
                // A tab stays current on its sub-sections and on a Worker's page.
                end={false}
              >
                {t(tab.label)}
              </NavLink>
            </li>
          ))}
        </ul>
      </nav>
      {current?.sub && (
        <nav aria-label={t("admin.subTabsLabel")}>
          <ul className="m-0 flex list-none flex-wrap gap-2 p-0">
            {current.sub.map((sub) => (
              <li key={sub.path}>
                <NavLink
                  to={`/admin/${current.path}${sub.path ? `/${sub.path}` : ""}`}
                  end
                  className={pillClass}
                >
                  {t(sub.label)}
                </NavLink>
              </li>
            ))}
          </ul>
        </nav>
      )}
      <RenderBoundary resetKey={pathname}>
        <Outlet />
      </RenderBoundary>
    </section>
  );
}

/** A tab whose screen arrives with a later ticket. */
export function AdminPlaceholder() {
  const { t } = useI18n();
  return (
    <div className="flex flex-col gap-3">
      <p className="max-w-xl text-mute">{t("pages.comingSoon")}</p>
    </div>
  );
}
