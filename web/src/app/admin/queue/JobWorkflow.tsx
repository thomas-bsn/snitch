import { Dialog } from "@base-ui/react/dialog";
import { useEffect, useState, type ReactNode } from "react";
import { Link } from "react-router";
import {
  AudioLines,
  Search,
  Users,
  ScanFace,
  CheckCheck,
  TriangleAlert,
  ArrowRight,
  ChevronDown,
} from "lucide-react";
import { api } from "../../api/client";
import { buttonClass } from "../../components/buttons";
import { useI18n } from "../../i18n";
import { actionText, causeText } from "./failureText";
import type { components } from "../../../api.gen";
type Job = components["schemas"]["Job"];
export type TaskCard = components["schemas"]["TaskCard"];
export type TaskCardBoard = components["schemas"]["TaskCardBoard"];
type Column = TaskCard["column"];
type State = TaskCard["state"];
/** What the card actions need from a task: its identity and raw status. */
export type JobRef = { id: number; status: Job["status"] };
const columnDefinitions: { key: Column; icon: typeof AudioLines }[] = [
  { key: "transcription", icon: AudioLines },
  { key: "search", icon: Search },
  { key: "voices", icon: Users },
  { key: "recognition", icon: ScanFace },
  { key: "completed", icon: CheckCheck },
];
const attention: State[] = [
  "queued",
  "waiting_worker",
  "retry",
  "failed",
  "dead",
  "cancelled",
];
function stateLabel(state: State, language: string) {
  const fr: Record<State, string> = {
    queued: "En file",
    waiting_worker: "En attente de Worker",
    running: "En cours",
    retry: "Nouvel essai",
    failed: "Échec",
    dead: "Épuisé",
    cancelled: "Annulé",
    ready: "Prêt à planifier",
    waiting: "En attente",
    done: "Terminé",
    not_applicable: "Sans objet",
  };
  const en: Record<State, string> = {
    queued: "Queued",
    waiting_worker: "Waiting for a Worker",
    running: "Running",
    retry: "Retrying",
    failed: "Failed",
    dead: "Exhausted",
    cancelled: "Cancelled",
    ready: "Ready to schedule",
    waiting: "Waiting",
    done: "Done",
    not_applicable: "Not applicable",
  };
  return (language === "fr" ? fr : en)[state];
}
function stepLabel(key: TaskCard["current_step"], language: string) {
  const fr = {
    transcription: "Paroles en texte",
    diarization: "Voix distinctes",
    fingerprint: "Empreintes vocales",
    recognition: "Reconnaissance vocale",
  };
  const en = {
    transcription: "Speech to text",
    diarization: "Distinct voices",
    fingerprint: "Voice fingerprints",
    recognition: "Voice recognition",
  };
  return (language === "fr" ? fr : en)[key];
}
function ClipArtwork({ id }: { id: number }) {
  return (
    <svg
      viewBox="0 0 80 64"
      className="h-12 w-14 shrink-0 rounded-md"
      aria-hidden="true"
    >
      <rect width="80" height="64" fill={id % 2 ? "#172a38" : "#24233c"} />
      <path
        d="M0 48 20 28 32 38 49 14 80 42V64H0Z"
        fill={id % 2 ? "#345769" : "#514b77"}
      />
      <path d="M0 57 27 41 45 53 63 32 80 46V64H0Z" fill="#101827" />
      <path
        d="M39 22v8m0 10v8M26 35h8m10 0h8"
        stroke="#e9e8f4"
        strokeWidth="1.5"
        opacity=".8"
      />
      <rect x="5" y="5" width="15" height="3" rx="1.5" fill="#c4b5fd" />
    </svg>
  );
}

