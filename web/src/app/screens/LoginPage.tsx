import { useState, type FormEvent } from "react";
import { Link, Navigate, useLocation } from "react-router";
import { AddressNotice } from "../components/AddressNotice";
import { ErrorAlert, StatusNotice } from "../components/Alert";
import { buttonClass } from "../components/buttons";
import { Field } from "../components/Field";
import { useDocumentTitle } from "../components/useDocumentTitle";
import { useI18n } from "../i18n";
import { useSession } from "../session";
import { AuthLayout } from "./AuthLayout";

type LoginState = { notice?: "registeredPending" | "registeredActive" } | null;

export function LoginPage() {
  const { t, error } = useI18n();
  const { user, setup, login, startDiscord } = useSession();
  const location = useLocation();
  const [busy, setBusy] = useState(false);
  const [failure, setFailure] = useState<string>();
  useDocumentTitle(t("auth.login.title"));

  const state = location.state as LoginState;
  const addressNotice = new URLSearchParams(location.search).get("notice");

  // Once signed in, go back to the screen that asked for the session, else the library.
  const from = (location.state as { from?: string } | null)?.from;
  if (user) return <Navigate to={from ?? "/library"} replace />;

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const data = new FormData(event.currentTarget);
    setBusy(true);
    setFailure(undefined);
    const outcome = await login(
      String(data.get("username") ?? ""),
      String(data.get("password") ?? ""),
    );
    setBusy(false);
    if (!outcome.ok) {
      // A malformed login can never match an account: say so like any wrong login.
      setFailure(
        outcome.code === "invalid_request" ||
          outcome.code === "invalid_credentials_format"
          ? "invalid_credentials_or_pending"
          : outcome.code,
      );
    }
  }

  async function discord() {
    setBusy(true);
    setFailure(undefined);
    const outcome = await startDiscord();
    if (!outcome.ok) {
      setBusy(false);
      setFailure(outcome.code);
    }
  }

  return (
    <AuthLayout title={t("auth.login.title")} lead={t("auth.login.lead")}>
      {state?.notice && (
        <StatusNotice>{t(`notices.${state.notice}`)}</StatusNotice>
      )}
      {addressNotice && <AddressNotice code={addressNotice} />}
      <form onSubmit={submit} className="flex flex-col gap-4">
        <Field
          label={t("auth.username")}
          name="username"
          autoComplete="username"
          required
        />
        <Field
          label={t("auth.password")}
          name="password"
          type="password"
          autoComplete="current-password"
          required
        />
        {failure && <ErrorAlert>{error(failure)}</ErrorAlert>}
        <button
          type="submit"
          disabled={busy}
          className={`${buttonClass("primary")} w-full`}
        >
          {busy ? t("auth.login.submitting") : t("auth.login.submit")}
        </button>
      </form>
      {setup?.initialized && (
        <Link to="/register" className={`${buttonClass("secondary")} w-full`}>
          {t("auth.login.toRegister")}
        </Link>
      )}
      {setup?.discord_enabled && (
        <div className="flex flex-col gap-2">
          <button
            type="button"
            disabled={busy}
            onClick={discord}
            className={`${buttonClass("secondary")} w-full`}
          >
            {t("auth.discord.button")}
          </button>
          <p className="text-xs text-mute">{t("auth.discord.hint")}</p>
        </div>
      )}
    </AuthLayout>
  );
}
