import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react";
import {
  api,
  errorCode,
  mutation,
  origin,
  type Session,
  type Setup,
  type User,
} from "./api/client";
import { markReachable } from "./api/connection";
import { useI18n } from "./i18n";
import { isLanguage, storedLanguage, type Language } from "./preferences";

type Boot =
  | { phase: "loading" }
  | { phase: "failed" }
  | { phase: "ready"; setup: Setup; session: Session | null };

/** Result of an action: success, or the stable error code the server gave. */
export type Outcome<T = undefined> =
  | ({ ok: true } & (T extends undefined ? object : { value: T }))
  | { ok: false; code: string };

type SessionContextValue = {
  phase: Boot["phase"];
  setup: Setup | null;
  session: Session | null;
  user: User | null;
  /** Loads the installation state and the session again (also the "try again" action). */
  reload: () => Promise<void>;
  login: (username: string, password: string) => Promise<Outcome>;
  register: (
    username: string,
    password: string,
  ) => Promise<Outcome<"active" | "pending">>;
  logout: () => Promise<void>;
  /**
  /**
   * Creates the first account (the Super-administrator) with the installation code. It opens
   * the session on the server but does not reload: the wizard reloads once it has moved on.
   */
  completeSetup: (
    code: string,
    username: string,
    password: string,
  ) => Promise<Outcome>;
  /**
   * Applies a language here and, when signed in, remembers it in the account. Answers
   * whether the account kept it (false: only this browser did).
   */
  chooseLanguage: (language: Language) => Promise<boolean>;
  startDiscord: () => Promise<Outcome>;
  /** Starts linking the Discord identity to the signed-in account (leaves for Discord). */
  linkDiscord: () => Promise<Outcome>;
  /** Removes Discord after the API confirms a local password keeps the account usable. */
  unlinkDiscord: () => Promise<Outcome>;
};

const SessionContext = createContext<SessionContextValue | undefined>(
  undefined,
);

async function fetchBoot(): Promise<Boot> {
  try {
    const setup = await api.GET("/api/setup");
    if (!setup.data) return { phase: "failed" };
    const me = await api.GET("/api/auth/me");
    if (me.data) return { phase: "ready", setup: setup.data, session: me.data };
    if (me.response.status === 401)
      return { phase: "ready", setup: setup.data, session: null };
    return { phase: "failed" };
  } catch {
    return { phase: "failed" };
  }
}

