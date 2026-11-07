import { ContextMenu } from "@base-ui/react/context-menu";
import { Menu } from "@base-ui/react/menu";
import {
  ImageUp,
  Info,
  Lock,
  LockOpen,
  MoreHorizontal,
  ScanSearch,
} from "lucide-react";
import type { ReactElement, ReactNode } from "react";
import { useI18n } from "../i18n";
import type { MessageKey, Params } from "../i18n/messages";
import type { ExplorerEntry } from "./explorerApi";

/** What the viewer may do on an entry, from their role. */
export type ExplorerRights = { admin: boolean; superAdmin: boolean };

/** The dialogs a menu can prepare; none of them writes anything by itself. */
export type ExplorerActions = {
  onAnalysis: (entry: ExplorerEntry) => void;
  onPrivacy: (entry: ExplorerEntry) => void;
  onChangeLogo: (entry: ExplorerEntry) => void;
  onProperties: (entry: ExplorerEntry) => void;
};

type Action = {
  key: string;
  label: string;
  icon: ReactNode;
  onSelect: () => void;
};

const itemClass =
  "flex min-h-11 w-full cursor-pointer items-center gap-2 rounded-md px-2.5 py-2 text-left text-sm outline-none data-[highlighted]:bg-side md:min-h-9";
const popupClass =
  "min-w-[14rem] rounded-lg border border-line bg-surface p-1 text-ink shadow-md outline-none";

/** The actions of one entry. A locked private folder has none: its contents are not reachable. */
function actionsFor(
  entry: ExplorerEntry,
  rights: ExplorerRights,
  handlers: ExplorerActions,
  t: (key: MessageKey, values?: Params) => string,
): Action[] {
  if (entry.directory) {
    const actions: Action[] = [];
    if (rights.admin && !entry.locked) {
      actions.push({
        key: "analysis",
        label: t("library.explorer.analyseUnanalysed"),
        icon: <ScanSearch size={14} aria-hidden="true" />,
        onSelect: () => handlers.onAnalysis(entry),
      });
    }
    if (
      rights.superAdmin &&
      !entry.locked &&
      !entry.path.includes("/") &&
      !entry.private
    ) {
      actions.push({
        key: "privacy",
        label: t("library.explorer.makePrivate"),
        icon: <Lock size={14} aria-hidden="true" />,
        onSelect: () => handlers.onPrivacy(entry),
      });
    } else if (
      rights.superAdmin &&
      !entry.locked &&
      !entry.path.includes("/") &&
      entry.private
    ) {
      actions.push({
        key: "privacy",
        label: t("library.explorer.makeShared"),
        icon: <LockOpen size={14} aria-hidden="true" />,
        onSelect: () => handlers.onPrivacy(entry),
      });
    }
    if (rights.admin && !entry.locked) {
      actions.push({
        key: "logo",
        label: t("library.explorer.changeLogo"),
        icon: <ImageUp size={14} aria-hidden="true" />,
        onSelect: () => handlers.onChangeLogo(entry),
      });
    }
    if (!entry.locked) {
      actions.push({
        key: "properties",
        label: t("library.explorer.properties"),
        icon: <Info size={14} aria-hidden="true" />,
        onSelect: () => handlers.onProperties(entry),
      });
    }
    return actions;
  }
  return [
    {
      key: "properties",
      label: t("library.explorer.properties"),
      icon: <Info size={14} aria-hidden="true" />,
      onSelect: () => handlers.onProperties(entry),
    },
  ];
}

/** The three-dot menu of a row, also reachable with the keyboard. */
export function ExplorerEntryMenu({
  entry,
  rights,
  handlers,
}: {
  entry: ExplorerEntry;
  rights: ExplorerRights;
  handlers: ExplorerActions;
}) {
  const { t } = useI18n();
  const actions = actionsFor(entry, rights, handlers, t);
  if (actions.length === 0) return null;
  return (
    <Menu.Root>
      <Menu.Trigger
        aria-label={
          entry.directory
            ? t("library.explorer.folderMenu", { name: entry.name })
            : t("library.explorer.videoMenu", { name: entry.name })
        }
        className="inline-flex min-h-11 min-w-11 items-center justify-center rounded px-1.5 text-mute hover:bg-line md:min-h-0 md:min-w-0"
      >
        <MoreHorizontal size={16} aria-hidden="true" />
      </Menu.Trigger>
      <Menu.Portal>
        <Menu.Positioner align="end" sideOffset={4} className="z-50">
          <Menu.Popup className={popupClass}>
            {actions.map((action) => (
              <Menu.Item
                key={action.key}
                className={itemClass}
                onClick={action.onSelect}
              >
                {action.icon}
                {action.label}
              </Menu.Item>
            ))}
          </Menu.Popup>
        </Menu.Positioner>
      </Menu.Portal>
    </Menu.Root>
  );
}

/**
 * A right click (or long press) on the whole entry opens the same actions. The trigger is the complete
 * element to wrap (a row, or a tile wrapper), so an entry with no action is returned untouched.
 */
export function ExplorerEntryContextMenu({
  entry,
  rights,
  handlers,
  trigger,
}: {
  entry: ExplorerEntry;
  rights: ExplorerRights;
  handlers: ExplorerActions;
  /** The element that receives the right click, children included. */
  trigger: ReactElement;
}) {
  const { t } = useI18n();
  const actions = actionsFor(entry, rights, handlers, t);
  if (actions.length === 0) return trigger;
  return (
    <ContextMenu.Root>
      <ContextMenu.Trigger render={trigger} />
      <ContextMenu.Portal>
        <ContextMenu.Positioner className="z-50">
          <ContextMenu.Popup className={popupClass}>
            {actions.map((action) => (
              <ContextMenu.Item
                key={action.key}
                className={itemClass}
                onClick={action.onSelect}
              >
                {action.icon}
                {action.label}
              </ContextMenu.Item>
            ))}
          </ContextMenu.Popup>
        </ContextMenu.Positioner>
      </ContextMenu.Portal>
    </ContextMenu.Root>
  );
}