export function JobWorkflow({
  board,
  actions,
  emptyMessage,
}: {
  board: TaskCardBoard;
  actions: (job: JobRef) => ReactNode;
  emptyMessage?: string;
}) {
  const { t, language } = useI18n();
  const tr = (fr: string, en: string) => (language === "fr" ? fr : en);
  const labels: Record<Column, [string, string]> = {
    transcription: ["Paroles en texte", "Speech to text"],
    search: ["Recherche", "Search"],
    voices: ["Voix distinctes", "Distinct voices"],
    recognition: ["Reconnaissance vocale", "Voice recognition"],
    completed: ["Terminé", "Completed"],
  };
  const descriptions: Record<Column, [string, string]> = {
    transcription: [
      "Écrire ce qui est dit dans la vidéo.",
      "Write down the speech in the video.",
    ],
    search: [
      "Retrouver le clip à partir de ses paroles.",
      "Find the clip by its spoken content.",
    ],
    voices: [
      "Distinguer les personnes qui parlent.",
      "Separate the voices in the clip.",
    ],
    recognition: [
      "Comparer les voix aux personnes connues.",
      "Compare voices with known people.",
    ],
    completed: ["Les résultats sont disponibles.", "Results are available."],
  };
  const game = (title: string) =>
    title.match(/^(.+?) \d{4}\.\d{2}\.\d{2} - \d{2}\.\d{2}\.\d{2}/)?.[1];
  const [historyOpen, setHistoryOpen] = useState(false);
  const [historyPage, setHistoryPage] = useState(1);
  const [completed, setCompleted] = useState<{ items: Job[]; total: number }>({
    items: [],
    total: 0,
  });
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
      .then((r) => {
        if (disposed) return;
        if (!r.data) {
          setHistoryError(true);
          return;
        }
        setCompleted({ items: r.data.items, total: r.data.total });
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
  const [titles, setTitles] = useState<Record<number, string>>({});
  const ids = [...new Set(completed.items.map((j) => j.clip_id))]
    .sort((a, b) => a - b)
    .join(",");
  useEffect(() => {
    if (!historyOpen) return;
    let disposed = false;
    const clipIds = ids ? ids.split(",").map(Number) : [];
    void Promise.all(
      clipIds.map(async (id) => {
        try {
          const result = await api.GET("/api/clips/{id}", {
            params: { path: { id } },
          });
          return [id, result.data?.original_name ?? `Clip #${id}`] as const;
        } catch {
          return [id, `Clip #${id}`] as const;
        }
      }),
    ).then((entries) => {
      if (!disposed) setTitles(Object.fromEntries(entries));
    });
    return () => {
      disposed = true;
    };
  }, [ids, historyOpen]);
  const all = [...board.cards, ...board.completed];
  return (
    <>
      <div className="mb-3 mt-1 flex flex-wrap items-end justify-end gap-3">
        <button
          onClick={() => {
            setHistoryPage(1);
            setHistoryOpen(true);
          }}
          className={buttonClass()}
        >
          {tr("Historique des clips", "Clip history")}
        </button>
      </div>
      {all.length === 0 && <p>{emptyMessage}</p>}
      <div
        data-workflow-grid
        className="grid items-stretch gap-3 sm:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-5"
      >
        {columnDefinitions.map((column, index) => {
          const cards = all.filter((card) => card.column === column.key);
          return (
            <section
              key={column.key}
              data-column={column.key}
              className="min-w-0 rounded-xl border border-line bg-side p-4"
            >
              <div className="mb-4 flex items-center justify-between gap-2">
                <span
                  className={`relative grid size-12 place-items-center rounded-xl border border-line shadow-sm ${column.key === "completed" ? "bg-ok-bg text-ok-fg" : "bg-surface text-accent-ink"}`}
                >
                  <column.icon size={22} strokeWidth={1.5} aria-hidden="true" />
                  <span className="absolute -bottom-1 -right-1 grid size-5 place-items-center rounded-full border border-line bg-surface text-xs text-mute">
                    {index + 1}
                  </span>
                </span>
                {column.key !== "completed" && (
                  <ArrowRight
                    size={18}
                    className="text-accent-ink"
                    aria-hidden="true"
                  />
                )}
              </div>
              <h3 className="mt-4 text-base font-semibold">
                {tr(...labels[column.key])} · {cards.length}
              </h3>
              <p className="mb-3 mt-1 min-h-10 text-sm leading-5 text-mute">
                {tr(...descriptions[column.key])}
              </p>
              <div className="max-h-none space-y-2 overflow-visible 2xl:max-h-[340px] 2xl:overflow-y-auto">
                {cards.map((card) => (
                  <details
                    key={card.clip_id}
                    id={`clip-${card.clip_id}`}
                    data-state={card.state}
                    className={`group rounded-xl border bg-surface shadow-sm transition-shadow hover:shadow-md motion-reduce:transition-none ${attention.includes(card.state) ? "border-warn-fg" : "border-line"}`}
                  >
                    <summary className="relative flex min-h-24 cursor-pointer list-none items-start gap-3 p-2.5 pr-2.5">
                      <ClipArtwork id={card.clip_id} />
                      <div className="min-w-0 flex-1">
                        <span className="block break-words text-sm font-medium leading-5">
                          {card.original_name || `Clip #${card.clip_id}`}
                        </span>
                        <span className="mt-1 block text-xs leading-4 text-mute">
                          {game(card.original_name)
                            ? `${game(card.original_name)} · `
                            : ""}
                          #{card.clip_id} · {stateLabel(card.state, language)}
                        </span>
                        {(attention.includes(card.state) ||
                          card.unknown_step) && (
                          <TriangleAlert
                            size={17}
                            className="absolute right-2 top-2 text-warn-fg"
                            aria-label={
                              card.unknown_step
                                ? tr("Étape inconnue", "Unknown step")
                                : stateLabel(card.state, language)
                            }
                          />
                        )}
                      </div>
                      <ChevronDown
                        size={14}
                        className="absolute bottom-3 right-2 text-mute transition-transform group-open:rotate-180 motion-reduce:transition-none"
                        aria-hidden="true"
                      />
                    </summary>
                    <div className="space-y-3 border-t border-line p-3 text-sm">
                      <Link
                        to={`/clips/${card.clip_id}`}
                        className="text-accent-ink underline"
                      >
                        {tr("Ouvrir le clip", "Open clip")}
                      </Link>
                      {card.recommended_action && (
                        <div
                          data-advice
                          className="space-y-1 rounded-lg bg-warn-bg p-2 text-warn-fg"
                        >
                          {card.failure_code && (
                            <p>
                              <strong>
                                {causeText(card.failure_code, language)}
                              </strong>{" "}
                              <code>{card.failure_code}</code>
                            </p>
                          )}
                          <p>
                            {tr("Action conseillée : ", "Advised action: ")}
                            {actionText(card.recommended_action, language)}
                          </p>
                        </div>
                      )}
                      {card.unknown_step && (
                        <p
                          data-unknown-step
                          role="alert"
                          className="break-words rounded-lg bg-warn-bg p-2 text-warn-fg"
                        >
                          {tr("Étape inconnue", "Unknown step")} ·{" "}
                          <code>{card.unknown_step.step}</code>
                          {card.unknown_step.last_error &&
                            ` · ${card.unknown_step.last_error}`}
                        </p>
                      )}
                      {card.last_error && (
                        <p className="break-words rounded-lg bg-warn-bg p-2 text-warn-fg">
                          {card.last_error}
                        </p>
                      )}
                      {card.steps
                        .filter((step) => step.state !== "not_applicable")
                        .map((step) => (
                          <div
                            key={step.key}
                            className="space-y-2 border-t border-line pt-2"
                          >
                            <p>
                              {stepLabel(step.key, language)} ·{" "}
                              {stateLabel(step.state, language)}
                            </p>
                            {step.job_id != null &&
                              step.job_status != null &&
                              actions({
                                id: step.job_id,
                                status: step.job_status as Job["status"],
                              })}
                          </div>
                        ))}
                    </div>
                  </details>
                ))}
              </div>
            </section>
          );
        })}
      </div>
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
                    {completed.items.map((job) => (
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
                  {t("queue.page", { page: historyPage })} · {completed.total}{" "}
                  clips
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
                      loadingHistory || historyPage * 10 >= completed.total
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
