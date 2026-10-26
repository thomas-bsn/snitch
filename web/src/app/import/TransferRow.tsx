import { useRef } from "react";
import { buttonClass } from "../components/buttons";
import { useI18n } from "../i18n";
import { VIDEO_ACCEPT } from "./DropZone";
import type { Transfer } from "./useTransfers";

const percentOf = (t: Transfer) =>
  t.size === 0 ? 0 : Math.min(100, Math.floor((t.offset / t.size) * 100));

/** One file: its state, a progress bar and what can be done (resume, cancel, dismiss). */
export function TransferRow({
  transfer,
  onResume,
  onCancel,
  onDismiss,
}: {
  transfer: Transfer;
  onResume: (key: string, file?: File) => void;
  onCancel: (key: string) => void;
  onDismiss: (key: string) => void;
}) {
  const { t, error } = useI18n();
  const picker = useRef<HTMLInputElement>(null);
  const percent = percentOf(transfer);
  const { name, status } = transfer;
  const busy =
    status === "queued" || status === "sending" || status === "verifying";

  return (
    <li
      className="flex flex-col gap-2 rounded-xl border border-line bg-surface p-4"
      data-status={status}
    >
      <div className="flex flex-wrap items-baseline justify-between gap-x-3 gap-y-1">
        <span className="break-all font-medium">{name}</span>
        <span className="text-xs text-mute">
          {t(`import.status.${status}`)}
          {status !== "queued" && status !== "done" && (
            <> · {t("import.percent", { percent })}</>
          )}
        </span>
      </div>

      <div
        role="progressbar"
        aria-label={t("import.progress", { name, percent })}
        aria-valuemin={0}
        aria-valuemax={100}
        aria-valuenow={status === "done" ? 100 : percent}
        className="h-2 overflow-hidden rounded-full bg-side"
      >
        <div
          className={`h-full rounded-full ${status === "failed" ? "bg-bad-fg" : "bg-accent-solid"}`}
          style={{ width: `${status === "done" ? 100 : percent}%` }}
        />
      </div>

      {transfer.code && (status === "interrupted" || status === "failed") && (
        <p role="alert" className="text-bad-fg">
          {error(transfer.code)}
        </p>
      )}
      {status === "interrupted" && !transfer.file && (
        <p className="text-xs text-mute">{t("import.resumeHint", { name })}</p>
      )}
      {transfer.mismatch && (
        <p role="alert" className="text-bad-fg">
          {t("import.mismatch")}
        </p>
      )}

      <div className="flex flex-wrap gap-2">
        {status === "interrupted" && (
          <>
            <button
              type="button"
              className={buttonClass("primary")}
              aria-label={
                transfer.file
                  ? t("import.resumeLabel", { name })
                  : t("import.resumeFileLabel", { name })
              }
              onClick={() =>
                transfer.file ? onResume(transfer.key) : picker.current?.click()
              }
            >
              {transfer.file ? t("import.resume") : t("import.resumeFile")}
            </button>
            <input
              ref={picker}
              type="file"
              accept={VIDEO_ACCEPT}
              tabIndex={-1}
              aria-label={t("import.resumeHint", { name })}
              className="sr-only"
              onChange={(event) => {
                const file = event.target.files?.[0];
                if (file) onResume(transfer.key, file);
                event.target.value = "";
              }}
            />
          </>
        )}
        {(busy || status === "interrupted") && (
          <button
            type="button"
            className={buttonClass("secondary")}
            aria-label={t("import.cancelLabel", { name })}
            onClick={() => onCancel(transfer.key)}
          >
            {t("import.cancel")}
          </button>
        )}
        {(status === "done" || status === "failed") && (
          <button
            type="button"
            className={buttonClass("secondary")}
            aria-label={t("import.dismissLabel", { name })}
            onClick={() => onDismiss(transfer.key)}
          >
            {t("import.dismiss")}
          </button>
        )}
      </div>
    </li>
  );
}
