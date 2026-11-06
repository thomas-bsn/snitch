import { AvatarEditor } from "./Avatar";
import { useI18n } from "../../i18n";
import { IdentityEditor, PasswordEditor } from "./ProfileEditors";
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
    <div className="mx-auto grid max-w-5xl [&_button]:min-h-11 [&_input]:min-h-11 [&_select]:min-h-11 items-start gap-6 lg:grid-cols-[minmax(0,1fr)_minmax(0,1fr)]">
      <div className="min-w-0 space-y-6">
        <Section title={t("account.profile.identityTitle")}>
          <div className="flex flex-wrap items-center justify-between gap-2">
            <p className="min-w-0 break-words text-xl font-semibold tracking-tight">
              {user.display_name ?? user.username}
            </p>
            <span className="rounded-full bg-side px-3 py-1 text-xs font-medium text-mute">
              {t(roleLabelKey(user))}
            </span>
          </div>
          <AvatarEditor />
          <div className="border-t border-line pt-5">
            <IdentityEditor />
          </div>
        </Section>
        <Section title={t("account.profile.preferencesTitle")}>
          <LanguagePreference />
        </Section>
      </div>
      <div className="min-w-0 space-y-6">
        <Section title={t("account.profile.securityTitle")}>
          <PasswordEditor />
        </Section>
        <Section title={t("account.profile.discordTitle")}>
          <DiscordLink />
        </Section>
        <CloseAccount />
      </div>
    </div>
  );
}
