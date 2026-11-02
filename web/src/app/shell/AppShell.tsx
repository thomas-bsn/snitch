import {
  Link,
  NavLink,
  Outlet,
  useSearchParams,
  useLocation,
} from "react-router";
import { AddressNotice } from "../components/AddressNotice";
import { buttonClass } from "../components/buttons";
import { Brand, Logo } from "../components/Logo";
import { useI18n } from "../i18n";
import { BellMenu } from "../notifications/BellMenu";
import { NotificationsProvider } from "../notifications/useNotifications";
import { SetupBanner } from "../setup/SetupBanner";
import { AvatarMenu } from "./AvatarMenu";
import { HeaderSearch } from "./HeaderSearch";
import { NAV_ENTRIES } from "./navigation";

const sidebarLink =
  "flex items-center gap-2.5 rounded-md px-2.5 py-2 text-sm font-medium text-nav aria-[current=page]:bg-accent-soft aria-[current=page]:text-accent-ink hover:bg-surface";

const tabLink =
  "flex min-h-14 flex-1 flex-col items-center justify-center gap-0.5 px-1 text-xs font-medium text-nav aria-[current=page]:text-accent";

/**
 * The frame of every signed-in screen: a sidebar with four entries, a header with the
 * search, the Import button and the avatar, and on a phone a tab bar instead of the sidebar.
 */
export function AppShell() {
  const { t } = useI18n();
  const [params, setParams] = useSearchParams();
  const location = useLocation();
  const libraryAddress =
    location.pathname === "/search" ? `/library${location.search}` : "/library";
  const notice = params.get("notice");
  return (
    <NotificationsProvider>
      <div className="flex min-h-dvh bg-canvas text-ink">
        <a
          href="#main"
          className="sr-only focus:not-sr-only focus:fixed focus:left-3 focus:top-3 focus:z-50 focus:rounded-md focus:bg-surface focus:px-3 focus:py-2"
        >
          {t("app.skipToContent")}
        </a>
        <aside className="hidden w-[220px] shrink-0 flex-col gap-1 border-r border-line bg-side px-3 py-5 md:flex">
          <Link to="/library" className="px-2 pb-5" aria-label="Snitch">
            <Brand />
          </Link>
          <nav aria-label={t("nav.label")} className="flex flex-col gap-1">
            {NAV_ENTRIES.map(({ to, label, icon: Icon }) => (
              <NavLink
                key={to}
                to={to === "/library" ? libraryAddress : to}
                className={sidebarLink}
              >
                <Icon className="size-4" />
                {t(label)}
              </NavLink>
            ))}
          </nav>
        </aside>
        <div className="flex min-w-0 flex-1 flex-col">
          <header className="flex items-center gap-3 border-b border-line px-4 py-3 md:px-7">
            <Link
              to="/library"
              className="flex min-h-11 min-w-11 items-center justify-center md:hidden"
              aria-label="Snitch"
            >
              <Logo className="size-8" />
            </Link>
            <HeaderSearch />
            <div className="ml-auto flex items-center gap-3">
              <Link
                to="/import"
                className={`${buttonClass("primary")} hidden md:inline-flex`}
              >
                {t("header.import")}
              </Link>
              <BellMenu />
              <AvatarMenu />
            </div>
          </header>
          <main
            id="main"
            tabIndex={-1}
            className="flex-1 pb-20 outline-none md:pb-0"
          >
            <SetupBanner />
            {notice && (
              <AddressNotice
                code={notice}
                onDismiss={() => {
                  const next = new URLSearchParams(params);
                  next.delete("notice");
                  setParams(next, { replace: true });
                }}
              />
            )}
            <Outlet />
          </main>
        </div>
        <nav
          aria-label={t("nav.label")}
          className="fixed inset-x-0 bottom-0 z-40 flex border-t border-line bg-side md:hidden"
        >
          {NAV_ENTRIES.map(({ to, label, icon: Icon }) => (
            <NavLink
              key={to}
              to={to === "/library" ? libraryAddress : to}
              className={tabLink}
            >
              <Icon className="size-5" />
              {t(label)}
            </NavLink>
          ))}
        </nav>
      </div>
    </NotificationsProvider>
  );
}
