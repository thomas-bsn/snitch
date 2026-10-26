import { useCallback, useEffect, useState } from "react";
import type { components } from "../../../api.gen";
import { api, errorCode, mutation } from "../../api/client";
import { useSession } from "../../session";

export type WorkerPage = components["schemas"]["WorkerPage"];
export type Worker = components["schemas"]["Worker"];
export type WorkerDetail = components["schemas"]["WorkerDetail"];
export type WorkerRegistration = components["schemas"]["WorkerRegistration"];
export type WorkerBundle = components["schemas"]["WorkerBundle"];

/** A call that can fail with the stable code the server gave. */
export type Result<T> = { ok: true; value: T } | { ok: false; code: string };

const failure = (error: unknown): { ok: false; code: string } => ({
  ok: false,
  code: errorCode(error) ?? "unknown",
});

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

export async function fetchWorkers(): Promise<Result<WorkerPage>> {
  const { data, error } = await api.GET("/api/admin/workers");
  return data ? { ok: true, value: data } : failure(error);
}

export async function fetchWorker(id: string): Promise<Result<WorkerDetail>> {
  const { data, error } = await api.GET("/api/admin/workers/{id}", {
    params: { path: { id } },
  });
  return data ? { ok: true, value: data } : failure(error);
}

export type NewWorker = {
  name: string;
  whisperUrl: string;
  /** Only the first Worker sets the library's model and language. */
  transcription?: { model: string; language: string };
};

/** The actions of the Workers screens; each one carries the session's CSRF token. */
export function useWorkerActions() {
  const csrf = useSession().session?.csrf_token;
  return {
    async create(input: NewWorker): Promise<Result<WorkerRegistration>> {
      // Never put the Hugging Face token here: it stays in the browser.
      const { data, error } = await api.POST("/api/admin/workers", {
        params: { header: mutation(csrf) },
        body: {
          name: input.name,
          max_concurrency: 1,
          ...(input.whisperUrl ? { transcription_url: input.whisperUrl } : {}),
          ...(input.transcription
            ? { transcription: input.transcription }
            : {}),
        },
      });
      return data ? { ok: true, value: data } : failure(error);
    },
    async renewToken(id: string): Promise<Result<WorkerRegistration>> {
      const { data, error } = await api.POST("/api/admin/workers/{id}/token", {
        params: { path: { id }, header: mutation(csrf) },
      });
      return data ? { ok: true, value: data } : failure(error);
    },
    async revoke(id: string): Promise<Result<null>> {
      const { error } = await api.DELETE("/api/admin/workers/{id}", {
        params: { path: { id }, header: mutation(csrf) },
      });
      return error ? failure(error) : { ok: true, value: null };
    },
  };
}
