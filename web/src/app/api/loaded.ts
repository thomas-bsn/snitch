import { useCallback, useEffect, useState } from "react";
import { errorCode } from "./client";

/** A call that can fail with the stable code the server gave. */
export type Result<T> = { ok: true; value: T } | { ok: false; code: string };

/** The failed result of an answer the server refused, with its stable code. */
export const failure = (error: unknown): { ok: false; code: string } => ({
  ok: false,
  code: errorCode(error) ?? "unknown",
});

/** Runs a call, turning a network failure into the "unavailable" result. */
export async function guard<T>(
  call: () => Promise<Result<T>>,
): Promise<Result<T>> {
  try {
    return await call();
  } catch {
    return { ok: false, code: "unavailable" };
  }
}

/** Loads something from the server, with a way to load it again. */
export function useLoaded<T>(load: () => Promise<Result<T>>) {
  const [state, setState] = useState<
    | { status: "loading" }
    | { status: "failed"; code: string }
    | { status: "ready"; value: T }
  >({ status: "loading" });
  const [version, setVersion] = useState(0);
  useEffect(() => {
    let current = true;
    void load().then((result) => {
      if (!current) return;
      setState(
        result.ok
          ? { status: "ready", value: result.value }
          : { status: "failed", code: result.code },
      );
    });
    return () => {
      current = false;
    };
  }, [load, version]);
  const reload = useCallback(() => setVersion((n) => n + 1), []);
  return {
    state,
    reload,
    set: (value: T) => setState({ status: "ready", value }),
  };
}
