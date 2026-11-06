import { useI18n } from "../i18n";
import { useCallback, useEffect, useRef, useState } from "react";
import { useSession } from "../session";
import {
  cancelUpload,
  completeUpload,
  createUpload,
  listUploads,
  putChunk,
  type Upload,
  type Visibility,
} from "./uploadApi";

export type TransferStatus =
  /** Waiting for its turn. */
  | "queued"
  | "sending"
  /** The server checks the file once every byte arrived. */
  | "verifying"
  | "done"
  /** Stopped before the end: it can be resumed. */
  | "interrupted"
  | "failed";

export type Transfer = {
  key: string;
  name: string;
  size: number;
  offset: number;
  status: TransferStatus;
  /** The file itself, when this page holds it (it does not after a reload). */
  file: File | null;
  visibility: Visibility;
  destination?: string;
  /** Server-side upload, once created. */
  upload: Upload | null;
  /** Stable error code of a failure or of the last interruption. */
  code?: string;
  /** Set when the file chosen to resume is not the interrupted one. */
  mismatch?: boolean;
  /** Named destination publication state, once every byte arrived. */
  publication?: "pending" | "failed" | "published";
};

/** Errors after which resuming is pointless: the server will not accept this upload. */
const FINAL_CODES = new Set([
  "upload_too_large",
  "invalid_upload",
  "invalid_video",
  "upload_expired",
  "upload_conflict",
  "video_path_unavailable",
  "not_found",
]);

let counter = 0;
const newKey = () => `transfer-${++counter}`;

/**
 * The transfers of the Import screen: one file at a time, in chunks, with progress, resume
 * and cancel. Uploads the server still holds (after a reload) are listed as interrupted.
 */
