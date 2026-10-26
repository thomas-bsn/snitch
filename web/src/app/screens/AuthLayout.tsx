import { useId, type ReactNode } from "react";
import { Brand } from "../components/Logo";
import { useI18n } from "../i18n";
import { LANGUAGES, isLanguage } from "../preferences";
import { useSession } from "../session";

/** Language choice available before signing in (kept in the browser). */
export function LanguageSelect() {
  const { t, language } = useI18n();
  const { chooseLanguage } = useSession();
  const id = useId();
  return (
    <div className="flex items-center gap-2 text-sm text-mute">
      <label htmlFor={id}>{t("auth.languageChoice")}</label>
      <select
        id={id}
        value={language}
        onChange={(event) => {
          const next = event.target.value;
          if (isLanguage(next)) void chooseLanguage(next);
        }}
        className="h-9 rounded-md border border-line-strong bg-surface px-2 text-ink"
      >
        {LANGUAGES.map((code) => (
          <option key={code} value={code}>
            {t(`language.${code}`)}
          </option>
        ))}
      </select>
    </div>
  );
}

/** Centered card shared by the sign-in, sign-up and installation screens. */
export function AuthLayout({
  title,
  lead,
  children,
}: {
  title: string;
  lead?: string;
  children: ReactNode;
}) {
  return (
    <main className="flex min-h-dvh items-center justify-center p-6">
      <div className="flex w-full max-w-[420px] flex-col gap-5">
        <Brand />
        <div>
          <h1 className="text-[22px] font-semibold">{title}</h1>
          {lead && <p className="mt-1.5 text-mute">{lead}</p>}
        </div>
        {children}
        <LanguageSelect />
      </div>
    </main>
  );
}
