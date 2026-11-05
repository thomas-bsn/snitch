import { useCallback, useEffect, useRef, useState } from "react";
import { api, errorCode, mutation } from "../../api/client";
import type { components } from "../../../api.gen";
import { ErrorAlert, StatusNotice } from "../../components/Alert";
import { useI18n } from "../../i18n";
import { useSession } from "../../session";
import { buttonClass } from "../../components/buttons";

export function Avatar({ initials }: { initials: string }) {
  const [version, setVersion] = useState(0);
  const [failed, setFailed] = useState(false);
  useEffect(() => {
    const update = () => {
      setFailed(false);
      setVersion(Date.now());
    };
    window.addEventListener("snitch-avatar", update);
    return () => window.removeEventListener("snitch-avatar", update);
  }, []);
  return failed ? (
    <span aria-hidden="true">{initials}</span>
  ) : (
    <img
      className="size-full rounded-full object-cover"
      alt=""
      src={`/api/account/avatar?v=${version}`}
      onError={() => setFailed(true)}
    />
  );
}

type Settings = components["schemas"]["AvatarSettings"];
type Source = Settings["source"];

/** The photo of the account: where it comes from, and how to change that. */
export function AvatarEditor() {
  const { t, error } = useI18n();
  const { session, user } = useSession();
  const [settings, setSettings] = useState<Settings>();
  const [busy, setBusy] = useState(false);
  const [failure, setFailure] = useState<string>();
  const [message, setMessage] = useState("");
  const [choosing, setChoosing] = useState(false);
  const picker = useRef<HTMLInputElement>(null);
  const header = mutation(session?.csrf_token);

  const load = useCallback(async () => {
    try {
      const r = await api.GET("/api/account/avatar-settings");
      if (r.data) setSettings(r.data);
    } catch {
      /* The photo itself still shows; the controls simply stay hidden. */
    }
  }, []);
  useEffect(() => {
    void load();
  }, [load]);

  /** Runs one change, then tells the rest of the page (menu, profile) that the photo may have changed. */
  async function run(work: () => Promise<string | undefined>, done: string) {
    setBusy(true);
    setFailure(undefined);
    setMessage("");
    try {
      const code = await work();
      if (code) setFailure(code);
      else setMessage(done);
      window.dispatchEvent(new Event("snitch-avatar"));
      await load();
    } catch {
      setFailure("unavailable");
    } finally {
      setBusy(false);
    }
  }
  const choose = (source: Source) =>
    run(async () => {
      const r = await api.PUT("/api/account/avatar-settings", {
        params: { header },
        body: { source },
      });
      return r.error ? (errorCode(r.error) ?? "unknown") : undefined;
    }, t("account.profile.avatar.updated"));
  const refresh = () =>
    run(async () => {
      const r = await api.POST("/api/account/avatar/discord-refresh", {
        params: { header },
      });
      return r.error ? (errorCode(r.error) ?? "unknown") : undefined;
    }, t("account.profile.avatar.refreshed"));
  const upload = (file: File) => {
    if (file.size > 5 * 1024 * 1024) {
      setFailure("avatar_too_large");
      return;
    }
    return run(async () => {
      const response = await fetch("/api/account/avatar", {
        method: "PUT",
        headers: {
          "X-CSRF-Token": session?.csrf_token ?? "",
          "Content-Type": file.type,
        },
        body: file,
      });
      if (response.ok) {
        setChoosing(false);
        return undefined;
      }
      const body = (await response.json().catch(() => ({}))) as unknown;
      return errorCode(body) ?? "invalid_avatar";
    }, t("account.profile.avatar.updated"));
  };
  const remove = () =>
    run(async () => {
      const response = await fetch("/api/account/avatar", {
        method: "DELETE",
        headers: { "X-CSRF-Token": session?.csrf_token ?? "" },
      });
      setChoosing(false);
      return response.ok ? undefined : "unavailable";
    }, t("account.profile.avatar.updated"));

  // Without a Discord identity there is nothing to choose between: only the personal photo is offered.
  const active: Source =
    choosing || settings?.discord_linked === false
      ? "custom"
      : (settings?.source ?? "none");
  const sourceLabel = {
    discord: t("account.profile.avatar.sourceDiscord"),
    custom: t("account.profile.avatar.sourceCustom"),
    none: t("account.profile.avatar.sourceNone"),
  }[settings?.source ?? "none"];
  return (
    <div className="flex flex-col gap-3">
      <h3 className="font-medium">{t("account.profile.avatar.title")}</h3>
      <div className="flex items-center gap-3">
        <div className="flex size-20 shrink-0 items-center justify-center rounded-full bg-avatar-bg text-accent-ink">
          <Avatar
            initials={
              (user?.display_name ?? user?.username)
                ?.slice(0, 2)
                .toUpperCase() ?? ""
            }
          />
        </div>
        {settings && (
          <p className="text-sm text-mute">
            {t("account.profile.avatar.current", { source: sourceLabel })}
            <br />
            {settings.auto_sync
              ? t("account.profile.avatar.autoOn")
              : t("account.profile.avatar.autoOff")}
          </p>
        )}
      </div>
      {settings?.discord_linked && (
        <div
          role="group"
          aria-label={t("account.profile.avatar.sourceLegend")}
          className="flex flex-wrap gap-2"
        >
          <button
            type="button"
            aria-pressed={active === "discord"}
            disabled={busy}
            className={buttonClass(
              active === "discord" ? "primary" : "secondary",
            )}
            onClick={() => {
              setChoosing(false);
              void choose("discord");
            }}
          >
            {t("account.profile.avatar.optionDiscord")}
          </button>
          <button
            type="button"
            aria-pressed={active === "custom"}
            disabled={busy}
            className={buttonClass(
              active === "custom" ? "primary" : "secondary",
            )}
            onClick={() => {
              if (settings.has_custom) void choose("custom");
              else setChoosing(true);
            }}
          >
            {t("account.profile.avatar.optionCustom")}
          </button>
        </div>
      )}
      {active === "discord" && settings?.discord_linked && (
        <div className="flex flex-col gap-2">
          {settings.bot_configured ? (
            <div>
              <button
                type="button"
                className={buttonClass("secondary")}
                disabled={busy}
                onClick={() => void refresh()}
              >
                {t("account.profile.avatar.refresh")}
              </button>
            </div>
          ) : (
            <p className="text-sm text-mute">
              {t("account.profile.avatar.noBot")}
            </p>
          )}
          {!settings.has_discord && (
            <p className="text-sm text-mute">
              {t("account.profile.avatar.noDiscordPicture")}
            </p>
          )}
        </div>
      )}
      {active === "custom" && (
        <div className="flex flex-col gap-2">
          <div className="flex flex-wrap gap-2">
            <button
              type="button"
              className={buttonClass("secondary")}
              disabled={busy}
              onClick={() => picker.current?.click()}
            >
              {t("account.profile.avatar.choose")}
            </button>
            {settings?.has_custom && (
              <button
                type="button"
                className={buttonClass("secondary")}
                disabled={busy}
                onClick={() => void remove()}
              >
                {t("account.profile.avatar.remove")}
              </button>
            )}
          </div>
          <input
            ref={picker}
            type="file"
            accept="image/png,image/jpeg"
            aria-label={t("account.profile.avatar.choose")}
            disabled={busy}
            className="sr-only"
            onChange={(e) => {
              const file = e.target.files?.[0];
              e.target.value = "";
              if (file) void upload(file);
            }}
          />
          <p className="text-sm text-mute">
            {t("account.profile.avatar.hint")}{" "}
            {settings?.discord_linked
              ? t("account.profile.avatar.customWins")
              : ""}
          </p>
        </div>
      )}
      {busy && <p role="status">{t("account.profile.avatar.working")}</p>}
      {failure && <ErrorAlert>{error(failure)}</ErrorAlert>}
      {message && !failure && <StatusNotice>{message}</StatusNotice>}
    </div>
  );
}
