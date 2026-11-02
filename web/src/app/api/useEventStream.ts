import { useEffect, useRef, useState } from "react";

/** Where a live view stands: connected to the stream, trying again, or refused. */
export type StreamStatus = "connecting" | "live" | "offline" | "forbidden";

/**
 * Follows a server-sent event stream whose events are complete snapshots. The browser
 * reconnects by itself and each connection starts with a fresh snapshot, so a missed event
 * never leaves a stale view. While the stream is down, `poll` is called at a moderate pace
 * as a fallback; a snapshot older than the last one seen (by sequence) is ignored.
 */
export function useEventStream<T>(
  url: string | null,
  onSnapshot: (snapshot: T) => void,
  poll: () => void,
  pollMs = 5000,
): StreamStatus {
  const [status, setStatus] = useState<StreamStatus>("connecting");
  const handler = useRef(onSnapshot);
  const fallback = useRef(poll);
  useEffect(() => {
    handler.current = onSnapshot;
    fallback.current = poll;
  });

  useEffect(() => {
    if (!url || typeof EventSource === "undefined") return;
    let seq = 0;
    const source = new EventSource(url);
    source.addEventListener("snapshot", (event) => {
      try {
        const frame = JSON.parse((event as MessageEvent<string>).data) as {
          seq: number;
          snapshot: T;
        };
        if (frame.seq < seq) return;
        seq = frame.seq;
        setStatus("live");
        handler.current(frame.snapshot);
      } catch {
        // A malformed frame is skipped: the next snapshot carries the whole state.
      }
    });
    source.addEventListener("forbidden", () => {
      source.close();
      setStatus("forbidden");
    });
    source.onerror = () => {
      // The browser retries by itself; the polling fallback covers a refused stream.
      setStatus((current) => (current === "forbidden" ? current : "offline"));
    };
    return () => source.close();
  }, [url]);

  const polling = status === "offline" || status === "connecting";
  useEffect(() => {
    if (!url || !polling) return;
    const timer = setInterval(() => fallback.current(), pollMs);
    return () => clearInterval(timer);
  }, [url, polling, pollMs]);
  return url ? status : "offline";
}
