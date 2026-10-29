import { AvatarEditor } from "./Avatar";
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
    <div className="flex flex-col gap-4">
      <Section title={t("account.profile.identityTitle")}>
        <AvatarEditor />
        <dl className="grid grid-cols-[max-content_1fr] gap-x-6 gap-y-1.5">
          <dt className="text-mute">{t("account.profile.username")}</dt>
          <dd className="font-medium">{user.username}</dd>
          <dt className="text-mute">{t("account.profile.role")}</dt>
          <dd className="font-medium">{t(roleLabelKey(user))}</dd>
        </dl>
      </Section>
      <Section title={t("account.profile.discordTitle")}>
        <DiscordLink />
      </Section>
      <CloseAccount />
      <Section title={t("account.profile.preferencesTitle")}>
        <LanguagePreference />
      </Section>
    </div>
  );
}
