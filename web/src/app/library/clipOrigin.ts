import { useMemo } from "react";
import { useLocation } from "react-router";

/**
 * Where a clip was opened from, so that "Back" returns to the same folder, filters and
 * position. The address (folder, filters, page, view) travels in the history entry's state;
 * the scroll position is kept per address in the session, because the native back button
 * restores no state of its own.
 */

const SCROLL_KEY = "snitch.library.scroll";

type Scrolls = Record<string, number>;

function readScrolls(): Scrolls {
  try {
    const parsed: unknown = JSON.parse(
      window.sessionStorage.getItem(SCROLL_KEY) ?? "{}",
    );
    return parsed && typeof parsed === "object" ? (parsed as Scrolls) : {};
  } catch {
    return {};
  }
}

function writeScrolls(scrolls: Scrolls) {
  try {
    window.sessionStorage.setItem(SCROLL_KEY, JSON.stringify(scrolls));
  } catch {
    // Without storage the position is simply not restored.
  }
}

export function rememberScroll(address: string) {
  writeScrolls({ ...readScrolls(), [address]: Math.round(window.scrollY) });
}

/** The position kept for an address, read once: a later visit starts at the top. */
export function takeScroll(address: string): number | null {
  const scrolls = readScrolls();
  const value = scrolls[address];
  if (typeof value !== "number") return null;
  delete scrolls[address];
  writeScrolls(scrolls);
  return value;
}

/** Only an address of the Library or of the search can be gone back to. */
export function backAddress(state: unknown): string {
  const from = (state as { from?: unknown } | null)?.from;
  return typeof from === "string" && /^\/(library|search)(\?|$)/.test(from)
    ? from
    : "/library";
}

/** The state to give a link to a clip, and what to do when it is followed. */
export function useClipOrigin() {
  const location = useLocation();
  const from = `${location.pathname}${location.search}`;
  return useMemo(
    () => ({ state: { from }, remember: () => rememberScroll(from) }),
    [from],
  );
}
