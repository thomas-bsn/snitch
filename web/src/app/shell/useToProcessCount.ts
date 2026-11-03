import { useEffect, useState } from "react";
import { useLocation } from "react-router";
import { fetchToProcess, type ToProcess } from "../people/peopleApi";

const CHANGED = "snitch:to-process-count";

/** The voices still to settle, as the "To process" list shows them: to confirm, ambiguous, unknown. */
export const toProcessCount = (totals: ToProcess["totals"]) =>
  totals.to_confirm + totals.ambiguous + totals.unknown;

/**
 * Tells every counter (tab, sidebar) the new number of voices to settle, from a screen that
 * has just read it or just changed it, so none waits for a change of address.
 */
export function publishToProcessCount(count: number) {
  window.dispatchEvent(new CustomEvent<number>(CHANGED, { detail: count }));
}

/**
 * The number of voices to settle, from the "To process" API, for the badges of the People
 * entry and of its tab. Read again when the path changes and whenever a screen publishes it;
 * a failed read hides it.
 */
export function useToProcessCount(): number | undefined {
  const { pathname } = useLocation();
  const [count, setCount] = useState<number>();
  useEffect(() => {
    let current = true;
    void fetchToProcess("all", 1, 1).then((result) => {
      if (!current) return;
      setCount(result.ok ? toProcessCount(result.value.totals) : undefined);
    });
    return () => {
      current = false;
    };
  }, [pathname]);
  useEffect(() => {
    const listen = (event: Event) =>
      setCount((event as CustomEvent<number>).detail);
    window.addEventListener(CHANGED, listen);
    return () => window.removeEventListener(CHANGED, listen);
  }, []);
  return count;
}
