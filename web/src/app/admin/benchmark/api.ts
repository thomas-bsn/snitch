import type { components } from "../../../api.gen";
import { api, errorCode } from "../../api/client";
import type { Result } from "../../api/loaded";

export type Dataset = components["schemas"]["BenchmarkDataset"];
export type Clip = components["schemas"]["BenchmarkClip"];
export type Model = components["schemas"]["BenchmarkModel"];
export type Run = components["schemas"]["BenchmarkRun"];
export type RunResult = components["schemas"]["BenchmarkResult"];
export type Metrics = components["schemas"]["BenchmarkMetrics"];

export type Stage = "transcription" | "diarization" | "fingerprint" | "e2e";

/** The stages a corpus can list, in the order of their importance. */
export const STAGES: Stage[] = [
  "transcription",
  "diarization",
  "fingerprint",
  "e2e",
];

/** The stages a run can execute today; the others show as coming soon, never as a dead button. */
export const EXECUTABLE_STAGES: ModelStage[] = ["fingerprint", "transcription"];

/** The stages whose models the server can list. */
export type ModelStage = "fingerprint" | "transcription";

/** Turns a call into the Result the loading hook expects: data, or the server's stable error code. */
export async function settle<T>(
  call: () => Promise<{ data?: T; error?: unknown }>,
): Promise<Result<T>> {
  try {
    const result = await call();
    return result.data !== undefined
      ? { ok: true, value: result.data }
      : { ok: false, code: errorCode(result.error) ?? "unknown" };
  } catch {
    return { ok: false, code: "unavailable" };
  }
}

export const loadDatasets = () =>
  settle(() => api.GET("/api/admin/benchmarks/datasets"));

export const loadClips = () =>
  settle(() => api.GET("/api/admin/benchmarks/clips"));

export const loadModels = (stage: ModelStage) =>
  settle(() =>
    api.GET("/api/admin/benchmarks/models", {
      params: { query: { stage } },
    }),
  );

export const loadRuns = () =>
  settle(() => api.GET("/api/admin/benchmarks/runs"));

export const loadRun = (id: number) =>
  settle(() =>
    api.GET("/api/admin/benchmarks/runs/{id}", { params: { path: { id } } }),
  );

/** A run still waiting for its tasks: the screen reads it again until it settles. */
export const isRunning = (run: Pick<Run, "status">) => run.status === "running";

export const loadFingerprintModels = () => loadModels("fingerprint");
export const loadTranscriptionModels = () => loadModels("transcription");
