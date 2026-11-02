import { Pencil } from "lucide-react";
import { useId, useState, type FormEvent } from "react";
import { api, errorCode, mutation } from "../api/client";
import { ErrorAlert } from "../components/Alert";
import { buttonClass } from "../components/buttons";
import { Modal } from "../components/Modal";
import { useI18n } from "../i18n";
import type { MessageKey } from "../i18n/messages";
import { isAdministrator } from "../roles";
import { useSession } from "../session";
import { clipTitle, type Clip } from "./libraryApi";

function localInput(value: string | null) {
  if (!value) return "";
  const date = new Date(value);
  return new Date(date.getTime() - date.getTimezoneOffset() * 60000)
    .toISOString()
    .slice(0, 16);
}

/** A local time the browser zone cannot express (a clock change makes it ambiguous or absent). */
function ambiguousLocalTime(date: string) {
  const parsed = new Date(date);
  if (
    Number.isNaN(parsed.getTime()) ||
    localInput(parsed.toISOString()) !== date
  )
    return true;
  for (let minutes = 1; minutes <= 180; minutes++) {
    if (
      [-1, 1].some(
        (direction) =>
          localInput(
            new Date(
              parsed.getTime() + direction * minutes * 60000,
            ).toISOString(),
          ) === date,
      )
    )
      return true;
  }
  return false;
}

const originKey = (origin: string | null | undefined) =>
  origin === "container"
    ? "metadata.origin.container"
    : origin === "filename"
      ? "metadata.origin.filename"
      : "metadata.origin.manual";

const pillClass = (manual: boolean) =>
  `ml-2 rounded-full px-2 py-0.5 text-xs font-medium ${manual ? "bg-warn-bg text-warn-fg" : "bg-accent-soft text-accent-ink"}`;

/** What the clip says about itself: game, date, where they come from, and the read-only technical data. */
export function ClipMetadata({ clip }: { clip: Clip }) {
  const { t } = useI18n();
  return (
    <div className="flex flex-col gap-3 text-sm">
      <dl className="m-0 grid grid-cols-[max-content_1fr] gap-x-4 gap-y-1.5 rounded-xl border border-line bg-surface p-4">
        <dt className="text-mute">{t("metadata.game")}</dt>
        <dd className="m-0">
          {clip.game ?? t("clip.speaker.unknown")}
          {clip.game_origin && (
            <span className={pillClass(clip.game_origin === "manual")}>
              {t(originKey(clip.game_origin))}
            </span>
          )}
        </dd>
        <dt className="text-mute">{t("metadata.date")}</dt>
        <dd className="m-0">
          {clip.recorded_at
            ? new Date(clip.recorded_at).toLocaleString()
            : t("clip.speaker.unknown")}
          {clip.capture_time_origin && (
            <span className={pillClass(clip.capture_time_origin === "manual")}>
              {t(originKey(clip.capture_time_origin))}
            </span>
          )}
        </dd>
        {clip.origins?.map((origin, index) => (
          <div key={`${origin.kind}-${index}`} className="contents">
            <dt className="text-mute">{t("metadata.origin.label")}</dt>
            <dd className="m-0">
              {origin.kind === "source"
                ? t("metadata.detected")
                : origin.name
                  ? t("metadata.uploaded", { name: origin.name })
                  : t("metadata.unknownUploader")}
            </dd>
          </div>
        ))}
        {!clip.origins?.length && (
          <>
            <dt className="text-mute">{t("metadata.origin.label")}</dt>
            <dd className="m-0">{t("metadata.unknownUploader")}</dd>
          </>
        )}
      </dl>
      <dl className="m-0 grid grid-cols-[max-content_1fr] gap-x-4 gap-y-1.5 rounded-xl border border-line bg-surface p-4">
        <dt className="font-medium">{t("metadata.technical")}</dt>
        <dd className="m-0 text-mute">{t("metadata.readOnly")}</dd>
        <dt className="text-mute">{t("metadata.fileName")}</dt>
        <dd className="m-0 break-all font-mono text-xs">
          {clip.original_name}
        </dd>
        <dt className="text-mute">{t("metadata.codecs")}</dt>
        <dd className="m-0">
          {[clip.video_codec, clip.audio_codec].filter(Boolean).join(" · ") ||
            "—"}
        </dd>
      </dl>
    </div>
  );
}

/** The “Edit metadata” action of the clip page, reserved to administrators (the API checks it too). */
export function EditMetadata({
  clip,
  onChanged,
}: {
  clip: Clip;
  onChanged: () => void;
}) {
  const { user } = useSession();
  if (!user || !isAdministrator(user)) return null;
  return <EditMetadataDialog clip={clip} onChanged={onChanged} />;
}

