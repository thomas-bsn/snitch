import { buttonClass } from "../components/buttons";
import { useI18n } from "../i18n";
import type { RootScan, RootScanPreview } from "./rootScanApi";

type Counts = Pick<
  RootScanPreview,
  "up_to_date" | "excluded" | "unreadable_folders" | "unreadable_paths"
> & { queued: number };

/** Counts of a scan of the whole Racine (#404): what is queued, up to date, and left out by reason. */
export function RootScanCounts({ counts }: { counts: Counts }) {
  const { t } = useI18n();
  const rows: [string, number][] = [
    [t("videoStorage.rootScan.queued"), counts.queued],
    [t("videoStorage.rootScan.upToDate"), counts.up_to_date],
    [
      t("videoStorage.rootScan.privateFolders"),
      counts.excluded.private_folders,
    ],
    [t("videoStorage.rootScan.hidden"), counts.excluded.hidden],
    [t("videoStorage.rootScan.system"), counts.excluded.system],
    [t("videoStorage.rootScan.privateClips"), counts.excluded.private_clips],
    [t("videoStorage.rootScan.unreadable"), counts.unreadable_folders],
  ];
  return (
    <>
      <dl className="grid max-w-md grid-cols-[1fr_auto] gap-x-4 gap-y-1 text-sm">
        {rows.map(([label, value]) => (
          <div key={label} className="contents">
            <dt>{label}</dt>
            <dd className="text-right tabular-nums">{value}</dd>
          </div>
        ))}
      </dl>
      {counts.unreadable_paths.length > 0 && (
        <p className="break-all text-sm text-mute">
          {t("videoStorage.rootScan.unreadablePaths", {
            paths: counts.unreadable_paths
              .map((p) => p || t("videoStorage.rootScan.rootItself"))
              .join(", "),
          })}
        </p>
      )}
    </>
  );
}

/** Progress, then report, of the latest scan of the whole Racine. */
export function RootScanProgress({
  run,
  busy,
  onCancel,
}: {
  run: RootScan;
  busy: boolean;
  onCancel: () => void;
}) {
  const { t, error } = useI18n();
  const running = run.status === "running";
  let headline: string;
  if (run.status === "cancelled")
    headline = t("videoStorage.rootScan.cancelled");
  else if (run.status === "completed")
    headline = t("videoStorage.rootScan.completed");
  else if (run.walking)
    headline = t("videoStorage.rootScan.walking", {
      seen: run.seen,
      queued: run.queued,
    });
  else
    headline = t("videoStorage.rootScan.queueing", {
      done: run.done,
      queued: run.queued,
    });
  return (
    <section
      aria-label={t("videoStorage.rootScan.title")}
      className="flex flex-col gap-2 rounded-lg border border-line p-4"
    >
      <h3 className="font-semibold">{t("videoStorage.rootScan.title")}</h3>
      <p role="status">{headline}</p>
      {running && !run.walking && run.queued > 0 && (
        <progress
          className="w-full max-w-md"
          max={run.queued}
          value={run.done}
          aria-label={t("videoStorage.rootScan.progressLabel")}
        />
      )}
      {running && run.waiting_for_worker && (
        <p className="text-sm text-mute">
          {t("videoStorage.rootScan.waitingForWorker")}
        </p>
      )}
      <RootScanCounts counts={run} />
      {run.failed > 0 && (
        <details className="text-sm">
          <summary>
            {t("videoStorage.rootScan.failed", { count: run.failed })}
          </summary>
          <ul>
            {run.failures.map((f) => (
              <li key={f.path} className="break-all py-1">
                {f.path} — {error(f.code)}
              </li>
            ))}
          </ul>
        </details>
      )}
      {running && (
        <div>
          <button
            type="button"
            disabled={busy}
            className={buttonClass("secondary")}
            onClick={onCancel}
          >
            {t("videoStorage.rootScan.cancel")}
          </button>
        </div>
      )}
    </section>
  );
}
