import { Avatar } from "../screens/account/Avatar";
import { Menu } from "@base-ui/react/menu";
import {
  LogOut,
  Monitor,
  Moon,
  Settings,
  ShieldCheck,
  Sun,
} from "lucide-react";
import { Link } from "react-router";
import { useI18n } from "../i18n";
import type { MessageKey } from "../i18n/messages";
import { LANGUAGES, THEMES, isLanguage, type Theme } from "../preferences";
import { isAdministrator, roleLabelKey } from "../roles";
import { useSession } from "../session";
import { useTheme } from "../theme";

const themeChoices: Record<Theme, { label: MessageKey; icon: typeof Sun }> = {
  light: { label: "menu.themeLight", icon: Sun },
  dark: { label: "menu.themeDark", icon: Moon },
  system: { label: "menu.themeSystem", icon: Monitor },
};

const itemClass =
  "flex min-h-11 w-full cursor-pointer items-center gap-2.5 rounded-md px-2.5 py-2 text-sm font-medium outline-none data-[highlighted]:bg-side md:min-h-9";

const segmentClass =
  "flex h-11 flex-1 cursor-pointer items-center justify-center rounded-md border border-line-strong text-sm font-medium text-nav outline-none data-[highlighted]:bg-side data-[checked]:border-accent data-[checked]:bg-accent-soft data-[checked]:text-accent-ink md:h-8";

/** The avatar at the far right of the header and the menu it opens. */
export function AvatarMenu() {
  const { t, language } = useI18n();
  const { user, logout, chooseLanguage } = useSession();
  const { theme, setTheme } = useTheme();
  if (!user) return null;
  const initials = user.username.slice(0, 2).toUpperCase();

  return (
    <Menu.Root>
      <Menu.Trigger
        // The visible initials are part of the accessible name, so assistive
        // technology and the visible label always agree.
        aria-label={`${t("header.accountMenu")} ${initials}`}
        className="flex size-11 shrink-0 items-center justify-center rounded-full bg-avatar-bg text-xs font-semibold text-accent-ink md:size-8"
      >
        <Avatar initials={initials} />
      </Menu.Trigger>
      <Menu.Portal>
        <Menu.Positioner align="end" sideOffset={8} className="z-50">
          <Menu.Popup className="w-64 rounded-xl border border-line bg-surface p-1.5 text-ink shadow-pop outline-none">
            <div className="px-2.5 pb-2.5 pt-2">
              <span className="block text-sm font-semibold">
                {user.username}
              </span>
              <span className="block text-xs text-mute">
                {t(roleLabelKey(user))}
              </span>
            </div>
            <Menu.Separator className="my-1.5 h-px bg-line" />
            <Menu.LinkItem
              render={<Link to="/account" />}
              closeOnClick
              className={itemClass}
            >
              <Settings className="size-4" />
              {t("menu.account")}
            </Menu.LinkItem>
            {isAdministrator(user) && (
              <Menu.LinkItem
                render={<Link to="/admin" />}
                closeOnClick
                className={itemClass}
              >
                <ShieldCheck className="size-4" />
                {t("menu.administration")}
              </Menu.LinkItem>
            )}
            <Menu.Separator className="my-1.5 h-px bg-line" />
            {/* The theme and language rows stay open on a pick, so the change is seen. */}
            <div className="px-2.5 pb-1 pt-1.5 text-xs font-medium text-mute">
              {t("menu.theme")}
            </div>
            <Menu.RadioGroup
              value={theme}
              onValueChange={(value) => {
                const next = THEMES.find((candidate) => candidate === value);
                if (next) setTheme(next);
              }}
              aria-label={t("menu.theme")}
              className="flex gap-1 px-1 pb-1"
            >
              {THEMES.map((choice) => {
                const { label, icon: Icon } = themeChoices[choice];
                return (
                  <Menu.RadioItem
                    key={choice}
                    value={choice}
                    aria-label={t(label)}
                    title={t(label)}
                    className={segmentClass}
                  >
                    <Icon className="size-4" />
                  </Menu.RadioItem>
                );
              })}
            </Menu.RadioGroup>
            <div className="px-2.5 pb-1 pt-1.5 text-xs font-medium text-mute">
              {t("menu.language")}
            </div>
            <Menu.RadioGroup
              value={language}
              onValueChange={(value) => {
                if (isLanguage(value)) void chooseLanguage(value);
              }}
              aria-label={t("menu.language")}
              className="flex gap-1 px-1 pb-1"
            >
              {LANGUAGES.map((code) => (
                <Menu.RadioItem
                  key={code}
                  value={code}
                  lang={code}
                  className={segmentClass}
                >
                  {t(`language.${code}`)}
                </Menu.RadioItem>
              ))}
            </Menu.RadioGroup>
            <Menu.Separator className="my-1.5 h-px bg-line" />
            <Menu.Item className={itemClass} onClick={() => void logout()}>
              <LogOut className="size-4" />
              {t("menu.signOut")}
            </Menu.Item>
          </Menu.Popup>
        </Menu.Positioner>
      </Menu.Portal>
    </Menu.Root>
  );
}
