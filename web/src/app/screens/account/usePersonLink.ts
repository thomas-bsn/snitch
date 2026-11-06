import { useCallback, useEffect, useState } from "react";
import type { components } from "../../../api.gen";
import { api, errorCode, mutation } from "../../api/client";
import type { Result } from "../../api/loaded";
import { useSession } from "../../session";

export type PersonLinkRequest = components["schemas"]["PersonLinkRequest"];

/** The link request of the signed-in account, as seen from the account screen. */
export type PersonLink =
  | { phase: "loading" }
  | { phase: "failed"; code: string }
  /** No request, or the last one was approved (the linked person is shown elsewhere). */
  | { phase: "idle" }
  | { phase: "pending"; request: PersonLinkRequest }
  | { phase: "refused"; request: PersonLinkRequest };

/**
 * The account↔Person link request: reading the latest request, asking for a link,
 * cancelling a pending request, and unlinking the account from its person. The
 * server decides; nothing is linked before an administrator approves.
 */
export function usePersonLink(onLinkedChange?: () => void) {
  const { session } = useSession();
  const header = mutation(session?.csrf_token);
  const [link, setLink] = useState<PersonLink>({ phase: "loading" });

  const load = useCallback(async () => {
    try {
      const r = await api.GET("/api/me/person/request");
      if (r.response.status === 404) {
        setLink({ phase: "idle" });
        return;
      }
      if (!r.data) {
        setLink({
          phase: "failed",
          code: errorCode(r.error) ?? "unavailable",
        });
        return;
      }
      if (r.data.status === "pending") {
        setLink({ phase: "pending", request: r.data });
      } else if (r.data.status === "refused") {
        setLink({ phase: "refused", request: r.data });
      } else {
        // Approved: the linked person is shown by useMyVoice, not here.
        setLink({ phase: "idle" });
      }
    } catch {
      setLink({ phase: "failed", code: "unavailable" });
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const request = useCallback(
    async (
      personId: number,
      message: string,
    ): Promise<Result<PersonLinkRequest>> => {
      try {
        const r = await api.POST("/api/me/person/request", {
          params: { header },
          body: { person_id: personId, message },
        });
        if (!r.data)
          return { ok: false, code: errorCode(r.error) ?? "unavailable" };
        await load();
        return { ok: true, value: r.data };
      } catch {
        return { ok: false, code: "unavailable" };
      }
    },
    [header, load],
  );

  const cancel = useCallback(async (): Promise<Result<true>> => {
    try {
      const r = await api.DELETE("/api/me/person/request", {
        params: { header },
      });
      if (r.error)
        return { ok: false, code: errorCode(r.error) ?? "unavailable" };
      await load();
      return { ok: true, value: true };
    } catch {
      return { ok: false, code: "unavailable" };
    }
  }, [header, load]);

  const unlink = useCallback(async (): Promise<Result<true>> => {
    try {
      const r = await api.DELETE("/api/me/person", { params: { header } });
      if (r.error)
        return { ok: false, code: errorCode(r.error) ?? "unavailable" };
      await load();
      onLinkedChange?.();
      return { ok: true, value: true };
    } catch {
      return { ok: false, code: "unavailable" };
    }
  }, [header, load, onLinkedChange]);

  return { link, request, cancel, unlink, reload: load };
}