function EditMetadataDialog({
  clip,
  onChanged,
}: {
  clip: Clip;
  onChanged: () => void;
}) {
  const { t, error } = useI18n();
  const { session } = useSession();
  const ids = { title: useId(), game: useId(), date: useId() };
  const [open, setOpen] = useState(false);
  const [title, setTitle] = useState("");
  const [game, setGame] = useState("");
  const [date, setDate] = useState("");
  const [restoreGame, setRestoreGame] = useState(false);
  const [restoreDate, setRestoreDate] = useState(false);
  const [failure, setFailure] = useState<string>();
  const [problem, setProblem] = useState<MessageKey>();
  const [busy, setBusy] = useState(false);
  const detected = clip.detected;

  function openDialog() {
    setTitle(clipTitle(clip));
    setGame(clip.game ?? "");
    setDate(localInput(clip.recorded_at));
    setRestoreGame(false);
    setRestoreDate(false);
    setFailure(undefined);
    setProblem(undefined);
    setOpen(true);
  }

  async function save(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setProblem(undefined);
    const trimmed = title.trim();
    if (!trimmed) return setProblem("metadata.titleRequired");
    const dateChanged = !restoreDate && date !== localInput(clip.recorded_at);
    if (dateChanged && date && ambiguousLocalTime(date))
      return setProblem("metadata.invalidDate");
    // Back on the file name means no custom title at all.
    const nextTitle = trimmed === clip.original_name ? "" : trimmed;
    const gameChanged = !restoreGame && game.trim() !== (clip.game ?? "");
    setBusy(true);
    setFailure(undefined);
    try {
      const result = await api.PUT("/api/admin/clips/{id}/metadata", {
        params: {
          path: { id: clip.id },
          header: mutation(session?.csrf_token),
        },
        body: {
          display_title: nextTitle,
          display_title_changed: nextTitle !== (clip.display_title ?? ""),
          game: game.trim() || null,
          game_changed: gameChanged,
          game_reset: restoreGame,
          recorded_at: date ? new Date(date).toISOString() : null,
          recorded_at_changed: dateChanged,
          recorded_at_reset: restoreDate,
        },
      });
      if (result.error) {
        setFailure(errorCode(result.error) ?? "unknown");
        return;
      }
      setOpen(false);
      onChanged();
    } catch {
      setFailure("unavailable");
    } finally {
      setBusy(false);
    }
  }

  const field =
    "h-11 w-full rounded-lg border border-line-strong bg-canvas px-3 text-ink md:h-10";
  const link = "min-h-8 text-accent-ink underline underline-offset-2";
  return (
    <>
      <button
        type="button"
        className={buttonClass("secondary")}
        onClick={openDialog}
      >
        <Pencil className="size-4" aria-hidden="true" />
        {t("metadata.editAction")}
      </button>
      <Modal
        open={open}
        onOpenChange={setOpen}
        title={t("metadata.editAction")}
        closeLabel={t("metadata.cancel")}
      >
        <form onSubmit={(e) => void save(e)} className="flex flex-col gap-4">
          <p className="m-0 text-sm text-mute">{t("metadata.noRename")}</p>
          <div className="flex flex-col gap-1.5">
            <label htmlFor={ids.title} className="font-medium">
              {t("metadata.titleLabel")}
            </label>
            <input
              id={ids.title}
              className={field}
              maxLength={200}
              value={title}
              onChange={(e) => setTitle(e.target.value)}
            />
            <p className="m-0 flex flex-wrap items-center gap-x-3 text-xs text-mute">
              {t("metadata.fileKept", { name: clip.original_name })}
              {title !== clip.original_name && (
                <button
                  type="button"
                  className={link}
                  onClick={() => setTitle(clip.original_name)}
                >
                  {t("metadata.useFileName")}
                </button>
              )}
            </p>
          </div>
          <div className="flex flex-col gap-1.5">
            <label htmlFor={ids.game} className="font-medium">
              {t("metadata.game")}
            </label>
            <input
              id={ids.game}
              className={field}
              maxLength={200}
              value={game}
              onChange={(e) => {
                setGame(e.target.value);
                setRestoreGame(false);
              }}
            />
            {clip.game_origin === "manual" && detected?.game_available && (
              <p className="m-0 flex flex-wrap items-center gap-x-3 text-xs text-mute">
                {t("metadata.detectedValue", {
                  value: detected.game ?? t("clip.speaker.unknown"),
                })}
                <button
                  type="button"
                  className={link}
                  onClick={() => {
                    setGame(detected.game ?? "");
                    setRestoreGame(true);
                  }}
                >
                  {t("metadata.restore")}
                </button>
              </p>
            )}
          </div>
          <div className="flex flex-col gap-1.5">
            <label htmlFor={ids.date} className="font-medium">
              {t("metadata.date")}
            </label>
            <input
              id={ids.date}
              className={field}
              type="datetime-local"
              value={date}
              onChange={(e) => {
                setDate(e.target.value);
                setRestoreDate(false);
              }}
            />
            <p className="m-0 text-xs text-mute">
              {t("metadata.localDate", {
                zone: Intl.DateTimeFormat().resolvedOptions().timeZone,
              })}
            </p>
            {clip.capture_time_origin === "manual" &&
              detected?.recorded_at_available && (
                <p className="m-0 flex flex-wrap items-center gap-x-3 text-xs text-mute">
                  {t("metadata.detectedValue", {
                    value: detected.recorded_at
                      ? new Date(detected.recorded_at).toLocaleString()
                      : t("clip.speaker.unknown"),
                  })}
                  <button
                    type="button"
                    className={link}
                    onClick={() => {
                      setDate(localInput(detected.recorded_at));
                      setRestoreDate(true);
                    }}
                  >
                    {t("metadata.restore")}
                  </button>
                </p>
              )}
          </div>
          {problem && <ErrorAlert>{t(problem)}</ErrorAlert>}
          {failure && <ErrorAlert>{error(failure)}</ErrorAlert>}
          <div className="flex flex-wrap justify-end gap-2">
            <button
              type="button"
              className={buttonClass("secondary")}
              onClick={() => setOpen(false)}
            >
              {t("metadata.cancel")}
            </button>
            <button
              type="submit"
              className={buttonClass("primary")}
              disabled={busy}
            >
              {t("metadata.save")}
            </button>
          </div>
        </form>
      </Modal>
    </>
  );
}
