import type { components } from "../../../api.gen";
import { api, errorCode, mutation } from "../../api/client";
import type { Result } from "../../api/loaded";
import { useSession } from "../../session";

export type WorkerPage = components["schemas"]["WorkerPage"];
export type Worker = components["schemas"]["Worker"];
export type WorkerDetail = components["schemas"]["WorkerDetail"];
export type WorkerRegistration = components["schemas"]["WorkerRegistration"];
export type WorkerBundle = components["schemas"]["WorkerBundle"];
export type WorkerRestartHistory =
  components["schemas"]["WorkerRestartHistory"];
export type WorkerLogPage = components["schemas"]["WorkerLogPage"];
export type WorkerLogLine = components["schemas"]["WorkerLogLine"];
export type WorkerControlState = components["schemas"]["WorkerControlState"];
export type WorkerControlAction =
  components["schemas"]["WorkerControlInput"]["action"];

const failure = (error: unknown): { ok: false; code: string } => ({
  ok: false,
  code: errorCode(error) ?? "unknown",
});

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

export async function fetchWorkerRestartHistory(
  id: string,
): Promise<Result<WorkerRestartHistory>> {
  const { data, error } = await api.GET("/api/admin/workers/{id}/restart", {
    params: { path: { id } },
  });
  return data ? { ok: true, value: data } : failure(error);
}

/** One page of the log of the Worker's current run: the newest, or after/before a line. */
export async function fetchWorkerLogs(
  id: string,
  query: { after?: number; before?: number; limit?: number } = {},
): Promise<Result<WorkerLogPage>> {
  const { data, error } = await api.GET("/api/admin/workers/{id}/logs", {
    params: { path: { id }, query },
  });
  return data ? { ok: true, value: data } : failure(error);
}

export async function fetchWorkerControl(
  id: string,
): Promise<Result<WorkerControlState>> {
  const { data, error } = await api.GET("/api/admin/workers/{id}/control", {
    params: { path: { id } },
  });
  return data ? { ok: true, value: data } : failure(error);
}

export type NewWorker = {
  name: string;
  managedRestart?: boolean;
  allowInsecureInstall?: boolean;
};

/** The actions of the Workers screens; each one carries the session's CSRF token. */
export function useWorkerActions() {
  const csrf = useSession().session?.csrf_token;
  return {
    async create(input: NewWorker): Promise<Result<WorkerRegistration>> {
      // Never put the Hugging Face token here: it stays in the browser.
      const { data, error } = await api.POST("/api/admin/workers", {
        params: {
          header: mutation(csrf),
          query: input.allowInsecureInstall
            ? { allow_insecure_install: true }
            : undefined,
        },
        body: {
          name: input.name,
          max_concurrency: 1,
          managed_restart: input.managedRestart ?? false,
        },
      });
      return data ? { ok: true, value: data } : failure(error);
    },
    async renewToken(
      id: string,
      allowInsecureInstall = false,
    ): Promise<Result<WorkerRegistration>> {
      const { data, error } = await api.POST("/api/admin/workers/{id}/token", {
        params: {
          path: { id },
          header: mutation(csrf),
          query: allowInsecureInstall
            ? { allow_insecure_install: true }
            : undefined,
        },
      });
      return data ? { ok: true, value: data } : failure(error);
    },
    /** Deletes the Worker outright, active or revoked: no revocation step, no reason. */
    async remove(id: string): Promise<Result<null>> {
      const { error } = await api.DELETE("/api/admin/workers/{id}", {
        params: { path: { id }, header: mutation(csrf) },
      });
      return error ? failure(error) : { ok: true, value: null };
    },
    /** Pause or resume: a state the Worker reads at its next check, not a command sent to it. */
    async control(
      id: string,
      action: WorkerControlAction,
    ): Promise<Result<WorkerControlState>> {
      const { data, error } = await api.POST(
        "/api/admin/workers/{id}/control",
        {
          params: { path: { id }, header: mutation(csrf) },
          body: { action },
        },
      );
      return data ? { ok: true, value: data } : failure(error);
    },
    async restart(id: string, reason: string): Promise<Result<unknown>> {
      const { data, error } = await api.POST(
        "/api/admin/workers/{id}/restart",
        {
          params: { path: { id }, header: mutation(csrf) },
          body: { reason: reason.trim(), idempotency_key: crypto.randomUUID() },
        },
      );
      return data ? { ok: true, value: data } : failure(error);
    },
  };
}
