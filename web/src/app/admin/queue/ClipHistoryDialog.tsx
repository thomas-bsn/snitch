import { Dialog } from "@base-ui/react/dialog";
import { useEffect, useState } from "react";
import { Link } from "react-router";
import { api } from "../../api/client";
import { buttonClass } from "../../components/buttons";
import { useI18n } from "../../i18n";
import type { components } from "../../../api.gen";

type HistoryPage = components["schemas"]["ClipHistoryPage"];
type HistoryStep = components["schemas"]["ClipHistoryStep"];

const PAGE_SIZE = 10;

const actionLabels: Record<string, [string, string]> = {
  check_worker_logs: [
    "Consulter les journaux du Worker.",
    "Check the Worker logs.",
  ],
  check_transcription_service: [
    "Vérifier le service de transcription.",
    "Check the transcription service.",
  ],
  check_clip_media: ["Vérifier le média du clip.", "Check the clip media."],
  check_worker_connection: [
    "Vérifier la connexion du Worker.",
    "Check the Worker connection.",
  ],
  relaunch_job: ["Relancer la tâche.", "Relaunch the task."],
  install_transcription_model: [
    "Installer le modèle de transcription.",
    "Install the transcription model.",
  ],
  install_voice_model: [
    "Installer le modèle vocal.",
    "Install the voice model.",
  ],
  check_huggingface_token: [
    "Vérifier le jeton Hugging Face.",
    "Check the Hugging Face token.",
  ],
  check_gpu: ["Vérifier le GPU.", "Check the GPU."],
  check_voice_model_repository: [
    "Vérifier le dépôt du modèle vocal.",
    "Check the voice model repository.",
  ],
  check_worker_network: [
    "Vérifier le réseau du Worker.",
    "Check the Worker network.",
  ],
  check_models_volume: [
    "Vérifier le volume des modèles.",
    "Check the models volume.",
  ],
};

/**
 * Historique du clip: per clip, the timeline of the events of every step.
 * Administrators only (the caller sits under the admin area).
 */
