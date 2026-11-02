import { useCallback, useEffect, useState } from "react";
import {
  defaultNotificationFeed,
  type NotificationFeed,
} from "./notificationFeed";
import { fetchNotifications, type NotificationList } from "./notificationsApi";

/** The signed-in user's notifications, refreshed whenever the feed says so. */
export function useNotifications(
  feed: NotificationFeed = defaultNotificationFeed,
) {
  const [list, setList] = useState<NotificationList>();
  const [failed, setFailed] = useState(false);
  const refresh = useCallback(async () => {
    const result = await fetchNotifications();
    if (result.ok) {
      setList(result.value);
      setFailed(false);
    } else {
      setFailed(true);
    }
  }, []);
  useEffect(() => {
    void refresh();
    return feed.subscribe(() => void refresh());
  }, [feed, refresh]);
  return { list, failed, refresh };
}