export function useTransfers() {
  const { session } = useSession();
  const { t } = useI18n();
  const csrf = session?.csrf_token ?? "";
  const [transfers, setTransfers] = useState<Transfer[]>([]);
  const controllers = useRef(new Map<string, AbortController>());
  const running = useRef(false);
  const latest = useRef<Transfer[]>([]);
  latest.current = transfers;

  const patch = useCallback((key: string, change: Partial<Transfer>) => {
    setTransfers((all) =>
      all.map((t) => (t.key === key ? { ...t, ...change } : t)),
    );
  }, []);

  useEffect(() => {
    if (!csrf) return;
    let current = true;
    void listUploads(csrf).then((result) => {
      if (!current || !result.ok) return;
      const held = result.value.filter((u) => u.status === "uploading");
      // A completed upload whose named destination never got published stays
      // visible so the publication can be retried or cancelled.
      const unpublished = result.value.filter(
        (u) => u.status === "complete" && u.publication !== "published",
      );
      setTransfers((all) => [
        ...held
          .filter((u) => !all.some((t) => t.upload?.id === u.id))
          .map<Transfer>((u) => ({
            key: newKey(),
            name: u.original_name,
            size: u.size_bytes,
            offset: u.offset_bytes,
            status: "interrupted",
            file: null,
            visibility: u.visibility,
            destination: u.destination,
            upload: u,
          })),
        ...unpublished
          .filter((u) => !all.some((t) => t.upload?.id === u.id))
          .map<Transfer>((u) => ({
            key: newKey(),
            name: u.original_name,
            size: u.size_bytes,
            offset: u.size_bytes,
            status: "failed",
            file: null,
            visibility: u.visibility,
            destination: u.destination,
            upload: u,
            code: u.publication_error || undefined,
            publication: u.publication,
          })),
        ...all,
      ]);
    });
    return () => {
      current = false;
    };
  }, [csrf]);

  const send = useCallback(
    async (transfer: Transfer) => {
      const file = transfer.file;
      if (!file) {
        patch(transfer.key, { status: "interrupted" });
        return;
      }
      const controller = new AbortController();
      controllers.current.set(transfer.key, controller);
      patch(transfer.key, {
        status: "sending",
        code: undefined,
        mismatch: undefined,
      });
      const stop = (code: string) =>
        patch(transfer.key, {
          status: FINAL_CODES.has(code) ? "failed" : "interrupted",
          code,
        });
      let upload = transfer.upload;
      if (!upload) {
        const target = `${transfer.destination ? `${transfer.destination}/` : "/"}${file.name}`;
        const created = await createUpload(
          csrf,
          file,
          transfer.visibility,
          transfer.destination,
          () => window.confirm(t("import.overwrite.confirm", { path: target })),
        );
        if (controller.signal.aborted) return;
        if (!created.ok) return stop(created.code);
        upload = created.value;
        patch(transfer.key, { upload, offset: upload.offset_bytes });
      }
      let offset = Math.max(upload.offset_bytes, transfer.offset);
      while (offset < file.size) {
        const sent = await putChunk(
          csrf,
          upload,
          file,
          offset,
          controller.signal,
        );
        if (controller.signal.aborted) return;
        if (!sent.ok) return stop(sent.code);
        offset = sent.value;
        patch(transfer.key, { offset });
      }
      patch(transfer.key, { status: "verifying" });
      const done = await completeUpload(csrf, upload.id);
      if (controller.signal.aborted) return;
      if (!done.ok) {
        // A publication conflict is reported with its stable server code.
        const code =
          done.code === "upload_conflict" ? "destination_changed" : done.code;
        patch(transfer.key, {
          status: "failed",
          code,
          publication: transfer.destination ? "failed" : undefined,
        });
        return;
      }
      patch(transfer.key, {
        status: "done",
        offset: file.size,
        upload: done.value,
        publication: transfer.destination ? "published" : undefined,
      });
    },
    [csrf, patch, t],
  );

  /** Takes the queued transfers one after the other. */
  const pump = useCallback(async () => {
    if (running.current) return;
    running.current = true;
    try {
      for (;;) {
        const next = latest.current.find((t) => t.status === "queued");
        if (!next) break;
        await send(next);
        controllers.current.delete(next.key);
      }
    } finally {
      running.current = false;
    }
  }, [send]);

  useEffect(() => {
    if (transfers.some((t) => t.status === "queued")) void pump();
  }, [transfers, pump]);

  const add = useCallback(
    (files: File[], visibility: Visibility, destination?: string) => {
      setTransfers((all) => [
        ...all,
        ...files.map<Transfer>((file) => ({
          key: newKey(),
          name: file.name,
          size: file.size,
          offset: 0,
          status: "queued",
          file,
          visibility,
          destination,
          upload: null,
        })),
      ]);
    },
    [],
  );

  /** Resumes an interrupted transfer, with the file when this page no longer holds it. */
  const resume = useCallback(
    (key: string, file?: File) => {
      const transfer = latest.current.find((t) => t.key === key);
      const chosen = file ?? transfer?.file;
      if (!transfer || !chosen) return;
      if (chosen.size !== transfer.size || chosen.name !== transfer.name) {
        patch(key, { mismatch: true });
        return;
      }
      patch(key, { file: chosen, status: "queued", mismatch: undefined });
    },
    [patch],
  );

  /** Retries publishing a completed upload whose destination was not written. */
  const retry = useCallback(
    async (key: string) => {
      const transfer = latest.current.find((t) => t.key === key);
      if (!transfer?.upload) return;
      patch(key, { status: "verifying", code: undefined });
      const done = await completeUpload(csrf, transfer.upload.id);
      if (!done.ok) {
        const code =
          done.code === "upload_conflict" ? "destination_changed" : done.code;
        patch(key, { status: "failed", code, publication: "failed" });
        return;
      }
      patch(key, {
        status: "done",
        code: undefined,
        upload: done.value,
        publication: "published",
      });
    },
    [csrf, patch],
  );

  const cancel = useCallback(
    async (key: string) => {
      const transfer = latest.current.find((t) => t.key === key);
      if (!transfer) return;
      controllers.current.get(key)?.abort();
      controllers.current.delete(key);
      patch(key, { status: "interrupted" });
      if (transfer.upload) {
        const result = await cancelUpload(csrf, transfer.upload.id);
        if (!result.ok && result.code !== "not_found") {
          patch(key, { status: "failed", code: result.code });
          return;
        }
      }
      setTransfers((all) => all.filter((t) => t.key !== key));
    },
    [csrf, patch],
  );

  const dismiss = useCallback((key: string) => {
    setTransfers((all) => all.filter((t) => t.key !== key));
  }, []);

  return { transfers, add, resume, retry, cancel, dismiss };
}
