import { Dialog } from "@base-ui/react/dialog";
import { useEffect, useState } from "react";
import { Link } from "react-router";
import { api } from "../../api/client";
import { buttonClass } from "../../components/buttons";
import { useI18n } from "../../i18n";
import type { components } from "../../../api.gen";
type Job = components["schemas"]["Job"];

/** The durable list of clips whose tasks all finished: the only paginated part of the queue. */
export function ClipHistory() {
  const { t, language } = useI18n();
  const tr = (fr: string, en: string) => (language === "fr" ? fr : en);
  const [historyOpen, setHistoryOpen] = useState(false);
  const [historyPage, setHistoryPage] = useState(1);
  const [completed, setCompleted] = useState<{ items: Job[]; total: number }>({
    items: [],
    total: 0,
  });
  const [titles, setTitles] = useState<Record<number, string>>({});
  const [historyError, setHistoryError] = useState(false);
  const [loadingHistory, setLoadingHistory] = useState(false);
  useEffect(() => {
    if (!historyOpen) return;
    let disposed = false;
    setLoadingHistory(true);
    setHistoryError(false);
    void api
      .GET("/api/admin/jobs", {
        params: {
          query: { workflow: "done", page: historyPage, page_size: 10 },
        },
      })
      .then(async (r) => {
        if (disposed) return;
        if (!r.data) {
          setHistoryError(true);
          return;
        }
        setCompleted({ items: r.data.items, total: r.data.total });
        const ids = [...new Set(r.data.items.map((j) => j.clip_id))];
        const entries = await Promise.all(
          ids.map(async (id) => {
            try {
              const clip = await api.GET("/api/clips/{id}", {
                params: { path: { id } },
              });
              return [id, clip.data?.original_name ?? `Clip #${id}`] as const;
            } catch {
              return [id, `Clip #${id}`] as const;
            }
          }),
        );
        if (!disposed) setTitles(Object.fromEntries(entries));
      })
      .catch(() => {
        if (!disposed) setHistoryError(true);
      })
      .finally(() => {
        if (!disposed) setLoadingHistory(false);
      });
    return () => {
      disposed = true;
    };
  }, [historyOpen, historyPage]);
  return (
    <>
      <button
        onClick={() => {
          setHistoryPage(1);
          setHistoryOpen(true);
        }}
        className={buttonClass()}
      >
        {tr("Historique des clips", "Clip history")}
      </button>
      <Dialog.Root open={historyOpen} onOpenChange={setHistoryOpen}>
        <Dialog.Portal>
          <Dialog.Backdrop className="fixed inset-0 z-[60] bg-black/50" />
          <Dialog.Viewport className="fixed inset-0 z-[70] flex items-center justify-center p-4">
            <Dialog.Popup className="flex max-h-[90dvh] w-full max-w-5xl flex-col overflow-hidden rounded-2xl border border-line bg-surface text-ink shadow-xl outline-none">
              <header className="flex items-center justify-between gap-4 border-b border-line p-5">
                <div>
                  <Dialog.Title className="text-lg font-semibold">
                    {tr(
                      "Historique des clips terminés",
                      "Completed clip history",
                    )}
                  </Dialog.Title>
                  <Dialog.Description className="text-sm text-mute">
                    {tr(
                      "Clips dont tous les traitements planifiés ont réussi, dix par page.",
                      "Clips whose scheduled tasks have all succeeded, ten per page.",
                    )}
                  </Dialog.Description>
                </div>
                <Dialog.Close className="min-h-11 rounded-lg border border-line-strong px-4">
                  {tr("Fermer", "Close")}
                </Dialog.Close>
              </header>
              <div className="min-h-0 overflow-y-auto p-5">
                {loadingHistory ? (
                  <p role="status">{t("app.loading")}</p>
                ) : historyError ? (
                  <p role="alert">
                    {tr(
                      "Impossible de charger l’historique.",
                      "Could not load history.",
                    )}
                  </p>
                ) : (
                  <ul className="divide-y divide-line">
                    {completed?.items.map((job) => (
                      <li key={job.id} className="py-3">
                        <Link
                          to={`/clips/${job.clip_id}`}
                          className="break-words font-semibold text-accent-ink underline"
                        >
                          {titles[job.clip_id] ?? `Clip #${job.clip_id}`}
                        </Link>
                        <p className="mt-1 text-sm text-mute">
                          {job.step} · {t(`queue.status.${job.status}`)} ·{" "}
                          {job.updated_at}
                        </p>
                      </li>
                    ))}
                  </ul>
                )}
              </div>
              <footer className="flex flex-wrap items-center justify-between gap-3 border-t border-line p-5">
                <span>
                  {t("queue.page", { page: historyPage })} ·{" "}
                  {completed?.total ?? 0} clips
                </span>
                <div className="flex gap-2">
                  <button
                    disabled={loadingHistory || historyPage === 1}
                    onClick={() => setHistoryPage((p) => p - 1)}
                    className="min-h-11 rounded-lg border border-line-strong px-4 disabled:opacity-40"
                  >
                    {t("queue.previous")}
                  </button>
                  <button
                    disabled={
                      loadingHistory ||
                      !completed ||
                      historyPage * 10 >= completed.total
                    }
                    onClick={() => setHistoryPage((p) => p + 1)}
                    className="min-h-11 rounded-lg border border-line-strong px-4 disabled:opacity-40"
                  >
                    {t("queue.next")}
                  </button>
                </div>
              </footer>
            </Dialog.Popup>
          </Dialog.Viewport>
        </Dialog.Portal>
      </Dialog.Root>
    </>
  );
}
