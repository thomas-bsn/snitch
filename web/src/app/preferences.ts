// Preferences remembered by the browser. Storage can be blocked or empty (private
// window, blocked site data), so every access is guarded and the defaults apply.
// Keep the keys and values in sync with public/preferences-init.js.

export const THEMES = ["light", "dark", "system"] as const;
export type Theme = (typeof THEMES)[number];
export const DEFAULT_THEME: Theme = "light";

export const LANGUAGES = ["en", "fr"] as const;
export type Language = (typeof LANGUAGES)[number];
export const DEFAULT_LANGUAGE: Language = "en";

const THEME_KEY = "snitch-ui-theme";
const LANGUAGE_KEY = "snitch-ui-language";

function read<T extends string>(
  key: string,
  allowed: readonly T[],
): T | undefined {
  try {
    const value = localStorage.getItem(key);
    return allowed.find((candidate) => candidate === value);
  } catch {
    return undefined;
  }
}

function write(key: string, value: string) {
  try {
    localStorage.setItem(key, value);
  } catch {
    // Not remembered, still applied for this visit.
  }
}

export function isLanguage(value: unknown): value is Language {
  return LANGUAGES.some((language) => language === value);
}

export const storedTheme = () => read(THEME_KEY, THEMES);
export const storeTheme = (theme: Theme) => write(THEME_KEY, theme);
export const storedLanguage = () => read(LANGUAGE_KEY, LANGUAGES);
export const storeLanguage = (language: Language) =>
  write(LANGUAGE_KEY, language);
