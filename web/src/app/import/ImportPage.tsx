import { useEffect, useRef, useState } from "react";
import { Link } from "react-router";
import { ErrorAlert } from "../components/Alert";
import { buttonClass } from "../components/buttons";
import { useDocumentTitle } from "../components/useDocumentTitle";
import { useI18n } from "../i18n";
import { FolderPicker } from "../library/FolderPicker";
import { DropZone, looksLikeVideo } from "./DropZone";
import { TransferRow } from "./TransferRow";
import type { Visibility } from "./uploadApi";
import { ServerSelection } from "./ServerSelection";
import { useTransfers } from "./useTransfers";

/** Import: drop videos or choose them, follow each transfer, resume or cancel it. */
export function ImportPage() {
  const { t } = useI18n();
  const [root, setRoot] = useState<string>();
  // Undefined until a folder was chosen on purpose: the video root is never picked implicitly.
  const [destination, setDestination] = useState<string>();
  const [blocked, setBlocked] = useState(false);
  const [missing, setMissing] = useState<string>();
  const handled = useRef(new Set<string>());
  // A drop can arrive before the video root is known: it waits for it, then follows the same rule.
  const rootLoad = useRef<Promise<string | undefined>>(undefined);
  useEffect(() => {
    rootLoad.current = fetch("/api/admin/video-root")
      .then((r) => (r.ok ? r.json() : null))
      .then((data) => data?.root?.path as string | undefined)
      .catch(() => undefined);
    void rootLoad.current.then(setRoot);
  }, []);
  const chosen = useRef<string | undefined>(undefined);
  useDocumentTitle(t("pages.import.title"));
  const { transfers, add, resume, retry, cancel, dismiss } = useTransfers();
  const [visibility, setVisibility] = useState<Visibility>("shared");
  const [rejected, setRejected] = useState<string[]>([]);

  // A folder deleted or moved after it was chosen: sending stops and a new choice is required.
  useEffect(() => {
    const lost = transfers.find(
      (x) =>
        x.status === "failed" &&
        x.code === "video_path_unavailable" &&
        !handled.current.has(x.key),
    );
    if (!lost) return;
    handled.current.add(lost.key);
    if (lost.destination === destination) {
      setMissing(lost.destination ?? "");
      setDestination(undefined);
    }
  }, [transfers, destination]);

  function chooseDestination(path: string) {
    setDestination(path);
    setBlocked(false);
    setMissing(undefined);
  }

  chosen.current = destination;

  async function receive(files: File[]) {
    const known = await rootLoad.current;
    if (known && chosen.current === undefined) {
      setBlocked(true);
      return;
    }
    const videos = files.filter(looksLikeVideo);
    setRejected(files.filter((f) => !looksLikeVideo(f)).map((f) => f.name));
    if (videos.length > 0)
      add(videos, visibility, known ? chosen.current : undefined);
  }

  const done = transfers.filter((x) => x.status === "done");

  return (
    <section className="flex flex-col gap-8 px-4 py-6 md:px-7">
      <h1 className="text-[22px] font-semibold">{t("pages.import.title")}</h1>

      <ServerSelection />

      <section
        aria-labelledby="import-computer"
        className="flex flex-col gap-5 border-t border-line pt-8"
      >
        <h2 id="import-computer" className="text-lg font-semibold">
          {t("import.sections.computer")}
        </h2>
        <div className="flex max-w-xl flex-col gap-1.5">
          <label htmlFor="import-visibility" className="font-medium">
            {t("import.visibility.label")}
          </label>
          <select
            id="import-visibility"
            value={visibility}
            onChange={(event) =>
              setVisibility(event.target.value as Visibility)
            }
            className="h-11 rounded-lg border border-line-strong bg-surface px-3 text-ink md:h-10"
          >
            <option value="shared">{t("import.visibility.shared")}</option>
            <option value="private">{t("import.visibility.private")}</option>
          </select>
        </div>

        {root && (
          <FolderPicker value={destination} onChange={chooseDestination} />
        )}
        {missing !== undefined && (
          <ErrorAlert>
            {t("import.destination.missing", { path: `/${missing}` })}
          </ErrorAlert>
        )}
        <DropZone
          onFiles={(files) => void receive(files)}
          blocked={Boolean(root) && destination === undefined}
          onBlocked={() => setBlocked(true)}
        />
        {blocked && <ErrorAlert>{t("import.destination.required")}</ErrorAlert>}

        {rejected.map((name) => (
          <ErrorAlert key={name}>{t("import.notVideo", { name })}</ErrorAlert>
        ))}

        {transfers.length > 0 && (
          <ul
            aria-label={t("import.list.label")}
            className="flex flex-col gap-3"
          >
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
    </section>
  );
}
