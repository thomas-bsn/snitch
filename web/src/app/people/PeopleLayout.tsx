import { Link, NavLink, Outlet, useLocation } from "react-router";
import { useDocumentTitle } from "../components/useDocumentTitle";
import { useI18n } from "../i18n";
import type { MessageKey } from "../i18n/messages";
import { useToProcessCount } from "../shell/useToProcessCount";

type PeopleTab = { path: string; label: MessageKey };

/** The sections of the People screen, in order; each one is an address. */
export const PEOPLE_TABS: PeopleTab[] = [
  { path: "", label: "people.tabs.persons" },
  { path: "verification", label: "people.tabs.verification" },
  { path: "groups", label: "people.tabs.groups" },
];

const tabClass = ({ isActive }: { isActive: boolean }) =>
  `-mb-px inline-flex h-11 shrink-0 items-center border-b-2 px-3 font-medium whitespace-nowrap md:h-10 ${
    isActive
      ? "border-accent text-ink"
      : "border-transparent text-mute hover:text-ink"
  }`;

/** The People screen: a title, its sections as a tab bar, and the section below. */
export function PeopleLayout() {
  const { t } = useI18n();
  useDocumentTitle(t("pages.people.title"));
  // The record of a Person lives under /people/:id, so the first tab stays current there.
  const { pathname } = useLocation();
  const toProcess = useToProcessCount();
  const onPersons = !/^\/people\/(verification|groups)(\/|$)/.test(pathname);
  return (
    <section className="flex flex-col gap-4 px-4 py-6 md:px-7">
      <div>
        <h1 className="text-[22px] font-semibold">{t("pages.people.title")}</h1>
        <p className="mt-1 max-w-2xl text-mute">{t("people.lead")}</p>
      </div>
      <nav
        aria-label={t("people.tabsLabel")}
        tabIndex={0}
        className="-mx-4 overflow-x-auto border-b border-line px-4 md:mx-0 md:px-0"
      >
        <ul className="m-0 flex list-none gap-1 p-0">
          {PEOPLE_TABS.map((tab) => (
            <li key={tab.path}>
              {tab.path === "" ? (
                <Link
                  to="/people"
                  aria-current={onPersons ? "page" : undefined}
                  className={tabClass({ isActive: onPersons })}
                >
                  {t(tab.label)}
                </Link>
              ) : (
                <NavLink to={`/people/${tab.path}`} end className={tabClass}>
                  {t(tab.label)}
                  {tab.path === "verification" && toProcess !== undefined && (
                    <b className="ml-2 rounded-full bg-accent-soft px-2 text-xs text-accent-ink">
                      {toProcess}
                    </b>
                  )}
                </NavLink>
              )}
            </li>
          ))}
        </ul>
      </nav>
      <Outlet />
    </section>
  );
}
