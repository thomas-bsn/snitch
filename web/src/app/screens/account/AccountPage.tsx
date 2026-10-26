import { Tabs } from "@base-ui/react/tabs";
import { useLocation, useNavigate } from "react-router";
import { useDocumentTitle } from "../../components/useDocumentTitle";
import { useI18n } from "../../i18n";
import { Page } from "../../pages/pages";
import { ProfileTab } from "./ProfileTab";
import { VoiceTab } from "./VoiceTab";

const TABS = [
  { value: "profile", path: "/account", label: "account.tabs.profile" },
  { value: "voice", path: "/account/voice", label: "account.tabs.voice" },
] as const;

type TabValue = (typeof TABS)[number]["value"];

const tabClass =
  "-mb-px flex h-11 cursor-pointer items-center border-b-2 border-transparent px-3 text-sm font-medium text-nav outline-none hover:text-ink data-[selected]:border-accent data-[selected]:text-accent-ink focus-visible:outline-2 focus-visible:outline-accent md:h-10";

/**
 * Account settings: two tabs, each with its own address (the profile, and "My voice"
 * under /account/voice). There is no theme setting here: it is chosen in the profile menu.
 */
export function AccountPage() {
  const { t } = useI18n();
  const navigate = useNavigate();
  const { pathname } = useLocation();
  useDocumentTitle(t("pages.account.title"));

  const active: TabValue = /^\/account\/voice\/?$/.test(pathname)
    ? "voice"
    : "profile";

  return (
    <Page title={t("pages.account.title")}>
      <Tabs.Root
        value={active}
        onValueChange={(value) => {
          const next = TABS.find((tab) => tab.value === value);
          if (next) navigate(next.path);
        }}
        className="flex flex-col gap-5"
      >
        <Tabs.List
          aria-label={t("account.tabsLabel")}
          className="flex gap-1 border-b border-line"
        >
          {TABS.map(({ value, label }) => (
            <Tabs.Tab key={value} value={value} className={tabClass}>
              {t(label)}
            </Tabs.Tab>
          ))}
        </Tabs.List>
        <Tabs.Panel value="profile" className="outline-none">
          <ProfileTab />
        </Tabs.Panel>
        <Tabs.Panel value="voice" className="outline-none">
          <VoiceTab />
        </Tabs.Panel>
      </Tabs.Root>
    </Page>
  );
}
