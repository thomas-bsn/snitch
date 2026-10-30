import { NavLink, useLocation } from "react-router";
import { useI18n } from "../i18n";

/** The URL carries both views' filters so switching and sharing preserve context. */
export function LibrarySearchTabs() {
  const { t } = useI18n();
  const location = useLocation();
  const params = new URLSearchParams(location.search);
  params.delete("page");
  const suffix = params.size ? `?${params}` : "";
  const tab =
    "inline-flex min-h-11 items-center rounded-lg px-4 text-sm font-medium aria-[current=page]:bg-accent-soft aria-[current=page]:text-accent-ink";
  return (
    <nav
      aria-label={t("nav.library")}
      className="flex gap-2 border-b border-line pb-2"
    >
      <NavLink
        to={`/library${location.pathname === "/library" ? location.search : suffix}`}
        className={tab}
        end
      >
        {t("nav.library")}
      </NavLink>
      <NavLink
        to={`/search${location.pathname === "/search" ? location.search : suffix}`}
        className={tab}
        end
      >
        {t("nav.search")}
      </NavLink>
    </nav>
  );
}
