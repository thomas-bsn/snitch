import { useEffect } from "react";

/** Sets the tab title to "<screen> · Snitch" while the screen is shown. */
export function useDocumentTitle(screen?: string) {
  useEffect(() => {
    document.title = screen ? `${screen} · Snitch` : "Snitch";
  }, [screen]);
}
