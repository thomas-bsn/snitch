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

/** The moment kept in a clip's address (`?t=75`), or 0 when absent or not valid. */
export function parseMoment(value: string | null): number {
  if (!value || !/^\d{1,7}(\.\d+)?$/.test(value)) return 0;
  return Math.floor(Number(value));
}
