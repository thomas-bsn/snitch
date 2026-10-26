import { Library, Search, Upload, Users, type LucideIcon } from "lucide-react";
import type { MessageKey } from "../i18n/messages";

export type NavEntry = {
  to: string;
  label: MessageKey;
  icon: LucideIcon;
};

/** The four entries of the sidebar (and of the tab bar on a phone). */
export const NAV_ENTRIES: NavEntry[] = [
  { to: "/library", label: "nav.library", icon: Library },
  { to: "/search", label: "nav.search", icon: Search },
  { to: "/people", label: "nav.people", icon: Users },
  { to: "/import", label: "nav.import", icon: Upload },
];
