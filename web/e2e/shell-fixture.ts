import { request, type APIRequestContext } from "@playwright/test";

// Helpers for the browser journeys of the new interface: the instance is installed,
// members are approved and sessions are opened through the API, so each journey starts
// from the screen it tests. Sign-ins are throttled by the server (a burst of ten), hence
// one cached sign-in per person.

type Credentials = { username: string; password: string };
type StorageState = Awaited<ReturnType<APIRequestContext["storageState"]>>;

export const ADMIN: Credentials = {
  username: "administrator",
  password: "synthetic-password-123",
};
export const MEMBER: Credentials = {
  username: "member-user",
  password: "synthetic-password-456",
};

const origin = () => process.env.SNITCH_TEST_URL!;

async function newApi(storageState?: StorageState): Promise<APIRequestContext> {
  return request.newContext({
    baseURL: origin(),
    extraHTTPHeaders: { Origin: origin() },
    storageState,
  });
}

const sessions = new Map<string, Promise<StorageState>>();

/** Cookies of a signed-in person, ready to hand to a browser context. */
export function sessionFor(user: Credentials): Promise<StorageState> {
  let state = sessions.get(user.username);
  if (!state) {
    state = (async () => {
      const api = await newApi();
      const response = await api.post("/api/auth/login", { data: user });
      if (response.status() !== 200) {
        throw new Error(`sign-in failed: ${response.status()}`);
      }
      const cookies = await api.storageState();
      await api.dispose();
      return cookies;
    })();
    sessions.set(user.username, state);
  }
  return state;
}

/** Creates the administrator with the installation code of this run. */
export async function installAdministrator() {
  const api = await newApi();
  const response = await api.post("/api/setup", {
    data: { ...ADMIN, code: process.env.SNITCH_TEST_CODE! },
  });
  if (response.status() !== 201) {
    throw new Error(`installation failed: ${response.status()}`);
  }
  await api.dispose();
}

/** Approves a pending sign-up as the administrator. */
export async function approveSignUp(user: Credentials) {
  const api = await newApi(await sessionFor(ADMIN));
  const me = await (await api.get("/api/auth/me")).json();
  const pending: { id: number; username: string }[] = await (
    await api.get("/api/admin/users")
  ).json();
  const member = pending.find((item) => item.username === user.username);
  if (!member) throw new Error("the sign-up is not pending");
  const approved = await api.post(`/api/admin/users/${member.id}/approve`, {
    headers: { "X-CSRF-Token": me.csrf_token },
  });
  if (approved.status() !== 200) throw new Error("approval failed");
  await api.dispose();
}

/** Registers the member through the API and approves the sign-up. */
export async function createApprovedMember() {
  await createApprovedUser(MEMBER);
}

/** Registers any account through the API and approves the sign-up. */
export async function createApprovedUser(user: Credentials) {
  const guest = await newApi();
  const registered = await guest.post("/api/auth/register", { data: user });
  if (registered.status() !== 201) {
    throw new Error(`registration failed: ${registered.status()}`);
  }
  await guest.dispose();
  await approveSignUp(user);
}

/** The language stored in the account of the given person, read through the API. */
export async function accountLanguage(user: Credentials) {
  const api = await newApi(await sessionFor(user));
  const preferences = await (await api.get("/api/account/preferences")).json();
  await api.dispose();
  return preferences.language as string | null;
}

/** Stores a language in the account of the given person, through the API. */
export async function setAccountLanguage(user: Credentials, language: string) {
  const api = await newApi(await sessionFor(user));
  const me = await (await api.get("/api/auth/me")).json();
  const response = await api.put("/api/account/preferences", {
    data: { language },
    headers: { "X-CSRF-Token": me.csrf_token },
  });
  if (response.status() !== 200) throw new Error("language not stored");
  await api.dispose();
}