export function ClipHistoryDialog({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const { t, language } = useI18n();
  const tr = (fr: string, en: string) => (language === "fr" ? fr : en);
  const [page, setPage] = useState(1);
  const [draft, setDraft] = useState("");
  const [search, setSearch] = useState("");
  const [data, setData] = useState<HistoryPage>();
  const [failed, setFailed] = useState(false);
  const [loading, setLoading] = useState(false);
  useEffect(() => {
    if (!open) return;
    let disposed = false;
    setLoading(true);
    setFailed(false);
    void api
      .GET("/api/admin/clip-history", {
        params: {
          query: { q: search || undefined, page, page_size: PAGE_SIZE },
        },
      })
      .then((r) => {
        if (disposed) return;
        if (r.data) setData(r.data);
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
  }, [open, page, search]);
  const total = data?.total ?? 0;
  return (
    <Dialog.Root open={open} onOpenChange={onOpenChange}>
      <Dialog.Portal>
        <Dialog.Backdrop className="fixed inset-0 z-[60] bg-black/50" />
        <Dialog.Viewport className="fixed inset-0 z-[70] flex items-center justify-center p-4">
          <Dialog.Popup
            data-clip-history
            className="flex max-h-[90dvh] w-full max-w-5xl flex-col overflow-hidden rounded-2xl border border-line bg-surface text-ink shadow-xl outline-none"
          >
            <header className="flex items-center justify-between gap-4 border-b border-line p-5">
              <div>
                <Dialog.Title className="text-lg font-semibold">
                  {tr("Historique du clip", "Clip history")}
                </Dialog.Title>
                <Dialog.Description className="text-sm text-mute">
                  {tr(
                    "Les événements de chaque étape, clip par clip, dix clips par page.",
                    "The events of each step, clip by clip, ten clips per page.",
                  )}
                </Dialog.Description>
              </div>
              <Dialog.Close className="min-h-11 rounded-lg border border-line-strong px-4">
                {tr("Fermer", "Close")}
              </Dialog.Close>
            </header>
            <form
              role="search"
              className="flex flex-wrap items-end gap-2 border-b border-line px-5 py-3"
              onSubmit={(event) => {
                event.preventDefault();
                setPage(1);
                setSearch(draft.trim());
              }}
            >
              <label className="flex min-w-0 flex-1 flex-col gap-1 text-sm font-medium">
                {tr("Rechercher un clip", "Search a clip")}
                <input
                  type="search"
                  name="clip-search"
                  value={draft}
                  maxLength={200}
                  onChange={(event) => setDraft(event.target.value)}
                  placeholder={tr(
                    "Nom ou numéro du clip",
                    "Clip name or number",
                  )}
                  className="min-h-11 rounded-lg border border-line-strong bg-surface px-3"
                />
              </label>
              <button type="submit" className={buttonClass()}>
                {tr("Rechercher", "Search")}
              </button>
            </form>
            <div className="min-h-0 overflow-y-auto p-5">
              {loading && !data ? (
                <p role="status">{t("app.loading")}</p>
              ) : failed ? (
                <p role="alert">
                  {tr(
                    "Impossible de charger l’historique.",
                    "Could not load history.",
                  )}
                </p>
              ) : total === 0 ? (
                <p role="status">
                  {tr("Aucun clip trouvé.", "No clip found.")}
                </p>
              ) : (
                <ul className="divide-y divide-line">
                  {data?.items.map((clip) => (
                    <li
                      key={clip.clip_id}
                      data-history-clip={clip.clip_id}
                      className="py-4"
                    >
                      <Link
                        to={`/clips/${clip.clip_id}`}
                        className="break-words font-semibold text-accent-ink underline"
                      >
                        {clip.original_name || `Clip #${clip.clip_id}`}
                      </Link>
                      <span className="ml-2 text-sm text-mute">
                        #{clip.clip_id}
                      </span>
                      <div className="mt-2 space-y-3">
                        {clip.steps.map((step) => (
                          <StepTimeline key={step.job_id} step={step} />
                        ))}
                      </div>
                    </li>
                  ))}
                </ul>
              )}
            </div>
            <footer className="flex flex-wrap items-center justify-between gap-3 border-t border-line p-5">
              <span>
                {t("queue.page", { page })} · {total} clips
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
                  disabled={loading || page * PAGE_SIZE >= total}
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
  );
}

function StepTimeline({ step }: { step: HistoryStep }) {
  const { t, language } = useI18n();
  const tr = (fr: string, en: string) => (language === "fr" ? fr : en);
  return (
    <section className="rounded-lg border border-line p-3 text-sm">
      <p className="font-medium">
        {step.step} ·{" "}
        {t(
          `queue.status.${step.status as components["schemas"]["Job"]["status"]}`,
        )}
        <span className="ml-2 font-normal text-mute">
          {tr("tentatives", "attempts")} {step.attempts}/{step.max_attempts} ·{" "}
          {tr("cycle", "cycle")} {step.cycle}
          {step.worker ? ` · Worker ${step.worker}` : ""}
        </span>
      </p>
      {step.events.length === 0 ? (
        <p className="mt-1 text-mute">
          {tr("Aucun événement enregistré.", "No event recorded.")}
        </p>
      ) : (
        <ol className="mt-2 space-y-1 border-l border-line pl-3">
          {step.events.map((event) => {
            const action = event.recommended_action
              ? actionLabels[event.recommended_action]
              : undefined;
            return (
              <li key={event.id} className="break-words">
                <time dateTime={event.created_at} className="text-mute">
                  {new Date(event.created_at).toLocaleString(language)}
                </time>{" "}
                · {t(`queue.event.${event.event}`)}
                {event.attempt != null &&
                  ` · ${tr("tentative", "attempt")} ${event.attempt}`}
                {event.model && ` · ${tr("modèle", "model")} ${event.model}`}
                {event.actor_id && ` · ${event.actor_id}`}
                {event.error_code && (
                  <span className="ml-1 rounded bg-warn-bg px-1.5 py-0.5 text-warn-fg">
                    {tr("cause", "cause")} {event.error_code}
                    {action ? ` — ${tr(...action)}` : ""}
                  </span>
                )}
              </li>
            );
          })}
        </ol>
      )}
    </section>
  );
}
