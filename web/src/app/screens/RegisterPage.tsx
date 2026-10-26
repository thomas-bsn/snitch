import { useState, type FormEvent } from "react";
import { Link, Navigate, useNavigate } from "react-router";
import { ErrorAlert } from "../components/Alert";
import { buttonClass } from "../components/buttons";
import { Field } from "../components/Field";
import { useDocumentTitle } from "../components/useDocumentTitle";
import { useI18n } from "../i18n";
import { useSession } from "../session";
import { AuthLayout } from "./AuthLayout";

export function RegisterPage() {
  const { t, error } = useI18n();
  const { user, register } = useSession();
  const navigate = useNavigate();
  const [busy, setBusy] = useState(false);
  const [failure, setFailure] = useState<string>();
  useDocumentTitle(t("auth.register.title"));

  if (user) return <Navigate to="/library" replace />;

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const data = new FormData(event.currentTarget);
    setBusy(true);
    setFailure(undefined);
    const outcome = await register(
      String(data.get("username") ?? ""),
      String(data.get("password") ?? ""),
    );
    setBusy(false);
    if (!outcome.ok) {
      setFailure(outcome.code);
      return;
    }
    navigate("/login", {
      replace: true,
      state: {
        notice:
          outcome.value === "pending"
            ? "registeredPending"
            : "registeredActive",
      },
    });
  }

  return (
    <AuthLayout title={t("auth.register.title")} lead={t("auth.register.lead")}>
      <form onSubmit={submit} className="flex flex-col gap-4">
        <Field
          label={t("auth.username")}
          hint={t("auth.usernameHint")}
          name="username"
          autoComplete="username"
          required
        />
        <Field
          label={t("auth.password")}
          hint={t("auth.passwordHint")}
          name="password"
          type="password"
          autoComplete="new-password"
          required
        />
        {failure && <ErrorAlert>{error(failure)}</ErrorAlert>}
        <button
          type="submit"
          disabled={busy}
          className={`${buttonClass("primary")} w-full`}
        >
          {busy ? t("auth.register.submitting") : t("auth.register.submit")}
        </button>
      </form>
      <Link to="/login" className={`${buttonClass("secondary")} w-full`}>
        {t("auth.register.toLogin")}
      </Link>
    </AuthLayout>
  );
}
