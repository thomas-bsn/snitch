import { useCallback, useEffect, useRef, useState } from "react";
import { Link } from "react-router";
import { ErrorAlert } from "../components/Alert";
import { buttonClass } from "../components/buttons";
import { useI18n } from "../i18n";
import type { MessageKey } from "../i18n/messages";
import { Page } from "../pages/pages";
import { notificationActionLabel, notificationText } from "./notificationText";
import {
  fetchNotifications,
  notificationAddress,
  useNotificationActions,
  type AppNotification,
  type NotificationFilter,
} from "./notificationsApi";
import { useNotifications } from "./useNotifications";

type Filter = NotificationFilter;

/** Items fetched per page; "Show more" asks for the next ones. */
const PAGE = 50;

const FILTERS: { id: Filter; label: MessageKey }[] = [
  { id: "all", label: "notifications.filterAll" },
  { id: "unread", label: "notifications.filterUnread" },
  { id: "actions", label: "notifications.filterActions" },
  { id: "errors", label: "notifications.filterErrors" },
];

const small = "h-9 px-3 md:h-8";
const filterClass = (on: boolean) =>
  `inline-flex min-h-11 items-center rounded-full border px-3.5 text-sm font-medium md:min-h-8 ${
    on
      ? "border-accent-solid bg-accent-soft text-accent-ink"
      : "border-line-strong bg-surface text-ink hover:bg-side"
  }`;

/**
 * The full list of the user's notifications: filters, distinct counts (unread, actions
 * still open), reading and removal of resolved ones, and a direct link to each action.
 * It follows the live stream shared with the bell.
 */
