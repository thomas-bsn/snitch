import { useCallback, useEffect, useState } from "react";
import {
  api,
  errorCode,
  mutation,
  type Person,
  type VoiceProfileDeletion,
} from "../../api/client";
import { fetchPersonImpact, type PersonImpact } from "../../people/peopleApi";
import { useSession, type Outcome } from "../../session";

export type MyVoice =
  | { phase: "loading" }
  | { phase: "failed"; code: string }
  /** No Person is linked to the account: Snitch holds no voice profile for it. */
  | { phase: "none" }
  | {
      phase: "ready";
      person: Person;
      /** Clips visible to the account where the Person is identified. */
      clipTotal: number;
      /** What deleting the voice profile erases; null once deleted or when unreadable. */
      impact: PersonImpact | null;
      impactFailed: boolean;
    };

/** The Person linked to the signed-in account, the clips where they are identified, and the deletion of the voice profile. */
export function useMyVoice() {
  const { session } = useSession();
  const csrfToken = session?.csrf_token;
  const [voice, setVoice] = useState<MyVoice>({ phase: "loading" });

  const load = useCallback(async () => {
    try {
      const mine = await api.GET("/api/me/person");
      if (mine.response.status === 404) {
        setVoice({ phase: "none" });
        return;
      }
      if (!mine.data) {
        setVoice({
          phase: "failed",
          code: errorCode(mine.error) ?? "unavailable",
        });
        return;
      }
      const person = mine.data;
      const deleted = person.voice_profile_deleted_at !== null;
      const [list, impact] = await Promise.all([
        api.GET("/api/clips", {
          params: { query: { person_id: person.id, page_size: 1 } },
        }),
        deleted ? undefined : fetchPersonImpact(person.id),
      ]);
      setVoice({
        phase: "ready",
        person,
        clipTotal: list.data?.total ?? 0,
        impact: impact?.ok ? impact.value : null,
        impactFailed: impact !== undefined && !impact.ok,
      });
    } catch {
      setVoice({ phase: "failed", code: "unavailable" });
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const erase = useCallback(
    async (person: Person): Promise<Outcome<VoiceProfileDeletion>> => {
      try {
        const result = await api.DELETE("/api/persons/{id}/voice-profile", {
          params: { header: mutation(csrfToken), path: { id: person.id } },
        });
        if (!result.data)
          return { ok: false, code: errorCode(result.error) ?? "unavailable" };
        await load();
        return { ok: true, value: result.data };
      } catch {
        return { ok: false, code: "unavailable" };
      }
    },
    [csrfToken, load],
  );

  return { voice, erase, reload: load };
}
