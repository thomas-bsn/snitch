import createClient from "openapi-fetch";
import type { components, paths } from "../../api.gen";
import { markUnreachable } from "./connection";

export type Session = components["schemas"]["Session"];
export type User = components["schemas"]["User"];
export type Setup = components["schemas"]["Setup"];
export type Person = components["schemas"]["Person"];
export type VoiceProfileDeletion =
  components["schemas"]["VoiceProfileDeletion"];

/**
 * A request with no usable answer (network failure, or a gateway page that is not the
 * API's JSON) means the server is unreachable. Answers carrying an error code are not:
 * they are translated and shown where they happened.
 */
async function guardedFetch(request: Request): Promise<Response> {
  let response: Response;
  try {
    response = await fetch(request);
  } catch (cause) {
    markUnreachable();
    throw cause;
  }
  const json = response.headers.get("content-type")?.includes("json");
  if (response.status >= 502 && response.status <= 504 && !json) {
    markUnreachable();
  }
  return response;
}

export const api = createClient<paths>({ fetch: guardedFetch });

/** Headers every browser mutation carries (the browser sets Origin itself). */
export function mutation(csrfToken?: string) {
  return { Origin: location.origin, "X-CSRF-Token": csrfToken ?? "" };
}

export const origin = () => ({ Origin: location.origin });

/** The stable code of an error answer, when the server sent one. */
export function errorCode(error: unknown): string | undefined {
  if (typeof error === "object" && error !== null && "error" in error) {
    const code = (error as { error: unknown }).error;
    if (typeof code === "string") return code;
  }
  return undefined;
}
