import { X } from "lucide-react";
import { useI18n } from "../i18n";
import { ErrorAlert, StatusNotice } from "./Alert";

/**
 * What the server puts in the address when it sends the browser back (after Discord, for
 * example): a code, shown as a message in the chosen language. The sign-in screen shows
 * it as is; inside the application it can be dismissed.
 */
export function AddressNotice({
  code,
  onDismiss,
}: {
  code: string;
  onDismiss?: () => void;
}) {
  const { t, error } = useI18n();
  if (code === "pending")
    return <StatusNotice>{t("notices.pending")}</StatusNotice>;
  if (!onDismiss) {
    return code === "discord_denied" ? (
      <StatusNotice>{error(code)}</StatusNotice>
    ) : (
      <ErrorAlert>{error(code)}</ErrorAlert>
    );
  }
  return (
    <div
      role="status"
      className="mx-4 mt-4 flex items-start gap-3 rounded-lg bg-accent-soft px-3 py-2.5 text-accent-ink md:mx-7"
    >
      <p className="flex-1">{error(code)}</p>
      <button
        type="button"
        onClick={onDismiss}
        aria-label={t("notices.dismiss")}
        className="flex size-6 shrink-0 items-center justify-center rounded-md hover:bg-surface"
      >
        <X className="size-4" />
      </button>
    </div>
  );
}
