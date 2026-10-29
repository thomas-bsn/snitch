import { useEffect, useState } from "react";
import { Link } from "react-router";
import { ErrorAlert } from "../components/Alert";
import { buttonClass } from "../components/buttons";
import { useDocumentTitle } from "../components/useDocumentTitle";
import { useI18n } from "../i18n";
import { FolderPicker } from "../library/FolderPicker";
import { DropZone, looksLikeVideo } from "./DropZone";
import { TransferRow } from "./TransferRow";
import type { Visibility } from "./uploadApi";
import { useTransfers } from "./useTransfers";

/** Import: drop videos or choose them, follow each transfer, resume or cancel it. */
export function ImportPage() {
  const { t } = useI18n();
  const [root, setRoot] = useState<string>();
  const [destination, setDestination] = useState("");
  useEffect(() => {
    void fetch("/api/admin/video-root")
      .then((r) => (r.ok ? r.json() : null))
      .then((data) => setRoot(data?.root?.path))
      .catch(() => {});
  }, []);
  useDocumentTitle(t("pages.import.title"));
  const { transfers, add, resume, retry, cancel, dismiss } = useTransfers();
  const [visibility, setVisibility] = useState<Visibility>("shared");
  const [rejected, setRejected] = useState<string[]>([]);

  function receive(files: File[]) {
    const videos = files.filter(looksLikeVideo);
    setRejected(files.filter((f) => !looksLikeVideo(f)).map((f) => f.name));
    if (videos.length > 0)
      add(videos, visibility, root ? destination : undefined);
  }

  const done = transfers.filter((x) => x.status === "done");

  return (
    <section className="flex flex-col gap-5 px-4 py-6 md:px-7">
      <h1 className="text-[22px] font-semibold">{t("pages.import.title")}</h1>

      <div className="flex max-w-xl flex-col gap-1.5">
        <label htmlFor="import-visibility" className="font-medium">
          {t("import.visibility.label")}
        </label>
        <select
          id="import-visibility"
          value={visibility}
          onChange={(event) => setVisibility(event.target.value as Visibility)}
          className="h-11 rounded-lg border border-line-strong bg-surface px-3 text-ink md:h-10"
        >
          <option value="shared">{t("import.visibility.shared")}</option>
          <option value="private">{t("import.visibility.private")}</option>
        </select>
      </div>

      {root && <FolderPicker value={destination} onChange={setDestination} />}
      <DropZone onFiles={receive} />

      {rejected.map((name) => (
        <ErrorAlert key={name}>{t("import.notVideo", { name })}</ErrorAlert>
      ))}

      {transfers.length > 0 && (
        <ul aria-label={t("import.list.label")} className="flex flex-col gap-3">
          {transfers.map((transfer) => (
            <TransferRow
              key={transfer.key}
              transfer={transfer}
              onResume={resume}
              onRetry={(key) => void retry(key)}
              onCancel={(key) => void cancel(key)}
              onDismiss={dismiss}
            />
          ))}
        </ul>
      )}

      <div className="sr-only" role="status">
        {done.map((x) => (
          <p key={x.key}>{t("import.doneAnnouncement", { name: x.name })}</p>
        ))}
      </div>

      {done.length > 0 && (
        <div>
          <Link to="/library" className={buttonClass("secondary")}>
            {t("import.viewInLibrary")}
          </Link>
        </div>
      )}
    </section>
  );
}
