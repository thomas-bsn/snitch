/**
 * The seam between the notifications and how the browser learns that something changed.
 * A feed only says "refresh now"; the data always comes from the notifications API. Today
 * the feed is short polling; live updates (SSE, #169) replace pollingFeed with a feed
 * that calls onChange when the server pushes an event, and nothing else has to change.
 */
export type NotificationFeed = {
  subscribe(onChange: () => void): () => void;
};

/** Polling every interval, and again when the tab comes back into view. */
export function pollingFeed(intervalMs = 20_000): NotificationFeed {
  return {
    subscribe(onChange) {
      const timer = window.setInterval(() => {
        if (document.visibilityState === "visible") onChange();
      }, intervalMs);
      const onVisible = () => {
        if (document.visibilityState === "visible") onChange();
      };
      document.addEventListener("visibilitychange", onVisible);
      return () => {
        window.clearInterval(timer);
        document.removeEventListener("visibilitychange", onVisible);
      };
    },
  };
}

export const defaultNotificationFeed: NotificationFeed = pollingFeed();
