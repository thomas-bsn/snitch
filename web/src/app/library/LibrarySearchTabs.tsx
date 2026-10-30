import { NavLink, useLocation } from "react-router";
import { useI18n } from "../i18n";

const SHARED = ["from", "to", "source", "person", "state"];

/** Switch views while retaining shared filters and each view's own address state. */
export function LibrarySearchTabs() {
  const { t } = useI18n();
  const location = useLocation();
  const params = new URLSearchParams(location.search);
  const target = location.pathname === "/search" ? "/library" : "/search";
  const next = new URLSearchParams();
  for (const key of SHARED) {
    const value = params.get(key);
    if (value) next.set(key, value);
  }
  const saved = sessionStorage.getItem(`snitch:view:${target}`);
  if (saved) {
    const previous = new URLSearchParams(saved);
    for (const key of target === "/search" ? ["q", "mode", "visibility"] : ["owner"]) {
      const value = previous.get(key);
      if (value) next.set(key, value);
    }
  }
  sessionStorage.setItem(`snitch:view:${location.pathname}`, location.search);
  const other = `${target}${next.size ? `?${next}` : ""}`;
  const tab = "inline-flex min-h-11 items-center rounded-lg px-4 text-sm font-medium aria-[current=page]:bg-accent-soft aria-[current=page]:text-accent-ink";
  return (
    <nav aria-label={t("nav.library")} className="flex gap-2 border-b border-line pb-2">
      <NavLink to={location.pathname === "/library" ? location.pathname + location.search : other} className={tab} end>{t("nav.library")}</NavLink>
      <NavLink to={location.pathname === "/search" ? location.pathname + location.search : other} className={tab} end>{t("nav.search")}</NavLink>
    </nav>
  );
}
