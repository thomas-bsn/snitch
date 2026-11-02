import type { components } from "../../api.gen";
import { api, errorCode, mutation } from "../api/client";
import type { Result } from "../api/loaded";
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
export type VoiceGroup = components["schemas"]["VoiceGroup"];
export type VoiceGroupMember = components["schemas"]["VoiceGroupMember"];
export type VoiceGroupNaming = components["schemas"]["VoiceGroupNaming"];
export type DecisionLevel = components["schemas"]["DecisionLevel"];
export type VerificationFilter = "all" | "unknown" | "ambiguous";

/** The person a case is settled for: an existing one, or a new one by name. */
export type Choice = { personId: number } | { newName: string };

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

export const fetchPersons = () =>
  guard(async () => {
    const { data, error } = await api.GET("/api/persons");
    return data ? { ok: true, value: data.items } : failure(error);
  });

/** A page of Persons by name; the search and the paging are done by the server. */
export type PersonPage = { items: Person[]; total: number };

export const fetchPersonPage = (
  query: string,
  page: number,
  pageSize: number,
) =>
  guard<PersonPage>(async () => {
    const { data, error } = await api.GET("/api/persons", {
      params: { query: { q: query, page, page_size: pageSize } },
    });
    return data
      ? { ok: true, value: { items: data.items, total: data.total ?? 0 } }
      : failure(error);
  });

export const fetchPerson = (id: number) =>
  guard<Person>(async () => {
    const { data, error } = await api.GET("/api/persons/{id}", {
      params: { path: { id } },
    });
    return data ? { ok: true, value: data } : failure(error);
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

/** The proposed groups of unknown voices (two voices or more), most frequent first. */
export const fetchVoiceGroups = (page: number, pageSize: number) =>
  guard(async () => {
    const { data, error } = await api.GET("/api/voice-groups", {
      params: {
        query: { proposed: true, sort: "clips", page, page_size: pageSize },
      },
    });
    return data ? { ok: true, value: data } : failure(error);
  });

/**
 * The Voice group that holds a voice today, or null when it is in none (named, no longer
 * unknown, or its clip is no longer visible). A group has no stable id (ADR 0024), so the
 * groups are read, isolated voices included, until the voice is found.
 */
export const fetchVoiceGroupOf = (localSpeakerId: number) =>
  guard<VoiceGroup | null>(async () => {
    const size = 100;
    for (let page = 1; ; page++) {
      const { data, error } = await api.GET("/api/voice-groups", {
        params: {
          query: { proposed: false, sort: "clips", page, page_size: size },
        },
      });
      if (!data) return failure(error);
      const found = data.items.find((group) =>
        group.members.some((m) => m.local_speaker_id === localSpeakerId),
      );
      if (found) return { ok: true, value: found };
      if (page * size >= data.total) return { ok: true, value: null };
    }
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

/** Leaving a voice out of its group, putting it back, naming what is left. */
export function useVoiceGroupActions() {
  const { session } = useSession();
  const header = mutation(session?.csrf_token);
  return {
    exclude: (localSpeakerId: number) =>
      guard<true>(async () => {
        const { data, error } = await api.PUT(
          "/api/local-speakers/{id}/group-exclusion",
          { params: { path: { id: localSpeakerId }, header } },
        );
        return data ? { ok: true, value: true } : failure(error);
      }),
    restore: (localSpeakerId: number) =>
      guard<true>(async () => {
        const { data, error } = await api.DELETE(
          "/api/local-speakers/{id}/group-exclusion",
          { params: { path: { id: localSpeakerId }, header } },
        );
        return data ? { ok: true, value: true } : failure(error);
      }),
    /**
     * Names exactly the voices that were listened to. The server refuses, and writes
     * nothing, when the group is no longer exactly that set.
     */
    name: (groupId: number, localSpeakerIds: number[], choice: Choice) =>
      guard<VoiceGroupNaming>(async () => {
        const { data, error } = await api.POST(
          "/api/voice-groups/{id}/naming",
          {
            params: { path: { id: groupId }, header },
            body: {
              local_speaker_ids: localSpeakerIds,
              ...("personId" in choice
                ? { person_id: choice.personId }
                : { new_person_name: choice.newName }),
            },
          },
        );
        return data ? { ok: true, value: data } : failure(error);
      }),
  };
}

export const excerptUrl = (localSpeakerId: number) =>
  `/api/local-speakers/${localSpeakerId}/excerpt`;
