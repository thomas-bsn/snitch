import { useState } from "react";
import { api, mutation } from "../api/client";
import { buttonClass } from "../components/buttons";
import { useI18n } from "../i18n";
import { isAdministrator } from "../roles";
import { useSession } from "../session";
import type { Clip } from "./libraryApi";

function localInput(value: string | null) {
  if (!value) return "";
  const date = new Date(value);
  return new Date(date.getTime() - date.getTimezoneOffset() * 60000)
    .toISOString()
    .slice(0, 16);
}

export function ClipMetadata({
  clip,
  onChanged,
}: {
  clip: Clip;
  onChanged: () => void;
}) {
  const { t } = useI18n();
  const { user, session } = useSession();
  const [editing, setEditing] = useState(false);
  const [game, setGame] = useState(clip.game ?? "");
  const [date, setDate] = useState(localInput(clip.recorded_at));
  const [error, setError] = useState(false);
  const [invalidDate, setInvalidDate] = useState(false);
  const [busy, setBusy] = useState(false);
  async function save() {
    setInvalidDate(false);
    if (date && date !== localInput(clip.recorded_at)) {
      const parsed = new Date(date);
      if (
        Number.isNaN(parsed.getTime()) ||
        localInput(parsed.toISOString()) !== date
      ) {
        setInvalidDate(true);
        return;
      }
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
        ) {
          setInvalidDate(true);
          return;
        }
      }
    }
    setBusy(true);
    setError(false);
    try {
      const result = await api.PUT("/api/admin/clips/{id}/metadata", {
        params: {
          path: { id: clip.id },
          header: mutation(session?.csrf_token),
        },
        body: {
          game: game.trim() || null,
          recorded_at: date ? new Date(date).toISOString() : null,
          game_changed: game.trim() !== (clip.game ?? ""),
          recorded_at_changed: date !== localInput(clip.recorded_at),
        },
      });
      if (!result.response.ok) throw new Error();
      setEditing(false);
      onChanged();
    } catch {
      setError(true);
    } finally {
      setBusy(false);
    }
  }
  return (
    <div className="flex flex-col gap-2 text-sm">
      <p>
        {t("metadata.game")}: {clip.game ?? t("clip.speaker.unknown")}{" "}
        {clip.game_origin &&
          `(${t(`metadata.origin.${clip.game_origin === "container" ? "container" : clip.game_origin === "filename" ? "filename" : "manual"}`)})`}
      </p>
      {clip.capture_time_origin && (
        <p>
          {t("metadata.dateOrigin")}:{" "}
          {t(
            `metadata.origin.${clip.capture_time_origin === "container" ? "container" : clip.capture_time_origin === "filename" ? "filename" : "manual"}`,
          )}
        </p>
      )}
      {clip.origins?.map((origin, index) => (
        <p key={`${origin.kind}-${index}`}>
          {origin.kind === "source"
            ? t("metadata.detected")
            : origin.name
              ? t("metadata.uploaded", { name: origin.name })
              : t("metadata.unknownUploader")}
        </p>
      ))}
      {!clip.origins?.length && <p>{t("metadata.unknownUploader")}</p>}
      {user &&
        isAdministrator(user) &&
        (editing ? (
          <div className="flex flex-col gap-2">
            <label>
              {t("metadata.game")}{" "}
              <input
                className="rounded border border-line bg-canvas p-2"
                value={game}
                onChange={(e) => setGame(e.target.value)}
              />
            </label>
            <p className="text-mute">
              {t("metadata.localDate", {
                zone: Intl.DateTimeFormat().resolvedOptions().timeZone,
              })}
            </p>
            <label>
              {t("metadata.date")}{" "}
              <input
                className="rounded border border-line bg-canvas p-2"
                type="datetime-local"
                value={date}
                onChange={(e) => setDate(e.target.value)}
              />
            </label>
            <div className="flex gap-2">
              <button
                type="button"
                className={buttonClass("primary")}
                disabled={busy}
                onClick={() => void save()}
              >
                {t("metadata.save")}
              </button>
              <button
                type="button"
                className={buttonClass()}
                onClick={() => {
                  setEditing(false);
                  setGame(clip.game ?? "");
                  setDate(localInput(clip.recorded_at));
                  setError(false);
                  setInvalidDate(false);
                }}
              >
                {t("metadata.cancel")}
              </button>
            </div>
            {error && <p role="alert">{t("metadata.error")}</p>}
            {invalidDate && <p role="alert">{t("metadata.invalidDate")}</p>}
          </div>
        ) : (
          <button
            type="button"
            className={buttonClass()}
            onClick={() => {
              setGame(clip.game ?? "");
              setDate(localInput(clip.recorded_at));
              setEditing(true);
            }}
          >
            {t("metadata.edit")}
          </button>
        ))}
    </div>
  );
}
