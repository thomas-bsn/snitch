import { Dialog } from "@base-ui/react/dialog";
import { useEffect, useState, type FormEvent } from "react";
import { Link } from "react-router";
import { api } from "../../api/client";
import { buttonClass } from "../../components/buttons";
import { useI18n } from "../../i18n";
import type { MessageKey } from "../../i18n/messages";
import type { components } from "../../../api.gen";
import { IdentificationBadge } from "./AnalysisBoard";

type History = components["schemas"]["AnalysisRunHistory"];

/**
 * The durable record of every finished analysis run, steps and failures included: the only
 * paginated and searchable part of the queue.
 */
export function ClipHistory() {
  const { t, language } = useI18n();
  const tr = (fr: string, en: string) => (language === "fr" ? fr : en);
  const [open, setOpen] = useState(false);
  const [page, setPage] = useState(1);
  const [search, setSearch] = useState("");
  const [query, setQuery] = useState("");
  const [history, setHistory] = useState<History>();
  const [failed, setFailed] = useState(false);
  const [loading, setLoading] = useState(false);
  useEffect(() => {
    if (!open) return;
    let disposed = false;
    setLoading(true);
    setFailed(false);
    void api
      .GET("/api/admin/analysis/history", {
        params: { query: { page, q: query || undefined } },
      })
      .then((r) => {
        if (disposed) return;
        if (r.data) setHistory(r.data);
        else setFailed(true);
      })
      .catch(() => {
        if (!disposed) setFailed(true);
      })
      .finally(() => {
        if (!disposed) setLoading(false);
      });
    return () => {
      disposed = true;
    };
  }, [open, page, query]);
  function submit(event: FormEvent) {
    event.preventDefault();
    setPage(1);
    setQuery(search.trim());
  }
  const total = history?.total ?? 0;
  const pageSize = history?.page_size ?? 10;
  return (
    <>
      <button
        onClick={() => {
          setPage(1);
          setOpen(true);
        }}
        className={buttonClass()}
      >
        {tr("Historique des clips", "Clip history")}
      </button>
      <Dialog.Root open={open} onOpenChange={setOpen}>
        <Dialog.Portal>
          <Dialog.Backdrop className="fixed inset-0 z-[60] bg-black/50" />
          <Dialog.Viewport className="fixed inset-0 z-[70] flex items-center justify-center p-4">
            <Dialog.Popup className="flex max-h-[90dvh] w-full max-w-5xl flex-col overflow-hidden rounded-2xl border border-line bg-surface text-ink shadow-xl outline-none">
              <header className="flex items-center justify-between gap-4 border-b border-line p-5">
                <div>
                  <Dialog.Title className="text-lg font-semibold">
                    {tr("Historique des analyses", "Analysis history")}
                  </Dialog.Title>
                  <Dialog.Description className="text-sm text-mute">
                    {tr(
                      "Toutes les analyses terminées, avec leurs étapes et leurs échecs, dix par page.",
                      "Every finished analysis with its steps and failures, ten per page.",
                    )}
                  </Dialog.Description>
                </div>
                <Dialog.Close className="min-h-11 rounded-lg border border-line-strong px-4">
                  {tr("Fermer", "Close")}
                </Dialog.Close>
              </header>
              <form
                role="search"
                onSubmit={submit}
                className="flex flex-wrap items-end gap-2 border-b border-line px-5 py-3"
              >
                <label className="flex min-w-0 flex-1 flex-col gap-1 text-sm font-medium">
                  {tr("Rechercher un clip", "Search a clip")}
                  <input
                    type="search"
                    name="q"
                    value={search}
                    onChange={(e) => setSearch(e.target.value)}
                    className="h-11 rounded-lg border border-line-strong bg-surface px-3 text-sm text-ink md:h-9"
                  />
                </label>
                <button className={buttonClass()}>
                  {tr("Rechercher", "Search")}
                </button>
              </form>
              <div className="min-h-0 overflow-y-auto p-5">
                {loading ? (
                  <p role="status">{t("app.loading")}</p>
                ) : failed ? (
                  <p role="alert">
                    {tr(
                      "Impossible de charger l’historique.",
                      "Could not load history.",
                    )}
                  </p>
                ) : history && history.items.length === 0 ? (
                  <p className="m-0 text-mute">
                    {tr("Aucune analyse terminée.", "No finished analysis.")}
                  </p>
                ) : (
                  <ul className="m-0 list-none divide-y divide-line p-0">
                    {history?.items.map((run) => (
                      <li
                        key={run.id}
                        data-history-run={run.id}
                        className="py-3"
                      >
                        <Link
                          to={`/clips/${run.clip_id}`}
                          className="break-words font-semibold text-accent-ink underline"
                        >
                          {run.clip_name || `Clip #${run.clip_id}`}
                        </Link>
                        <p className="m-0 mt-1 text-sm text-mute">
                          {t(`queue.board.scope.${run.scope}`)} ·{" "}
                          {t(`queue.board.run.${run.status}`)} ·{" "}
                          {run.finished_at ?? run.updated_at}
                        </p>
                        <IdentificationBadge run={run} />
                        <ul className="m-0 mt-2 flex list-none flex-wrap gap-2 p-0">
                          {run.steps
                            .filter((s) => s.status !== "not_applicable")
                            .map((step) => (
                              <li
                                key={step.step}
                                data-history-step={step.step}
                                className={`rounded-lg border px-2 py-1 text-xs ${step.status === "failed" ? "border-warn-fg bg-warn-bg text-warn-fg" : "border-line"}`}
                              >
                                <strong>
                                  {t(`clip.analysis.step.${step.step}`)}
                                </strong>{" "}
                                · {t(`clip.analysis.status.${step.status}`)}
                                {step.status === "failed" &&
                                  ` · ${t(`clip.analysis.reason.${step.reason}` as MessageKey)}`}
                                {step.attempt > 1 &&
                                  ` · ${t("queue.board.attempt", { n: step.attempt })}`}
                              </li>
                            ))}
                        </ul>
                      </li>
                    ))}
                  </ul>
                )}
              </div>
              <footer className="flex flex-wrap items-center justify-between gap-3 border-t border-line p-5">
                <span>
                  {t("queue.page", { page })} · {total}{" "}
                  {tr("analyses", "analyses")}
                </span>
                <div className="flex gap-2">
                  <button
                    disabled={loading || page === 1}
                    onClick={() => setPage((p) => p - 1)}
                    className="min-h-11 rounded-lg border border-line-strong px-4 disabled:opacity-40"
                  >
                    {t("queue.previous")}
                  </button>
                  <button
                    disabled={loading || page * pageSize >= total}
                    onClick={() => setPage((p) => p + 1)}
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
