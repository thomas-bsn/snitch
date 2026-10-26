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
  DEFAULT_LANGUAGE,
  storeLanguage,
  storedLanguage,
  type Language,
} from "../preferences";
import { en } from "./en";
import { fr } from "./fr";
import {
  translate,
  translateError,
  type MessageKey,
  type Messages,
  type Params,
} from "./messages";

const catalogs: Record<Language, Messages> = { en, fr };

type I18nContextValue = {
  language: Language;
  /** Applies a language to this browser and remembers it there. */
  setLanguage: (language: Language) => void;
  t: (key: MessageKey, params?: Params) => string;
  /** Text for a stable server error code. */
  error: (code: string) => string;
};

const I18nContext = createContext<I18nContextValue | undefined>(undefined);

export function I18nProvider({ children }: { children: ReactNode }) {
  const [language, setLanguageState] = useState<Language>(
    () => storedLanguage() ?? DEFAULT_LANGUAGE,
  );

  useEffect(() => {
    document.documentElement.setAttribute("lang", language);
  }, [language]);

  const setLanguage = useCallback((next: Language) => {
    storeLanguage(next);
    setLanguageState(next);
  }, []);

  const value = useMemo<I18nContextValue>(() => {
    const messages = catalogs[language];
    return {
      language,
      setLanguage,
      t: (key, params) => translate(messages, key, params),
      error: (code) => translateError(messages, code),
    };
  }, [language, setLanguage]);

  return <I18nContext.Provider value={value}>{children}</I18nContext.Provider>;
}

export function useI18n(): I18nContextValue {
  const value = useContext(I18nContext);
  if (!value) throw new Error("useI18n needs an I18nProvider");
  return value;
}
