/** "1:05" or "1:02:05" for a number of seconds. */
export function formatTimecode(totalSeconds: number): string {
  const whole = Math.max(0, Math.floor(totalSeconds));
  const hours = Math.floor(whole / 3600);
  const minutes = Math.floor((whole % 3600) / 60);
  const seconds = String(whole % 60).padStart(2, "0");
  return hours > 0
    ? `${hours}:${String(minutes).padStart(2, "0")}:${seconds}`
    : `${minutes}:${seconds}`;
}

/** A file size in the units of the interface language. */
export function formatSize(bytes: number, language: string) {
  const units = ["B", "KB", "MB", "GB", "TB"];
  let value = bytes;
  let unit = 0;
  while (value >= 1024 && unit < units.length - 1) {
    value /= 1024;
    unit += 1;
  }
  return `${new Intl.NumberFormat(language, { maximumFractionDigits: 1 }).format(value)} ${units[unit]}`;
}

/** The moment kept in a clip's address (`?t=75`), or 0 when absent or not valid. */
export function parseMoment(value: string | null): number {
  if (!value || !/^\d{1,7}(\.\d+)?$/.test(value)) return 0;
  return Math.floor(Number(value));
}
