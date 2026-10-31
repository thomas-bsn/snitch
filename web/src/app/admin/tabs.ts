import type { MessageKey } from "../i18n/messages";

export type SubTab = { path: string; label: MessageKey };

export type AdminTab = {
  /** Path segment under /admin. */
  path: string;
  label: MessageKey;
  /** Sections shown as pills under the tab bar; the first one is the tab's own address. */
  sub?: SubTab[];
};

/** The tabs of the administration, in order. Each ticket fills in its own tab's screen. */
export const ADMIN_TABS: AdminTab[] = [
  { path: "workers", label: "admin.tabs.workers" },
  { path: "queue", label: "admin.tabs.queue" },
  { path: "folders", label: "admin.tabs.folders" },
  {
    path: "transcription",
    label: "admin.tabs.transcription",
    sub: [
      { path: "", label: "admin.subTabs.model" },
      { path: "vocabulary", label: "admin.subTabs.vocabulary" },
    ],
  },
  { path: "voice", label: "admin.tabs.voice" },
  { path: "semantic", label: "admin.tabs.semantic" },
  {
    path: "users",
    label: "admin.tabs.users",
    sub: [
      { path: "", label: "admin.subTabs.members" },
      { path: "signups", label: "admin.subTabs.signups" },
      { path: "person-links", label: "admin.subTabs.personLinks" },
    ],
  },
  { path: "exports", label: "admin.tabs.exports" },
];