export function NotificationsPage() {
  const { t, language } = useI18n();
  const { list, failed, status, refresh } = useNotifications();
  const actions = useNotificationActions();
  const [filter, setFilter] = useState<Filter>("all");
  const [removeFailed, setRemoveFailed] = useState(false);
  // The page asks the server for the filter it shows (and for more of it), instead of
  // filtering the capped copy the bell keeps: an old unread item or an action is never lost.
  const [shown, setShown] = useState<AppNotification[]>();
  const [total, setTotal] = useState(0);
  const [pageFailed, setPageFailed] = useState(false);
  const [loadingMore, setLoadingMore] = useState(false);
  const loaded = useRef(PAGE);

  const load = useCallback(
    async (count: number) => {
      const out: AppNotification[] = [];
      let all = 0;
      // The server pages by at most 200; a reload after a change refetches what is on screen.
      while (out.length < count) {
        const r = await fetchNotifications({
          filter,
          limit: Math.min(200, count - out.length),
          offset: out.length,
        });
        if (!r.ok) {
          setPageFailed(true);
          return;
        }
        out.push(...r.value.items);
        all = r.value.total;
        if (r.value.items.length === 0 || out.length >= all) break;
      }
      setPageFailed(false);
      setShown(out);
      setTotal(all);
    },
    [filter],
  );

  // A new filter starts from its first page; every change pushed by the stream refreshes the view.
  useEffect(() => {
    loaded.current = PAGE;
    setShown(undefined);
  }, [filter]);
  useEffect(() => {
    void load(loaded.current);
  }, [load, list]);

  const reload = useCallback(async () => {
    await Promise.all([load(loaded.current), refresh()]);
  }, [load, refresh]);

  async function showMore() {
    setLoadingMore(true);
    loaded.current += PAGE;
    await load(loaded.current);
    setLoadingMore(false);
  }

  const items = shown ?? [];
  const active = items.filter((i) => i.state === "active");
  const resolved = items.filter((i) => i.state === "resolved");
  const dates = new Intl.DateTimeFormat(language, {
    dateStyle: "medium",
    timeStyle: "short",
  });

  async function toggle(item: AppNotification) {
    await (item.read ? actions.markUnread(item.id) : actions.markRead(item.id));
    void reload();
  }

  async function remove(ids: number[]) {
    setRemoveFailed(false);
    const results = await Promise.all(ids.map((id) => actions.remove(id)));
    if (results.some((r) => !r.ok)) setRemoveFailed(true);
    void reload();
  }

  function row(item: AppNotification) {
    const actionLabel = notificationActionLabel(item);
    return (
      <li
        key={item.id}
        data-testid="notification-row"
        className="flex flex-col gap-2 rounded-xl border border-line bg-surface p-3 md:flex-row md:items-center md:gap-4"
      >
        <span
          aria-hidden="true"
          className={`hidden size-2 shrink-0 rounded-full md:block ${item.read ? "bg-transparent" : "bg-accent"}`}
        />
        <div className="min-w-0 flex-1">
          <p className={item.read ? "text-mute" : "font-medium"}>
            {notificationText(item, t)}
            {!item.read && (
              <span className="sr-only"> ({t("notifications.unread")})</span>
            )}
          </p>
          <p className="text-xs text-mute">
            {item.state === "resolved" && (
              <span>{t("notifications.resolved")} · </span>
            )}
            <time dateTime={item.updated_at}>
              {dates.format(new Date(item.updated_at))}
            </time>
          </p>
        </div>
        <div className="flex flex-wrap gap-2">
          {actionLabel && (
            <Link
              to={notificationAddress(item)}
              onClick={() => {
                if (!item.read) void actions.markRead(item.id);
              }}
              className={`${buttonClass("primary")} ${small}`}
            >
              {t(actionLabel)}
            </Link>
          )}
          <button
            type="button"
            onClick={() => void toggle(item)}
            className={`${buttonClass()} ${small}`}
          >
            {item.read
              ? t("notifications.markUnread")
              : t("notifications.markRead")}
          </button>
          {item.state === "resolved" && (
            <button
              type="button"
              onClick={() => void remove([item.id])}
              className={`${buttonClass()} ${small}`}
            >
              {t("notifications.remove")}
            </button>
          )}
        </div>
      </li>
    );
  }

  return (
    <Page title={t("notifications.title")}>
      <div className="flex flex-wrap items-center gap-x-6 gap-y-1 text-sm">
        <p data-testid="count-unread" className="font-medium">
          {t("notifications.unreadCount", { count: list?.unread ?? 0 })}
        </p>
        <p data-testid="count-actions" className="font-medium">
          {t("notifications.actionsCount", { count: list?.pending ?? 0 })}
        </p>
        {list && list.unread > 0 && (
          <button
            type="button"
            className={`${buttonClass()} ${small}`}
            onClick={async () => {
              await actions.markAllRead();
              void reload();
            }}
          >
            {t("notifications.markAllRead")}
          </button>
        )}
      </div>
      <div
        role="group"
        aria-label={t("notifications.filters")}
        className="flex flex-wrap gap-2"
      >
        {FILTERS.map(({ id, label }) => (
          <button
            key={id}
            type="button"
            aria-pressed={filter === id}
            onClick={() => setFilter(id)}
            className={filterClass(filter === id)}
          >
            {t(label)}
          </button>
        ))}
      </div>
      <div role="status" className="text-xs text-mute">
        {status === "forbidden"
          ? t("notifications.forbidden")
          : status === "offline"
            ? t("notifications.polling")
            : ""}
      </div>
      {((failed && !list) || (pageFailed && !shown)) && (
        <ErrorAlert>{t("notifications.loadFailed")}</ErrorAlert>
      )}
      {removeFailed && (
        <ErrorAlert>{t("notifications.removeFailed")}</ErrorAlert>
      )}
      {shown && items.length === 0 && (
        <p className="text-mute">
          {filter === "all"
            ? t("notifications.empty")
            : t("notifications.emptyFiltered")}
        </p>
      )}
      {active.length > 0 && (
        <section
          aria-labelledby="notifications-actions"
          className="flex flex-col gap-2"
        >
          <h2 id="notifications-actions" className="text-base font-semibold">
            {t("notifications.actions")}
          </h2>
          <ul
            aria-label={t("notifications.activeList")}
            className="flex flex-col gap-2"
          >
            {active.map(row)}
          </ul>
        </section>
      )}
      {resolved.length > 0 && (
        <section
          aria-labelledby="notifications-history"
          className="flex flex-col gap-2"
        >
          <div className="flex flex-wrap items-center justify-between gap-2">
            <h2 id="notifications-history" className="text-base font-semibold">
              {t("notifications.history")}
            </h2>
            <button
              type="button"
              className={`${buttonClass()} ${small}`}
              onClick={() => void remove(resolved.map((i) => i.id))}
            >
              {t("notifications.removeResolved")}
            </button>
          </div>
          <ul
            aria-label={t("notifications.historyList")}
            className="flex flex-col gap-2"
          >
            {resolved.map(row)}
          </ul>
        </section>
      )}
      {shown && items.length < total && (
        <button
          type="button"
          disabled={loadingMore}
          onClick={() => void showMore()}
          className={`${buttonClass()} ${small} self-start`}
        >
          {t("notifications.showMore", { shown: items.length, total })}
        </button>
      )}
    </Page>
  );
}
