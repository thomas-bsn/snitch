import { Play, Copy, MoreHorizontal, Check, X } from "lucide-react";
import { useMemo, useState, useCallback } from "react";
import { Link } from "react-router";
import { useI18n } from "../i18n";
import { ClipStatePill } from "./ClipStatePill";
import { formatTimecode } from "./clipFormat";
import { ConfirmDialog } from "../components/ConfirmDialog";
import { buttonClass } from "../components/buttons";
import { useSession } from "../session";
import { isAdministrator } from "../roles";
import type { Clip } from "./libraryApi";
import { api, mutation } from "../api/client";

// The server makes no image of a clip: the tile is a quiet gradient picked from the clip's
// number, so the same clip always looks the same.
const tiles = [
  "from-accent-soft to-side",
  "from-ok-bg to-side",
  "from-warn-bg to-side",
  "from-avatar-bg to-accent-soft",
  "from-hit to-side",
  "from-bad-bg to-side",
];

/** A clip of the library grid: tile, name, owner, date, state, ID, visibility, and admin actions. */
export function ClipCard({ clip, from }: { clip: Clip; from: string }) {
  const { t, language } = useI18n();
  const { user, session } = useSession();
  const isAdmin = user ? isAdministrator(user) : false;

  const date = useMemo(
    () =>
      new Intl.DateTimeFormat(language, { dateStyle: "medium" }).format(
        new Date(clip.recorded_at),
      ),
    [clip.recorded_at, language],
  );
  const tile = tiles[clip.id % tiles.length];

  // Admin dropdown state
  const [showMenu, setShowMenu] = useState(false);
  const [confirmOpen, setConfirmOpen] = useState(false);
  const [confirmTarget, setConfirmTarget] = useState<
    "shared" | "private" | null
  >(null);
  const [busy, setBusy] = useState(false);

  const currentVisibility = clip.visibility; // "shared" | "private"
  const nextVisibility = currentVisibility === "shared" ? "private" : "shared";

  // Translation helpers for dynamic visibility keys
  const visibilityLabel =
    currentVisibility === "shared"
      ? t("library.card.visibility.shared")
      : t("library.card.visibility.private");
  const nextVisibilityLabel =
    nextVisibility === "shared"
      ? t("library.card.visibility.shared")
      : t("library.card.visibility.private");
  const confirmMessage =
    confirmTarget === "shared"
      ? t("library.card.confirmMakeShared")
      : t("library.card.confirmMakePrivate");
  const confirmButtonLabel =
    confirmTarget === "shared"
      ? t("library.card.confirmShared")
      : t("library.card.confirmPrivate");

  const handleVisibilityChange = useCallback(async () => {
    if (!confirmTarget) return;
    setBusy(true);
    try {
      const r = await api.PATCH("/api/clips/{id}", {
        params: {
          path: { id: clip.id },
          header: mutation(session?.csrf_token ?? ""),
        } as any,
        body: { visibility: confirmTarget },
      });
      if (r.error) {
        console.error("Failed to update visibility:", r.error);
      }
    } catch (e) {
      console.error("Failed to update visibility:", e);
    } finally {
      setBusy(false);
      setConfirmOpen(false);
      setConfirmTarget(null);
      setShowMenu(false);
    }
  }, [clip.id, confirmTarget, session?.csrf_token]);

  const openConfirm = (target: "shared" | "private") => {
    setConfirmTarget(target);
    setConfirmOpen(true);
    setShowMenu(false);
  };

  return (
    <li className="min-w-0">
      <article className="relative flex flex-col gap-2 group">
        <div
          className={`relative flex aspect-video items-center justify-center rounded-lg border border-line bg-linear-to-br ${tile}`}
        >
          <Play className="size-7 text-mute" aria-hidden="true" />
          {typeof clip.duration_seconds === "number" && (
            <span className="absolute right-1.5 bottom-1.5 rounded bg-black/75 px-1.5 py-0.5 text-[11px] font-medium text-white">
              {formatTimecode(clip.duration_seconds)}
            </span>
          )}
        </div>

        <div className="flex items-start justify-between gap-2 min-w-0 px-1">
          <h2 className="truncate text-sm font-medium flex-1 min-w-0">
            <Link
              to={`/clips/${clip.id}`}
              state={{ from }}
              aria-label={t("library.card.open", { name: clip.original_name })}
              className="after:absolute after:inset-0 after:rounded-lg"
            >
              {clip.original_name}
            </Link>
          </h2>
          {/* Visibility badge in header */}
          <span className="flex items-center gap-1 shrink-0">
            <span
              className={`inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-[11px] font-medium ${
                currentVisibility === "shared"
                  ? "bg-green-50 text-green-700"
                  : "bg-gray-100 text-gray-600"
              }`}
              aria-label={visibilityLabel}
            >
              <span
                className={`size-1.5 rounded-full ${
                  currentVisibility === "shared"
                    ? "bg-green-500"
                    : "bg-gray-400"
                }`}
                aria-hidden="true"
              />
              {visibilityLabel}
            </span>
          </span>
        </div>

        <div className="flex flex-wrap items-center gap-x-2 gap-y-1 text-xs text-mute px-1">
          <span>
            {t("library.card.by", {
              owner: clip.owner_deleted
                ? t("library.deletedOwner")
                : clip.owner,
            })}
          </span>
          <span aria-hidden="true">·</span>
          <time dateTime={clip.recorded_at}>{date}</time>

          <div className="ml-auto flex items-center gap-1">
            {/* Clip ID with copy button */}
            <span
              className="font-mono text-primary-hover"
              id={`clip-id-${clip.id}`}
            >
              #{clip.id}
            </span>
            <button
              type="button"
              className="relative z-10 inline-flex min-h-11 min-w-11 items-center justify-center rounded hover:bg-muted transition-colors"
              aria-label={t("library.card.copyId")}
              onClick={async () => {
                try {
                  await navigator.clipboard.writeText(String(clip.id));
                } catch {
                  // Fallback for non-HTTPS
                  const textarea = document.createElement("textarea");
                  textarea.value = String(clip.id);
                  document.body.appendChild(textarea);
                  textarea.select();
                  document.execCommand("copy");
                  document.body.removeChild(textarea);
                }
              }}
            >
              <Copy className="size-4" aria-hidden="true" />
            </button>

            <ClipStatePill status={clip.status} />
          </div>
        </div>

        {/* Admin dropdown menu */}
        {isAdmin && (
          <div className="relative">
            <button
              type="button"
              className="absolute top-0 right-0 z-10 inline-flex min-h-11 min-w-11 items-center justify-center rounded hover:bg-muted/50 transition-colors opacity-100 md:opacity-0 md:group-hover:opacity-100"
              aria-label={t("library.card.changeVisibility")}
              aria-expanded={showMenu}
              aria-haspopup="menu"
              onClick={(e) => {
                e.stopPropagation();
                setShowMenu((prev) => !prev);
              }}
            >
              <MoreHorizontal className="size-4 text-mute" aria-hidden="true" />
            </button>

            {showMenu && (
              <>
                <div
                  className="fixed inset-0 z-30"
                  onClick={() => setShowMenu(false)}
                  aria-hidden="true"
                />
                <div className="absolute top-full right-0 mt-1 z-40 min-w-[160px] rounded-lg border border-line bg-surface shadow-pop">
                  <button
                    type="button"
                    className="w-full flex items-center gap-2 px-3 py-2 text-sm text-ink hover:bg-muted"
                    onClick={() => openConfirm(nextVisibility)}
                  >
                    {nextVisibility === "shared" ? (
                      <>
                        <Check
                          className="size-4 text-green-600"
                          aria-hidden="true"
                        />
                        {nextVisibilityLabel}
                      </>
                    ) : (
                      <>
                        <X
                          className="size-4 text-gray-500"
                          aria-hidden="true"
                        />
                        {nextVisibilityLabel}
                      </>
                    )}
                  </button>
                </div>
              </>
            )}
          </div>
        )}

        {/* Confirm Dialog */}
        <ConfirmDialog
          open={confirmOpen}
          onOpenChange={setConfirmOpen}
          title={t("library.card.confirmVisibilityChange")}
          cancelLabel={t("segmentEdit.cancel")}
          confirmLabel={confirmButtonLabel}
          danger={confirmTarget === "private"}
          busy={busy}
          onConfirm={handleVisibilityChange}
        >
          <p>{confirmMessage}</p>
        </ConfirmDialog>
      </article>
    </li>
  );
}
