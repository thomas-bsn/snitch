import { useState, type ReactNode } from "react";
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
import { buttonClass } from "../../components/buttons";
import { useI18n } from "../../i18n";
import { ClipHistoryDialog } from "./ClipHistoryDialog";
import { actionText, causeText } from "./failureText";
import {
  DEFAULT_THUMBNAIL_SIZE,
  THUMBNAIL_SIZES,
  storeThumbnailSize,
  storedThumbnailSize,
  type ThumbnailSize,
} from "../../preferences";
import { ClipThumbnail } from "./ClipThumbnail";
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
    indexing: "Recherche",
    diarization: "Voix distinctes",
    fingerprint: "Empreintes vocales",
    recognition: "Reconnaissance vocale",
  };
  const en = {
    transcription: "Speech to text",
    indexing: "Search",
    diarization: "Distinct voices",
    fingerprint: "Voice fingerprints",
    recognition: "Voice recognition",
  };
  return (language === "fr" ? fr : en)[key];
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
  const [thumbnailSize, setThumbnailSize] = useState<ThumbnailSize>(
    () => storedThumbnailSize() ?? DEFAULT_THUMBNAIL_SIZE,
  );
  const thumbnailLabels: Record<ThumbnailSize, [string, string]> = {
    none: ["Aucune", "None"],
    small: ["Petite", "Small"],
    large: ["Grande", "Large"],
  };
  const [historyOpen, setHistoryOpen] = useState(false);
  const all = [...board.cards, ...board.completed];
  return (
    <>
      <div className="mb-3 mt-1 flex flex-wrap items-end justify-end gap-3">
        <div
          role="radiogroup"
          aria-label={tr("Miniatures", "Thumbnails")}
          className="inline-flex items-center gap-1 rounded-lg border border-line p-1 text-sm"
        >
          {THUMBNAIL_SIZES.map((size) => (
            <label
              key={size}
              className="min-h-9 cursor-pointer rounded-md px-3 py-1.5 has-[:checked]:bg-accent-soft has-[:checked]:font-medium has-[:focus-visible]:outline-2"
            >
              <input
                type="radio"
                name="thumbnail-size"
                value={size}
                checked={thumbnailSize === size}
                onChange={() => {
                  setThumbnailSize(size);
                  storeThumbnailSize(size);
                }}
                className="sr-only"
              />
              {tr(...thumbnailLabels[size])}
            </label>
          ))}
        </div>
        <button onClick={() => setHistoryOpen(true)} className={buttonClass()}>
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
                    <summary
                      className={`relative flex min-h-24 cursor-pointer list-none gap-3 p-2.5 pr-2.5 ${thumbnailSize === "large" ? "flex-col" : "items-start"}`}
                    >
                      {thumbnailSize !== "none" && (
                        <ClipThumbnail
                          clipId={card.clip_id}
                          size={thumbnailSize}
                        />
                      )}
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
                        {card.voices_to_review && (
                          <span
                            data-voices-to-review
                            className="mt-1 inline-block rounded-full bg-warn-bg px-2 py-0.5 text-xs text-warn-fg"
                          >
                            {tr("Voix à vérifier", "Voices to review")}
                          </span>
                        )}
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
                        .filter(
                          (step) =>
                            step.state !== "not_applicable" ||
                            step.key === "indexing",
                        )
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
      <ClipHistoryDialog open={historyOpen} onOpenChange={setHistoryOpen} />
    </>
  );
}
