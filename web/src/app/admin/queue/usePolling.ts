import { useEffect } from "react";

/** Interval of the intermediate refresh, until the live feed of #131. */
export const POLL_INTERVAL_MS = 5000;

/**
 * Calls `tick` every few seconds while the tab is visible. Nothing runs while
 * the tab is hidden; coming back to the tab refreshes once at once.
 */
export function usePolling(tick: () => void, interval = POLL_INTERVAL_MS) {
  useEffect(() => {
    let timer: number | undefined;
    const stop = () => {
      if (timer !== undefined) window.clearInterval(timer);
      timer = undefined;
    };
    const start = () => {
      stop();
      timer = window.setInterval(tick, interval);
    };
    const onVisibility = () => {
      if (document.visibilityState === "visible") {
        tick();
        start();
      } else stop();
    };
    if (document.visibilityState === "visible") start();
    document.addEventListener("visibilitychange", onVisibility);
    return () => {
      stop();
      document.removeEventListener("visibilitychange", onVisibility);
    };
  }, [tick, interval]);
}
