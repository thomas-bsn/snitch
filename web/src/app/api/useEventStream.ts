import { useEffect, useRef, useState } from "react";

/** Where a live view stands: connected to the stream, trying again, or refused. */
export type StreamStatus = "connecting" | "live" | "offline" | "forbidden";

/**
 * Follows a server-sent event stream whose events are complete snapshots. The browser
 * reconnects by itself and each connection starts with a fresh snapshot, so a missed event
 * never leaves a stale view. While the stream is down, `poll` is called at a moderate pace
 * as a fallback (suspended while the tab is hidden); a snapshot older than the last one seen (by sequence) is ignored.
 */
export function useEventStream<T, D = never>(
  url: string | null,
  onSnapshot: (snapshot: T) => void,
  poll: () => void,
  pollMs = 5000,
  onDelta?: (delta: D) => boolean,
): StreamStatus {
  const [status, setStatus] = useState<StreamStatus>("connecting");
  const [epoch, setEpoch] = useState(0);
  const handler = useRef(onSnapshot);
  const fallback = useRef(poll);
  const deltaHandler = useRef(onDelta);
  const wantsDelta = onDelta !== undefined;
  useEffect(() => {
    handler.current = onSnapshot;
    fallback.current = poll;
    deltaHandler.current = onDelta;
  });

  useEffect(() => {
    if (!url || typeof EventSource === "undefined") return;
    let seq = 0;
    // With a delta handler, the stream sends the whole state once, then only what changed.
    const source = new EventSource(
      wantsDelta ? `${url}${url.includes("?") ? "&" : "?"}delta=1` : url,
    );
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
    source.addEventListener("delta", (event) => {
      try {
        const frame = JSON.parse((event as MessageEvent<string>).data) as {
          seq: number;
          base: number;
          delta: D;
        };
        if (frame.seq < seq) return;
        // A change that does not follow the state held is never guessed: reconnect, which
        // starts again from a whole snapshot.
        if (frame.base !== seq || !deltaHandler.current?.(frame.delta)) {
          source.close();
          setEpoch((n) => n + 1);
          return;
        }
        seq = frame.seq;
        setStatus("live");
      } catch {
        source.close();
        setEpoch((n) => n + 1);
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
  }, [url, wantsDelta, epoch]);

  // The fallback is for a visible tab only: a hidden one stops asking, and asks once as soon as
  // it is shown again.
  const [visible, setVisible] = useState(
    () =>
      typeof document === "undefined" || document.visibilityState !== "hidden",
  );
  useEffect(() => {
    const change = () => {
      const shown = document.visibilityState !== "hidden";
      setVisible(shown);
      if (shown) fallback.current();
    };
    document.addEventListener("visibilitychange", change);
    return () => document.removeEventListener("visibilitychange", change);
  }, []);

  const polling = status === "offline" || status === "connecting";
  useEffect(() => {
    if (!url || !polling || !visible) return;
    // A stream that just failed is covered at once, not after the first interval.
    if (status === "offline") fallback.current();
    const timer = setInterval(() => fallback.current(), pollMs);
    return () => clearInterval(timer);
  }, [url, polling, visible, pollMs, status]);
  return url ? status : "offline";
}
