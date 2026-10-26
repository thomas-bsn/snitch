import { useCallback } from "react";
import type { components } from "../../api.gen";
import { api, errorCode, mutation, origin } from "../api/client";
import { useSession } from "../session";
import type { Result } from "../api/loaded";

export type SetupProgress = components["schemas"]["SetupProgress"];
export type SetupStep = components["schemas"]["SetupStepState"]["step"];
export type SetupStepState = components["schemas"]["SetupStepState"]["state"];

const failure = (error: unknown): { ok: false; code: string } => ({
  ok: false,
  code: errorCode(error) ?? "unknown",
});

export async function fetchSetupProgress(): Promise<Result<SetupProgress>> {
  const { data, error } = await api.GET("/api/admin/setup-progress");
  return data ? { ok: true, value: data } : failure(error);
}

/** The state of one step in a progress report. */
export function stepState(
  progress: SetupProgress,
  step: SetupStep,
): SetupStepState {
  return progress.steps.find((item) => item.step === step)?.state ?? "todo";
}

/** True while an Administrator still has optional steps to finish and kept the banner. */
export function hasRemainingSteps(progress: SetupProgress): boolean {
  return progress.steps.some((item) => item.state !== "done");
}

/** Checks the installation code without using it up. */
export async function checkInstallationCode(
  code: string,
): Promise<Result<undefined>> {
  try {
    const { response, error } = await api.POST("/api/setup/code", {
      params: { header: origin() },
      body: { code },
    });
    return response.ok
      ? { ok: true, value: undefined }
      : { ok: false, code: errorCode(error) ?? "unavailable" };
  } catch {
    return { ok: false, code: "unavailable" };
  }
}

/** The calls that change the progress; each one carries the session's CSRF token. */
export function useSetupActions() {
  const csrf = useSession().session?.csrf_token;
  const record = useCallback(
    async (
      step: SetupStep,
      state: "done" | "skipped",
    ): Promise<Result<SetupProgress>> => {
      try {
        const { data, error } = await api.PUT("/api/admin/setup-progress", {
          params: { header: mutation(csrf) },
          body: { step, state },
        });
        return data ? { ok: true, value: data } : failure(error);
      } catch {
        return { ok: false, code: "unavailable" };
      }
    },
    [csrf],
  );
  const dismiss = useCallback(async (): Promise<Result<SetupProgress>> => {
    try {
      const { data, error } = await api.POST(
        "/api/admin/setup-progress/dismiss",
        { params: { header: mutation(csrf) } },
      );
      return data ? { ok: true, value: data } : failure(error);
    } catch {
      return { ok: false, code: "unavailable" };
    }
  }, [csrf]);
  return { record, dismiss };
}
