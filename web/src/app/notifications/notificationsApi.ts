import type { components } from "../../api.gen";
import { api, errorCode, mutation } from "../api/client";
import type { Result } from "../api/loaded";
import { useSession } from "../session";

export type AppNotification = components["schemas"]["Notification"];
export type NotificationList = components["schemas"]["NotificationList"];

const failure = (error: unknown): { ok: false; code: string } => ({
  ok: false,
  code: errorCode(error) ?? "unknown",
});

async function guard<T>(call: () => Promise<Result<T>>): Promise<Result<T>> {
  try {
    return await call();
  } catch {
    return { ok: false, code: "unknown" };
  }
}

export type NotificationFilter = "all" | "unread" | "actions" | "errors";

/** One page of the list; the filter runs on the server, so it never depends on what the client holds. */
export const fetchNotifications = (
  page: { filter?: NotificationFilter; limit?: number; offset?: number } = {},
) =>
  guard<NotificationList>(async () => {
    const { data, error } = await api.GET("/api/notifications", {
      params: { query: page },
    });
    return data ? { ok: true, value: data } : failure(error);
  });

/** Where a notification leads: the Workers page, the job queue, or the Verification list on its clip. */
export function notificationAddress(item: AppNotification): string {
  if (item.generic)
    return item.clip_id === null ? "/library" : `/clips/${item.clip_id}`;
  if (item.kind === "worker_unavailable") return "/admin/workers";
  if (item.kind !== "needs_speaker_review") return "/admin/queue";
  return item.clip_id === null
    ? "/people/verification"
    : `/people/verification?clip=${item.clip_id}`;
}

/** Reading is individual and never resolves the shared action. */
export function useNotificationActions() {
  const { session } = useSession();
  const header = mutation(session?.csrf_token);
  return {
    markRead: (id: number) =>
      guard<true>(async () => {
        const { response, error } = await api.PUT(
          "/api/notifications/{id}/read",
          { params: { path: { id }, header } },
        );
        return response.ok ? { ok: true, value: true } : failure(error);
      }),
    markUnread: (id: number) =>
      guard<true>(async () => {
        const { response, error } = await api.DELETE(
          "/api/notifications/{id}/read",
          { params: { path: { id }, header } },
        );
        return response.ok ? { ok: true, value: true } : failure(error);
      }),
    markAllRead: () =>
      guard<true>(async () => {
        const { response, error } = await api.POST("/api/notifications/read", {
          params: { header },
        });
        return response.ok ? { ok: true, value: true } : failure(error);
      }),
    /** Only a resolved notification can be removed, from the caller's history. */
    remove: (id: number) =>
      guard<true>(async () => {
        const { response, error } = await api.DELETE(
          "/api/notifications/{id}",
          { params: { path: { id }, header } },
        );
        return response.ok ? { ok: true, value: true } : failure(error);
      }),
  };
}
