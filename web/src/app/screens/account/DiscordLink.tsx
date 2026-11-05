import { CheckCircle2, Link2 } from "lucide-react";
import { useId, useState } from "react";
import { ErrorAlert } from "../../components/Alert";
import { ConfirmDialog } from "../../components/ConfirmDialog";
import { buttonClass } from "../../components/buttons";
import { useI18n } from "../../i18n";
import { useSession } from "../../session";

function DiscordMark() {
  return (
    <svg
      aria-hidden="true"
      viewBox="0 0 24 24"
      className="size-6 shrink-0 fill-current"
    >
      <path d="M19.73 5.36A18.2 18.2 0 0 0 15.25 4a12.6 12.6 0 0 0-.57 1.17 16.8 16.8 0 0 0-5.36 0A12.6 12.6 0 0 0 8.75 4a18.2 18.2 0 0 0-4.48 1.36C1.43 9.55.66 13.64 1.05 17.67a18.3 18.3 0 0 0 5.5 2.78c.45-.61.85-1.26 1.2-1.95-.66-.25-1.29-.57-1.88-.95l.46-.36a13 13 0 0 0 11.34 0l.46.36c-.59.38-1.22.7-1.88.95.35.69.75 1.34 1.2 1.95a18.3 18.3 0 0 0 5.5-2.78c.47-4.67-.8-8.72-3.22-12.31ZM8.55 14.7c-1.1 0-2-.99-2-2.2s.88-2.2 2-2.2 2.02.99 2 2.2c0 1.21-.88 2.2-2 2.2Zm6.9 0c-1.1 0-2-.99-2-2.2s.88-2.2 2-2.2 2.02.99 2 2.2c0 1.21-.88 2.2-2 2.2Z" />
    </svg>
  );
}

/** Shows the Discord link state and keeps unlinking behind an explicit confirmation. */
export function DiscordLink() {
  const { t, error } = useI18n();
  const { setup, session, linkDiscord, unlinkDiscord } = useSession();
  const [busy, setBusy] = useState(false);
  const [confirmOpen, setConfirmOpen] = useState(false);
  const [failure, setFailure] = useState<string>();
  const [notice, setNotice] = useState<string>();

  async function link() {
    setBusy(true);
    setFailure(undefined);
    setNotice(undefined);
    const outcome = await linkDiscord();
    // On success the browser leaves for Discord; only a failure comes back here.
    if (!outcome.ok) {
      setBusy(false);
      setFailure(outcome.code);
      setNotice(t("account.profile.discordLinkFailedState"));
    }
  }

  async function unlink() {
    setBusy(true);
    setFailure(undefined);
    setNotice(undefined);
    const outcome = await unlinkDiscord();
    setBusy(false);
    setConfirmOpen(false);
    if (outcome.ok) setNotice(t("account.profile.discordUnlinked"));
    else {
      setFailure(outcome.code);
      setNotice(t("account.profile.discordUnlinkFailedState"));
    }
  }

  const blockedId = useId();
  const linked = session?.discord_linked === true;
  return (
    <div className="flex flex-col gap-3">
      <div className="flex items-center gap-3">
        <span className="flex size-11 items-center justify-center rounded-full bg-[#5865f2] text-white">
          <DiscordMark />
        </span>
        <div className="min-w-0">
          <div className="flex flex-wrap items-center gap-3">
            <p className="font-semibold">Discord</p>
            <span
              className={`inline-flex items-center gap-1.5 rounded-full px-2.5 py-1 text-xs font-medium ${linked ? "bg-ok-bg text-ok-fg" : "bg-side text-mute"}`}
            >
              {linked ? (
                <CheckCircle2 size={14} aria-hidden="true" />
              ) : (
                <Link2 size={14} aria-hidden="true" />
              )}
              {linked
                ? t("account.profile.discordBadgeLinked")
                : t("account.profile.discordBadgeNotLinked")}
            </span>
          </div>
          {linked ? (
            <p className="break-words text-sm text-mute">
              {session?.discord_name
                ? t("account.profile.discordIdentity", {
                    name: session.discord_name,
                  })
                : t("account.profile.discordLinkedNoName")}
            </p>
          ) : (
            <p className="text-sm text-mute">
              {t("account.profile.discordBody")}
            </p>
          )}
          {linked && session?.discord_username && (
            <p className="break-all text-sm text-mute">
              @{session.discord_username}
            </p>
          )}
        </div>
      </div>
      {failure && <ErrorAlert>{error(failure)}</ErrorAlert>}
      {notice && (
        <p role="status" className="rounded-lg bg-ok-bg p-3 text-sm text-ok-fg">
          {notice}
        </p>
      )}
      {linked ? (
        <div className="flex flex-col items-start gap-2">
          <button
            type="button"
            disabled={busy || !session?.can_unlink_discord}
            aria-describedby={
              session?.can_unlink_discord ? undefined : blockedId
            }
            onClick={() => setConfirmOpen(true)}
            className={buttonClass("danger")}
          >
            {t("account.profile.discordUnlink")}
          </button>
          {!session?.can_unlink_discord && (
            <p id={blockedId} className="text-sm text-mute">
              {t("account.profile.discordUnlinkBlocked")}
            </p>
          )}
        </div>
      ) : setup?.discord_enabled ? (
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
      ) : (
        <p className="text-sm text-mute">
          {t("account.profile.discordDisabled")}
        </p>
      )}
      <ConfirmDialog
        open={confirmOpen}
        onOpenChange={setConfirmOpen}
        title={t("account.profile.discordUnlinkConfirmTitle")}
        cancelLabel={t("queue.cancel")}
        confirmLabel={
          busy
            ? t("account.profile.discordUnlinking")
            : t("account.profile.discordUnlinkConfirm")
        }
        onConfirm={() => void unlink()}
        danger
        busy={busy}
      >
        {t("account.profile.discordUnlinkConfirmBody")}
      </ConfirmDialog>
    </div>
  );
}
