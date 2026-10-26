import { useCallback, useEffect, useState } from "react";
import type { components } from "../../api.gen";
import { api, errorCode, mutation } from "../api/client";
import { useSession } from "../session";

export type Person = components["schemas"]["Person"];
export type PersonVoice = components["schemas"]["PersonVoice"];
export type PersonEvent = components["schemas"]["PersonEvent"];
export type PersonCorrectionResult =
  components["schemas"]["PersonCorrectionResult"];
export type Verification = components["schemas"]["VoiceVerification"];
export type VerificationCase = components["schemas"]["VoiceVerificationCase"];
export type VerificationCounts =
  components["schemas"]["VoiceVerificationCounts"];
export type DecisionLevel = components["schemas"]["DecisionLevel"];
export type VerificationFilter = "all" | "unknown" | "ambiguous";

/** The person a case is settled for: an existing one, or a new one by name. */
export type Choice = { personId: number } | { newName: string };

export type Result<T> = { ok: true; value: T } | { ok: false; code: string };

const failure = (error: unknown): { ok: false; code: string } => ({
  ok: false,
  code: errorCode(error) ?? "unknown",
});

/** The call that failed to answer at all is reported like an unknown error. */
async function guard<T>(call: () => Promise<Result<T>>): Promise<Result<T>> {
  try {
    return await call();
  } catch {
    return { ok: false, code: "unknown" };
  }
}

/** Loads something from the server, with a way to load it again quietly. */
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
  return { state, reload };
}

export const fetchPersons = () =>
  guard(async () => {
    const { data, error } = await api.GET("/api/persons");
    return data ? { ok: true, value: data.items } : failure(error);
  });

export const fetchPersonVoices = (id: number) =>
  guard(async () => {
    const { data, error } = await api.GET("/api/persons/{id}/voices", {
      params: { path: { id } },
    });
    return data ? { ok: true, value: data.items } : failure(error);
  });

export const fetchPersonEvents = () =>
  guard(async () => {
    const { data, error } = await api.GET("/api/person-events");
    return data ? { ok: true, value: data.items } : failure(error);
  });

export const fetchVerification = (
  level: VerificationFilter,
  page: number,
  pageSize: number,
) =>
  guard(async () => {
    const { data, error } = await api.GET("/api/voice-verification", {
      params: { query: { level, page, page_size: pageSize } },
    });
    return data ? { ok: true, value: data } : failure(error);
  });

/** The calls that change something carry the session's CSRF token. */
export function usePersonActions() {
  const { session } = useSession();
  const csrf = session?.csrf_token;
  const header = mutation(csrf);
  return {
    create: (name: string) =>
      guard<Person>(async () => {
        const { data, error } = await api.POST("/api/persons", {
          params: { header },
          body: { display_name: name },
        });
        return data ? { ok: true, value: data } : failure(error);
      }),
    merge: (id: number, targetId: number) =>
      guard<PersonCorrectionResult>(async () => {
        const { data, error } = await api.POST("/api/persons/{id}/merge", {
          params: { path: { id }, header },
          body: { target_person_id: targetId },
        });
        return data ? { ok: true, value: data } : failure(error);
      }),
    remove: (id: number) =>
      guard<PersonCorrectionResult>(async () => {
        const { data, error } = await api.DELETE("/api/persons/{id}", {
          params: { path: { id }, header },
        });
        return data ? { ok: true, value: data } : failure(error);
      }),
    separate: (id: number, localSpeakerId: number) =>
      guard<PersonCorrectionResult>(async () => {
        const { data, error } = await api.POST("/api/persons/{id}/separation", {
          params: { path: { id }, header },
          body: { local_speaker_ids: [localSpeakerId] },
        });
        return data ? { ok: true, value: data } : failure(error);
      }),
    /** Settling a case is the ordinary human validation of the voice. */
    validate: (localSpeakerId: number, choice: Choice) =>
      guard<{ personName: string | null }>(async () => {
        const { data, error } = await api.POST(
          "/api/local-speakers/{id}/validation",
          {
            params: { path: { id: localSpeakerId }, header },
            body:
              "personId" in choice
                ? { person_id: choice.personId }
                : { new_person_name: choice.newName },
          },
        );
        if (data)
          return {
            ok: true,
            value: { personName: data.person?.display_name ?? null },
          };
        // The first validation stands: tell who won.
        const existing = (
          error as { attribution?: { person?: { display_name: string } } }
        )?.attribution?.person?.display_name;
        return existing
          ? { ok: false, code: `already_validated:${existing}` }
          : failure(error);
      }),
  };
}

export const excerptUrl = (localSpeakerId: number) =>
  `/api/local-speakers/${localSpeakerId}/excerpt`;
