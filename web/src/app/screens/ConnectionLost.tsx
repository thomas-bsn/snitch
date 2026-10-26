import { buttonClass } from "../components/buttons";
import { Brand } from "../components/Logo";
import { useDocumentTitle } from "../components/useDocumentTitle";
import { useI18n } from "../i18n";

/** Shown instead of a blank page when the server does not answer. */
export function ConnectionLost({ onRetry }: { onRetry: () => void }) {
  const { t } = useI18n();
  useDocumentTitle(t("connection.title"));
  return (
    <main className="flex min-h-dvh items-center justify-center p-6">
      <div role="alert" className="flex w-full max-w-md flex-col gap-5">
        <Brand />
        <div>
          <h1 className="text-[22px] font-semibold">{t("connection.title")}</h1>
          <p className="mt-1.5 text-mute">{t("connection.body")}</p>
        </div>
        <div>
          <button
            type="button"
            className={buttonClass("primary")}
            onClick={onRetry}
          >
            {t("connection.retry")}
          </button>
        </div>
      </div>
    </main>
  );
}

export function Splash() {
  const { t } = useI18n();
  return (
    <main className="flex min-h-dvh items-center justify-center">
      <p role="status" className="text-mute">
        {t("app.loading")}
      </p>
    </main>
  );
}
