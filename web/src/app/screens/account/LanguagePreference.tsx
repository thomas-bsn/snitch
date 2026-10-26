import { useId, useState } from "react";
import { StatusNotice } from "../../components/Alert";
import { useI18n } from "../../i18n";
import { LANGUAGES, isLanguage } from "../../preferences";
import { useSession } from "../../session";

/** The language of the interface, kept in the account (and in this browser). */
export function LanguagePreference() {
  const { t, language } = useI18n();
  const { chooseLanguage } = useSession();
  const id = useId();
  const hintId = `${id}-hint`;
  const [saved, setSaved] = useState<boolean>();

  return (
    <div className="flex max-w-sm flex-col gap-1.5">
      <label htmlFor={id} className="font-medium">
        {t("account.profile.language")}
      </label>
      <select
        id={id}
        value={language}
        aria-describedby={hintId}
        onChange={async (event) => {
          const next = event.target.value;
          if (!isLanguage(next)) return;
          setSaved(await chooseLanguage(next));
        }}
        className="h-11 rounded-lg border border-line-strong bg-surface px-3 text-ink md:h-10"
      >
        {LANGUAGES.map((code) => (
          <option key={code} value={code} lang={code}>
            {t(`language.${code}`)}
          </option>
        ))}
      </select>
      <p id={hintId} className="text-xs text-mute">
        {t("account.profile.languageHint")}
      </p>
      {saved !== undefined && (
        <StatusNotice>
          {saved
            ? t("account.profile.languageSaved")
            : t("account.profile.languageNotSaved")}
        </StatusNotice>
      )}
    </div>
  );
}