export function SessionProvider({ children }: { children: ReactNode }) {
  const { setLanguage } = useI18n();
  const [boot, setBoot] = useState<Boot>({ phase: "loading" });

  const reload = useCallback(async () => {
    markReachable();
    setBoot((current) =>
      current.phase === "ready" ? current : { phase: "loading" },
    );
    setBoot(await fetchBoot());
  }, []);

  useEffect(() => {
    void reload();
  }, [reload]);

  const session = boot.phase === "ready" ? boot.session : null;
  const setup = boot.phase === "ready" ? boot.setup : null;
  const user = session?.user ?? null;

  const rememberLanguage = useCallback(
    async (language: Language, csrfToken: string): Promise<boolean> => {
      try {
        const result = await api.PUT("/api/account/preferences", {
          params: { header: mutation(csrfToken) },
          body: { language },
        });
        if (!result.data) return false;
        setBoot((current) =>
          current.phase === "ready" && current.session
            ? {
                ...current,
                session: {
                  ...current.session,
                  user: { ...current.session.user, language },
                },
              }
            : current,
        );
        return true;
      } catch {
        // Not saved in the account this time: the browser keeps the choice.
        return false;
      }
    },
    [],
  );

  // The account's language wins on every device. An account that never chose one takes
  // the choice this browser remembers (made on the sign-in page, for example).
  const accountLanguage = user?.language;
  const csrfToken = session?.csrf_token;
  const signedIn = user !== null;
  useEffect(() => {
    if (!signedIn || !csrfToken) return;
    if (isLanguage(accountLanguage)) {
      setLanguage(accountLanguage);
      return;
    }
    const chosen = storedLanguage();
    if (chosen) void rememberLanguage(chosen, csrfToken);
  }, [signedIn, accountLanguage, csrfToken, setLanguage, rememberLanguage]);

  const chooseLanguage = useCallback(
    async (language: Language): Promise<boolean> => {
      setLanguage(language);
      return csrfToken ? rememberLanguage(language, csrfToken) : false;
    },
    [csrfToken, setLanguage, rememberLanguage],
  );

  const login = useCallback(
    async (username: string, password: string): Promise<Outcome> => {
      try {
        const result = await api.POST("/api/auth/login", {
          params: { header: origin() },
          body: { username, password },
        });
        if (result.error)
          return { ok: false, code: errorCode(result.error) ?? "unavailable" };
        await reload();
        return { ok: true };
      } catch {
        return { ok: false, code: "unavailable" };
      }
    },
    [reload],
  );

  const register = useCallback(
    async (
      username: string,
      password: string,
    ): Promise<Outcome<"active" | "pending">> => {
      try {
        const result = await api.POST("/api/auth/register", {
          params: { header: origin() },
          body: { username, password },
        });
        if (result.error || !result.data)
          return { ok: false, code: errorCode(result.error) ?? "unavailable" };
        return {
          ok: true,
          value: result.data.status === "pending" ? "pending" : "active",
        };
      } catch {
        return { ok: false, code: "unavailable" };
      }
    },
    [],
  );

  const logout = useCallback(async () => {
    try {
      await api.POST("/api/auth/logout", {
        params: { header: mutation(csrfToken) },
      });
    } catch {
      // The gate shows the unreachable message; the session is still there.
      return;
    }
    await reload();
  }, [csrfToken, reload]);

  const completeSetup = useCallback(
    async (
      code: string,
      username: string,
      password: string,
    ): Promise<Outcome> => {
      try {
        const result = await api.POST("/api/setup", {
          params: { header: origin() },
          body: { code, username, password },
        });
        if (result.error)
          return { ok: false, code: errorCode(result.error) ?? "unavailable" };
        return { ok: true };
      } catch {
        return { ok: false, code: "unavailable" };
      }
    },
    [],
  );

  const startDiscord = useCallback(async (): Promise<Outcome> => {
    try {
      const result = await api.POST("/api/auth/discord/start", {
        params: { header: origin() },
      });
      if (!result.data)
        return { ok: false, code: errorCode(result.error) ?? "unavailable" };
      location.assign(result.data.url);
      return { ok: true };
    } catch {
      return { ok: false, code: "unavailable" };
    }
  }, []);

  const linkDiscord = useCallback(async (): Promise<Outcome> => {
    try {
      const result = await api.POST("/api/auth/discord/link", {
        params: { header: mutation(csrfToken) },
      });
      if (!result.data)
        return { ok: false, code: errorCode(result.error) ?? "unavailable" };
      location.assign(result.data.url);
      return { ok: true };
    } catch {
      return { ok: false, code: "unavailable" };
    }
  }, [csrfToken]);

  const unlinkDiscord = useCallback(async (): Promise<Outcome> => {
    try {
      const result = await api.DELETE("/api/account/discord", {
        params: { header: mutation(csrfToken) },
      });
      if (!result.data)
        return { ok: false, code: errorCode(result.error) ?? "unavailable" };
      await reload();
      return { ok: true };
    } catch {
      return { ok: false, code: "unavailable" };
    }
  }, [csrfToken, reload]);

  const value = useMemo<SessionContextValue>(
    () => ({
      phase: boot.phase,
      setup,
      session,
      user,
      reload,
      login,
      register,
      logout,
      completeSetup,
      chooseLanguage,
      startDiscord,
      linkDiscord,
      unlinkDiscord,
    }),
    [
      boot.phase,
      setup,
      session,
      user,
      reload,
      login,
      register,
      logout,
      completeSetup,
      chooseLanguage,
      startDiscord,
      linkDiscord,
      unlinkDiscord,
    ],
  );

  return (
    <SessionContext.Provider value={value}>{children}</SessionContext.Provider>
  );
}

export function useSession(): SessionContextValue {
  const value = useContext(SessionContext);
  if (!value) throw new Error("useSession needs a SessionProvider");
  return value;
}
