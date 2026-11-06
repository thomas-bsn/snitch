import { useEffect, useState, type FormEvent } from "react";
import { api, errorCode, mutation } from "../../api/client";
import { ErrorAlert, StatusNotice } from "../../components/Alert";
import { Field } from "../../components/Field";
import { buttonClass } from "../../components/buttons";
import { useI18n } from "../../i18n";
import { useSession } from "../../session";

export function IdentityEditor() {
  const { user, session, reload } = useSession();
  const { t, error } = useI18n();
  const [name, setName] = useState(user?.display_name ?? user?.username ?? "");
  const [username, setUsername] = useState(user?.username ?? "");
  const [busy, setBusy] = useState(false);
  const [failure, setFailure] = useState<string>();
  const [saved, setSaved] = useState(false);
  useEffect(() => {
    setName(user?.display_name ?? user?.username ?? "");
    setUsername(user?.username ?? "");
  }, [user?.display_name, user?.username]);
  async function save(e: FormEvent) {
    e.preventDefault();
    setBusy(true);
    setFailure(undefined);
    setSaved(false);
    try {
      const r = await api.PUT("/api/account/profile", {
        params: { header: mutation(session?.csrf_token) },
        body: { display_name: name, username },
      });
      if (r.error) setFailure(errorCode(r.error) ?? "unknown");
      else {
        await reload();
        setSaved(true);
      }
    } catch {
      setFailure("unavailable");
    } finally {
      setBusy(false);
    }
  }
  return (
    <form onSubmit={save} className="flex flex-col gap-4">
      <Field
        label={t("account.profile.displayName")}
        hint={t("account.profile.displayNameHint")}
        value={name}
        onChange={(e) => {
          setName(e.target.value);
          setSaved(false);
        }}
        required
        maxLength={100}
        autoComplete="nickname"
        disabled={busy}
      />
      <Field
        label={t("account.profile.username")}
        hint={t("account.profile.loginHint")}
        value={username}
        onChange={(e) => {
          setUsername(e.target.value);
          setSaved(false);
        }}
        required
        minLength={3}
        maxLength={64}
        autoComplete="username"
        disabled={busy}
      />
      {failure && <ErrorAlert>{error(failure)}</ErrorAlert>}
      {saved && (
        <StatusNotice>{t("account.profile.identitySaved")}</StatusNotice>
      )}
      <div>
        <button
          className={buttonClass("primary")}
          disabled={
            busy ||
            (name === (user?.display_name ?? user?.username) &&
              username === user?.username)
          }
        >
          {busy
            ? t("account.profile.saving")
            : t("account.profile.saveIdentity")}
        </button>
      </div>
    </form>
  );
}

export function PasswordEditor() {
  const { session, reload } = useSession();
  const { t, error } = useI18n();
  const [current, setCurrent] = useState("");
  const [password, setPassword] = useState("");
  const [confirmation, setConfirmation] = useState("");
  const [busy, setBusy] = useState(false);
  const [failure, setFailure] = useState<string>();
  const [saved, setSaved] = useState(false);
  const [touched, setTouched] = useState(false);
  const hasPassword = session?.has_password === true;
  const mismatch = confirmation !== password;
  async function save(e: FormEvent) {
    e.preventDefault();
    setTouched(true);
    setSaved(false);
    if (mismatch) return;
    setBusy(true);
    setFailure(undefined);
    try {
      const r = await api.PUT("/api/account/password", {
        params: { header: mutation(session?.csrf_token) },
        body: { current_password: current, new_password: password },
      });
      if (r.error) setFailure(errorCode(r.error) ?? "unknown");
      else {
        setCurrent("");
        setPassword("");
        setConfirmation("");
        setTouched(false);
        await reload();
        setSaved(true);
      }
    } catch {
      setFailure("unavailable");
    } finally {
      setBusy(false);
    }
  }
  return (
    <form onSubmit={save} className="flex flex-col gap-4">
      <p className="text-sm leading-relaxed text-mute">
        {t(
          hasPassword
            ? "account.profile.passwordHint"
            : "account.profile.firstPasswordHint",
        )}
      </p>
      {hasPassword && (
        <Field
          label={t("account.profile.currentPassword")}
          type="password"
          autoComplete="current-password"
          value={current}
          onChange={(e) => setCurrent(e.target.value)}
          required
          maxLength={256}
          disabled={busy}
        />
      )}
      <Field
        label={t("account.profile.newPassword")}
        hint={t("account.profile.passwordLength")}
        type="password"
        autoComplete="new-password"
        value={password}
        onChange={(e) => {
          setPassword(e.target.value);
          setSaved(false);
        }}
        required
        minLength={12}
        maxLength={256}
        disabled={busy}
      />
      <Field
        label={t("account.profile.confirmPassword")}
        hint={
          touched && mismatch
            ? t("account.profile.passwordMismatch")
            : undefined
        }
        aria-invalid={(touched && mismatch) || undefined}
        type="password"
        autoComplete="new-password"
        value={confirmation}
        onBlur={() => setTouched(true)}
        onChange={(e) => setConfirmation(e.target.value)}
        required
        maxLength={256}
        disabled={busy}
      />
      {failure && <ErrorAlert>{error(failure)}</ErrorAlert>}
      {saved && (
        <StatusNotice>{t("account.profile.passwordSaved")}</StatusNotice>
      )}
      <div>
        <button className={buttonClass("secondary")} disabled={busy}>
          {busy
            ? t("account.profile.saving")
            : t(
                hasPassword
                  ? "account.profile.changePassword"
                  : "account.profile.setPassword",
              )}
        </button>
      </div>
    </form>
  );
}
