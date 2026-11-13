import { useEffect, useState } from "react";
import {
  fetchExplorerProperties,
  type ExplorerProperties,
} from "./explorerApi";

// The list view has no aggregate for a folder: each folder row asks for the same properties as the
// Properties dialog, so both always agree. A few at a time, so a long listing never floods the server
// with subtree walks; a folder that cannot be summed (too large, unavailable) keeps its dashes.
const MAX_CONCURRENT = 3;
let running = 0;
const waiting: (() => void)[] = [];

async function limited<T>(work: () => Promise<T>): Promise<T> {
  if (running >= MAX_CONCURRENT)
    await new Promise<void>((resolve) => waiting.push(resolve));
  running++;
  try {
    return await work();
  } finally {
    running--;
    waiting.shift()?.();
  }
}

/** The properties of a folder row, loaded in the background; undefined while loading or on failure. */
export function useFolderSummary(
  path: string,
  enabled: boolean,
  // Any new value (a refreshed listing) loads the summary again.
  revision: unknown,
): ExplorerProperties | undefined {
  const [summary, setSummary] = useState<ExplorerProperties>();
  useEffect(() => {
    if (!enabled) return;
    let current = true;
    void limited(async () => {
      // A row scrolled away or replaced before its turn costs nothing.
      if (!current) return;
      const result = await fetchExplorerProperties(path).catch(() => undefined);
      if (current) setSummary(result?.ok ? result.value : undefined);
    });
    return () => {
      current = false;
    };
  }, [path, enabled, revision]);
  return enabled ? summary : undefined;
}
