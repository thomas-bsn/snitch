import type { WorkerLogLine } from "./workersApi";

/** Lines kept in the page: the oldest go first, so a Worker that never stops cannot fill the browser. */
export const MAX_KEPT_LINES = 5000;

export type ParsedLogLine = {
  at?: string;
  level?: "DEBUG" | "INFO" | "WARN" | "ERROR";
  event?: string;
  /** The other keys of the JSON line, as text. */
  fields: [string, string][];
  /** The line as the Worker wrote it: what is shown when it is not structured. */
  raw: string;
};

const LEVELS = new Set(["DEBUG", "INFO", "WARN", "ERROR"]);
const KNOWN = new Set(["timestamp", "level", "service", "event"]);

const text = (value: unknown): string =>
  typeof value === "string" ? value : JSON.stringify(value);

/** Reads a Worker log line; a line that is not a JSON object stays as it is. */
export function parseLogLine(raw: string): ParsedLogLine {
  let value: unknown;
  try {
    value = JSON.parse(raw);
  } catch {
    return { fields: [], raw };
  }
  if (typeof value !== "object" || value === null || Array.isArray(value)) {
    return { fields: [], raw };
  }
  const record = value as Record<string, unknown>;
  const level = typeof record.level === "string" ? record.level : "";
  return {
    at: typeof record.timestamp === "string" ? record.timestamp : undefined,
    level: LEVELS.has(level) ? (level as ParsedLogLine["level"]) : undefined,
    event: typeof record.event === "string" ? record.event : undefined,
    fields: Object.entries(record)
      .filter(([key]) => !KNOWN.has(key))
      .map(([key, field]) => [key, text(field)]),
    raw,
  };
}

/** Appends the lines that arrived, ignoring one already held, and keeps the newest MAX_KEPT_LINES. */
export function appendLines(
  held: WorkerLogLine[],
  arrived: WorkerLogLine[],
  max = MAX_KEPT_LINES,
): { lines: WorkerLogLine[]; trimmed: boolean } {
  const last = held.length ? held[held.length - 1].id : 0;
  const fresh = arrived.filter((line) => line.id > last);
  const all = held.concat(fresh);
  return all.length > max
    ? { lines: all.slice(all.length - max), trimmed: true }
    : { lines: all, trimmed: false };
}

/** Puts older lines in front of the ones held, ignoring one already held. The newest lines are never dropped for them. */
export function prependLines(
  held: WorkerLogLine[],
  older: WorkerLogLine[],
  max = MAX_KEPT_LINES,
): WorkerLogLine[] {
  const first = held.length ? held[0].id : Number.MAX_SAFE_INTEGER;
  const room = Math.max(0, max - held.length);
  const fit = older.filter((line) => line.id < first).slice(-room || undefined);
  return room === 0 ? held : fit.concat(held);
}
