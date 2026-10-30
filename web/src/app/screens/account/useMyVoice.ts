import { useCallback, useEffect, useState } from "react";
import {
  api,
  errorCode,
  mutation,
  type Person,
  type VoiceProfileDeletion,
} from "../../api/client";
import { useSession, type Outcome } from "../../session";

export type MyClip = { id: number; original_name: string; recorded_at: string };

export type MyVoice =
  | { phase: "loading" }
  | { phase: "failed"; code: string }
  /** No Person is linked to the account: Snitch holds no voice profile for it. */
  | { phase: "none" }
  | { phase: "ready"; person: Person; clips: MyClip[] };

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
      const list = await api.GET("/api/clips", {
        params: { query: { person_id: person.id, page_size: 100 } },
      });
      setVoice({ phase: "ready", person, clips: list.data?.items ?? [] });
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
