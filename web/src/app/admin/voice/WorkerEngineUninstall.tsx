import { useEffect, useState } from "react";
import { api, errorCode } from "../../api/client";
import { ErrorAlert } from "../../components/Alert";
import { Modal } from "../../components/Modal";
import { useI18n } from "../../i18n";
import type { components } from "../../../api.gen";
import type { EngineEntry, EngineWorker } from "./engineOrders";

type Preview = components["schemas"]["EngineUninstallPreview"];

type Props = {
  worker: EngineWorker;
  entry: EngineEntry;
  busy: boolean;
  onClose: () => void;
  /** The key is fixed for the life of the dialog: a double click records one removal. */
  onConfirm: (idempotencyKey: string) => Promise<void>;
};

/**
 * What Uninstall shows before anything is ordered: the models bound to the release, the jobs still using them and
 * the fingerprint transitions that hold them. Nothing is removed from here except the one release shown.
 */
export function WorkerEngineUninstall({
  worker,
  entry,
  busy,
  onClose,
  onConfirm,
}: Props) {
  const { t, error } = useI18n();
  const [key] = useState(() => crypto.randomUUID());
  const [preview, setPreview] = useState<Preview>();
  const [failure, setFailure] = useState<string>();
  useEffect(() => {
    let cancelled = false;
    api
      .GET("/api/admin/voice-engines/uninstall-preview", {
        params: {
          query: { worker_id: worker.worker_id, digest: entry.digest },
        },
      })
      .then(({ data, error: problem }) => {
        if (cancelled) return;
        if (data) setPreview(data);
        else setFailure(errorCode(problem) ?? "unknown");
      })
      .catch(() => !cancelled && setFailure("unavailable"));
    return () => {
      cancelled = true;
    };
  }, [worker.worker_id, entry.digest]);

  return (
    <Modal
      open
      onOpenChange={(open) => !open && !busy && onClose()}
      title={t("workerEngines.uninstall.title", {
        engine: `${entry.engine} ${entry.version}`,
        worker: worker.name,
      })}
      closeLabel={t("voiceModels.byWorker.close")}
    >
      <p className="text-mute">{t("workerEngines.uninstall.lead")}</p>
      {failure && <ErrorAlert>{error(failure)}</ErrorAlert>}
      {!preview && !failure && (
        <p role="status">{t("workerEngines.uninstall.checking")}</p>
      )}
      {preview && (
        <div aria-live="polite">
          {preview.blockers.models.length > 0 && (
            <>
              <h4>{t("workerEngines.uninstall.models")}</h4>
              <ul className="engine-blockers">
                {preview.blockers.models.map((m) => (
                  <li key={m.model_id}>
                    <strong>{m.label}</strong> ({m.kind})
                    {m.busy && (
                      <span className="engine-badge run">
                        {t("workerEngines.uninstall.busy")}
                      </span>
                    )}
                    {m.in_transition && (
                      <span className="engine-badge run">
                        {t("workerEngines.uninstall.transition")}
                      </span>
                    )}
                  </li>
                ))}
              </ul>
            </>
          )}
          {preview.blockers.open_order && (
            <p className="engine-warn">
              {t("workerEngines.uninstall.openOrder")}
            </p>
          )}
          <p className={preview.removable ? "engine-note" : "engine-warn"}>
            {t(
              preview.removable
                ? "workerEngines.uninstall.removable"
                : "workerEngines.uninstall.blocked",
            )}
          </p>
        </div>
      )}
      <div className="engine-bar">
        <button type="button" className="voice-btn" onClick={onClose}>
          {t("workerEngines.cancel")}
        </button>
        <button
          type="button"
          className="voice-btn primary"
          disabled={busy || !preview?.removable}
          onClick={() => void onConfirm(key)}
        >
          {t("workerEngines.uninstall.confirm")}
        </button>
      </div>
    </Modal>
  );
}
