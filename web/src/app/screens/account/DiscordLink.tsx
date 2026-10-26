import { useState } from "react";
import { ErrorAlert } from "../../components/Alert";
import { buttonClass } from "../../components/buttons";
import { useI18n } from "../../i18n";
import { useSession } from "../../session";

/** Links the signed-in account to the person's Discord identity (leaves for Discord). */
export function DiscordLink() {
  const { t, error } = useI18n();
  const { setup, linkDiscord } = useSession();
  const [busy, setBusy] = useState(false);
  const [failure, setFailure] = useState<string>();

  if (!setup?.discord_enabled) {
    return <p className="text-mute">{t("account.profile.discordDisabled")}</p>;
  }

  async function link() {
    setBusy(true);
    setFailure(undefined);
    const outcome = await linkDiscord();
    // On success the browser leaves for Discord; only a failure comes back here.
    if (!outcome.ok) {
      setBusy(false);
      setFailure(outcome.code);
    }
  }

  return (
    <>
      <p className="text-mute">{t("account.profile.discordBody")}</p>
      {failure && <ErrorAlert>{error(failure)}</ErrorAlert>}
      <div>
        <button
          type="button"
          disabled={busy}
          onClick={() => void link()}
          className={buttonClass("secondary")}
        >
          {busy
            ? t("account.profile.discordLinking")
            : t("account.profile.discordLink")}
        </button>
      </div>
    </>
  );
}
