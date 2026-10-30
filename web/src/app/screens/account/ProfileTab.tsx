import { Shield } from "lucide-react";
import { useI18n } from "../../i18n";
import { roleLabelKey } from "../../roles";
import { useSession } from "../../session";
import { DiscordLink } from "./DiscordLink";
import { LanguagePreference } from "./LanguagePreference";
import { CloseAccount } from "./CloseAccount";
import { Section } from "./Section";

/** The "Profile" tab: who you are, the Discord link and the language. */
export function ProfileTab() {
  const { t } = useI18n();
  const { user } = useSession();
  if (!user) return null;
  return (
    <div className="grid items-start gap-5 md:grid-cols-[minmax(0,1fr)_minmax(0,2fr)]">
      <section
        aria-label={t("account.profile.identityTitle")}
        className="min-w-0 rounded-2xl border border-line bg-surface p-6"
      >
        <div className="flex flex-wrap items-center gap-4">
          <span
            aria-hidden="true"
            className="grid size-16 shrink-0 place-items-center rounded-full bg-accent-soft text-xl font-semibold text-accent-ink"
          >
            {Array.from(user.username).slice(0, 2).join("").toLocaleUpperCase()}
          </span>
          <div className="min-w-0">
            <h2 className="break-words text-2xl font-semibold">
              {user.username}
            </h2>
            <p className="mt-2 flex items-center gap-2 text-sm text-mute">
              <Shield size={16} aria-hidden="true" />
              {t(roleLabelKey(user))}
            </p>
          </div>
        </div>
      </section>
      <div className="min-w-0 space-y-5">
        <Section title={t("account.profile.discordTitle")}>
          <DiscordLink />
        </Section>
        <Section title={t("account.profile.preferencesTitle")}>
          <LanguagePreference />
        </Section>
        <CloseAccount />
      </div>
    </div>
  );
}
