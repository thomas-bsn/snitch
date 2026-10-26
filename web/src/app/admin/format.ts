import { useMemo } from "react";
import { useI18n } from "../i18n";

/** Formats an ISO date for the current language. */
export function useFormatDate() {
  const { language } = useI18n();
  return useMemo(() => {
    const format = new Intl.DateTimeFormat(language, {
      dateStyle: "medium",
      timeStyle: "short",
    });
    return (iso: string) => format.format(new Date(iso));
  }, [language]);
}
