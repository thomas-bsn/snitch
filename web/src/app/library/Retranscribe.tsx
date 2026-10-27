import { useState } from "react";
import type { components } from "../../api.gen";
import { api, errorCode, mutation } from "../api/client";
import { ErrorAlert, StatusNotice } from "../components/Alert";
import { buttonClass } from "../components/buttons";
import { ConfirmDialog } from "../components/ConfirmDialog";
import { useI18n } from "../i18n";
import { isAdministrator } from "../roles";
import { useSession } from "../session";
export function Retranscribe({ id, stream }: { id: number; stream?: number }) {
  const { t, error } = useI18n();
  const { user, session } = useSession();
  const [warning, setWarning] =
    useState<components["schemas"]["CorrectionsToReplace"]>();
  const [failure, setFailure] = useState<string>();
  const [done, setDone] = useState(false);
  const [busy, setBusy] = useState(false);
  if (!user || !isAdministrator(user)) return null;
  async function submit(confirm = false) {
    setBusy(true);
    setFailure(undefined);
    try {
      const r = await api.POST("/api/admin/clips/{id}/transcription", {
        params: {
          path: { id },
          header: mutation(session?.csrf_token),
          query: { confirm_replace_corrections: confirm || undefined },
        },
        body: {
          ...(stream !== undefined && stream >= 0
            ? { audio_stream_index: stream }
            : {}),
        },
      });
      if (r.error) {
        if (
          "corrections" in r.error &&
          r.error.error === "corrections_would_be_replaced" &&
          r.error.corrections
        )
          setWarning(r.error.corrections);
        else setFailure(errorCode(r.error) ?? "unknown");
        return;
      }
      setWarning(undefined);
      setDone(true);
    } catch {
      setFailure("unavailable");
    } finally {
      setBusy(false);
    }
  }
  return (
    <div className="flex flex-col gap-2">
      <div>
        <button
          type="button"
          disabled={busy}
          onClick={() => void submit()}
          className={buttonClass()}
        >
          {t("segmentEdit.retranscribe")}
        </button>
      </div>
      {failure && <ErrorAlert>{error(failure)}</ErrorAlert>}
      {done && <StatusNotice>{t("segmentEdit.scheduled")}</StatusNotice>}
      <ConfirmDialog
        open={!!warning}
        onOpenChange={(open) => {
          if (!open) setWarning(undefined);
        }}
        title={t("segmentEdit.retranscribeTitle")}
        cancelLabel={t("segmentEdit.cancel")}
        confirmLabel={t("segmentEdit.confirmRetranscribe")}
        danger
        busy={busy}
        onConfirm={() => void submit(true)}
      >
        {warning && (
          <p>
            {t("segmentEdit.retranscribeWarning", {
              texts: warning.text_segments,
              speakers: warning.speaker_segments,
              authors: warning.authors.join(", "),
            })}
          </p>
        )}
      </ConfirmDialog>
    </div>
  );
}
