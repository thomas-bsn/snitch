import { Menu } from "@base-ui/react/menu";
import { Bell } from "lucide-react";
import { useNavigate } from "react-router";
import { useI18n } from "../i18n";
import {
  notificationAddress,
  useNotificationActions,
} from "./notificationsApi";
import type { AppNotification } from "./notificationsApi";
import { notificationText } from "./notificationText";
import { useNotifications } from "./useNotifications";

const itemClass =
  "flex min-h-11 w-full cursor-pointer items-start gap-2.5 rounded-md px-2.5 py-2 text-sm outline-none data-[highlighted]:bg-side md:min-h-9";

/** The bell next to the avatar: the unread count, and a menu of the latest notifications. */
export function BellMenu() {
  const { t } = useI18n();
  const navigate = useNavigate();
  const { list, failed, refresh } = useNotifications();
  const actions = useNotificationActions();
  const unread = list?.unread ?? 0;
  const items = list?.items ?? [];

  async function open(item: AppNotification) {
    // Opening reads the notification; the shared action stays pending until handled.
    if (!item.read) await actions.markRead(item.id);
    void refresh();
    void navigate(notificationAddress(item));
  }

  return (
    <Menu.Root onOpenChange={(opened) => opened && void refresh()}>
      <Menu.Trigger
        aria-label={
          unread > 0
            ? t("notifications.bellUnread", { count: unread })
            : t("notifications.bell")
        }
        className="relative flex size-11 shrink-0 items-center justify-center rounded-full text-nav outline-none hover:bg-side md:size-8"
      >
        <Bell className="size-5 md:size-4" aria-hidden="true" />
        {unread > 0 && (
          <span
            data-testid="bell-count"
            aria-hidden="true"
            className="absolute right-0.5 top-0.5 flex min-w-4 items-center justify-center rounded-full bg-accent px-1 text-[10px] font-semibold leading-4 text-white"
          >
            {unread > 99 ? "99+" : unread}
          </span>
        )}
      </Menu.Trigger>
      <Menu.Portal>
        <Menu.Positioner align="end" sideOffset={8} className="z-50">
          <Menu.Popup className="w-80 max-w-[calc(100vw-2rem)] rounded-xl border border-line bg-surface p-1.5 text-ink shadow-pop outline-none">
            <div className="flex items-center justify-between px-2.5 pb-1.5 pt-2">
              <span className="text-sm font-semibold">
                {t("notifications.title")}
              </span>
              {list && list.pending > 0 && (
                <span className="text-xs text-mute">
                  {t("notifications.pending", { count: list.pending })}
                </span>
              )}
            </div>
            <Menu.Separator className="my-1 h-px bg-line" />
            {failed && !list && (
              <p role="alert" className="px-2.5 py-2 text-sm text-mute">
                {t("notifications.loadFailed")}
              </p>
            )}
            {list && items.length === 0 && (
              <p className="px-2.5 py-3 text-sm text-mute">
                {t("notifications.empty")}
              </p>
            )}
            <div className="max-h-96 overflow-y-auto">
              {items.map((item) => (
                <Menu.Item
                  key={item.id}
                  className={itemClass}
                  onClick={() => void open(item)}
                >
                  <span
                    aria-hidden="true"
                    className={`mt-1.5 size-2 shrink-0 rounded-full ${item.read ? "bg-transparent" : "bg-accent"}`}
                  />
                  <span className="min-w-0 flex-1">
                    <span
                      className={`block ${item.read ? "text-mute" : "font-medium"}`}
                    >
                      {notificationText(item, t)}
                    </span>
                    {!item.read && (
                      <span className="sr-only">
                        {t("notifications.unread")}
                      </span>
                    )}
                    {item.state === "active" &&
                      !item.generic &&
                      item.kind !== "needs_speaker_review" && (
                        <span className="block text-xs text-accent">
                          {item.kind === "worker_unavailable"
                            ? t("notifications.incidentWorkerAction")
                            : t("notifications.incidentJobAction")}
                        </span>
                      )}
                    {item.state === "resolved" && (
                      <span className="block text-xs text-mute">
                        {t("notifications.resolved")}
                      </span>
                    )}
                  </span>
                </Menu.Item>
              ))}
            </div>
            <Menu.Separator className="my-1 h-px bg-line" />
            <Menu.Item
              className={`${itemClass} items-center font-medium text-accent-ink`}
              onClick={() => void navigate("/notifications")}
            >
              {t("notifications.viewAll")}
            </Menu.Item>
            {unread > 0 && (
              <>
                <Menu.Separator className="my-1 h-px bg-line" />
                <Menu.Item
                  className={`${itemClass} items-center font-medium`}
                  closeOnClick={false}
                  onClick={async () => {
                    await actions.markAllRead();
                    void refresh();
                  }}
                >
                  {t("notifications.markAllRead")}
                </Menu.Item>
              </>
            )}
          </Menu.Popup>
        </Menu.Positioner>
      </Menu.Portal>
    </Menu.Root>
  );
}
