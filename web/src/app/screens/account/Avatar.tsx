import { useEffect, useRef, useState } from "react";
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

export function AvatarEditor() {
  const { t } = useI18n();
  const { session, user } = useSession();
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState("");
  const picker = useRef<HTMLInputElement>(null);
  async function save(file?: File) {
    if (file && file.size > 5 * 1024 * 1024) {
      setMessage(t("account.profile.avatar.tooLarge"));
      return;
    }
    setBusy(true);
    setMessage("");
    try {
      const response = await fetch("/api/account/avatar", {
        method: file ? "PUT" : "DELETE",
        headers: {
          "X-CSRF-Token": session?.csrf_token ?? "",
          ...(file ? { "Content-Type": file.type } : {}),
        },
        body: file,
      });
      if (!response.ok) throw new Error();
      window.dispatchEvent(new Event("snitch-avatar"));
      setMessage(t("account.profile.avatar.updated"));
    } catch {
      setMessage(t("account.profile.avatar.failed"));
    } finally {
      setBusy(false);
    }
  }
  return (
    <div className="flex flex-col gap-3">
      <h3 className="font-medium">{t("account.profile.avatar.title")}</h3>
      <div className="flex size-16 items-center justify-center rounded-full bg-avatar-bg text-accent-ink">
        <Avatar initials={user?.username.slice(0, 2).toUpperCase() ?? ""} />
      </div>
      <div>
        <button
          type="button"
          className={buttonClass("secondary")}
          disabled={busy}
          onClick={() => picker.current?.click()}
        >
          {t("account.profile.avatar.choose")}
        </button>
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
            if (file) void save(file);
          }}
        />
      </div>
      <p className="text-sm text-mute">{t("account.profile.avatar.hint")}</p>
      <div>
        <button
          className={buttonClass("secondary")}
          disabled={busy}
          onClick={() => void save()}
        >
          {t("account.profile.avatar.remove")}
        </button>
      </div>
      <p aria-live="polite" className="text-sm">
        {message}
      </p>
    </div>
  );
}
