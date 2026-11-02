import {
  createContext,
  useCallback,
  useContext,
  useState,
  type ReactNode,
} from "react";
import { useEventStream, type StreamStatus } from "../api/useEventStream";
import { fetchNotifications, type NotificationList } from "./notificationsApi";

type Notifications = {
  list: NotificationList | undefined;
  failed: boolean;
  /** Where the live stream stands; "offline" means polling covers it. */
  status: StreamStatus;
  refresh: () => Promise<void>;
};

const NotificationsContext = createContext<Notifications | undefined>(
  undefined,
);

/**
 * The signed-in user's notifications, shared by the bell and the page: one server-sent
 * stream of complete snapshots (rights rechecked by the server at each one and at every
 * reconnection), with short polling only while the stream is down.
 */
export function NotificationsProvider({ children }: { children: ReactNode }) {
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
  const status = useEventStream<NotificationList>(
    "/api/notifications/events",
    useCallback((snapshot: NotificationList) => {
      setList(snapshot);
      setFailed(false);
    }, []),
    () => void refresh(),
    10_000,
  );
  return (
    <NotificationsContext.Provider value={{ list, failed, status, refresh }}>
      {children}
    </NotificationsContext.Provider>
  );
}

export function useNotifications(): Notifications {
  const value = useContext(NotificationsContext);
  if (!value) throw new Error("useNotifications needs a NotificationsProvider");
  return value;
}
