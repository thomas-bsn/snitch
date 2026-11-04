import {
  Library,
  ListChecks,
  Upload,
  Users,
  type LucideIcon,
} from "lucide-react";
import type { MessageKey } from "../i18n/messages";

export type NavEntry = {
  to: string;
  label: MessageKey;
  icon: LucideIcon;
  /** Shown to administrators and the super-administrator only. */
  adminOnly?: boolean;
};

/** The entries of the sidebar (and of the tab bar on a phone). */
export const NAV_ENTRIES: NavEntry[] = [
  { to: "/library", label: "nav.library", icon: Library },
  { to: "/people", label: "nav.people", icon: Users },
  { to: "/import", label: "nav.import", icon: Upload },
  { to: "/queue", label: "nav.queue", icon: ListChecks, adminOnly: true },
];
