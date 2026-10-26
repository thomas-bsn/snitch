import { useI18n } from "../i18n";

/** Text for the code of a failed call; the first validation that won is named. */
export function useFailureText() {
  const { t, error } = useI18n();
  return (code: string) => {
    const [name, winner] = code.split(/:(.*)/s);
    return name === "already_validated" && winner
      ? t("people.verification.alreadyValidated", { person: winner })
      : error(code);
  };
}
