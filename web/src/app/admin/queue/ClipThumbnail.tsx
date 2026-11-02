import { ImageOff } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { api } from "../../api/client";
import { useI18n } from "../../i18n";
import type { ThumbnailSize } from "../../preferences";

type Phase = "waiting" | "loading" | "ready" | "missing";

/** Same colored tiles as the library: they stay visible while the image is not there. */
const tiles = [
  "from-accent-soft to-side",
  "from-ok-bg to-side",
  "from-warn-bg to-side",
  "from-avatar-bg to-accent-soft",
  "from-hit to-side",
  "from-bad-bg to-side",
];

/**
 * First image of a clip, taken by the browser as in the library. The clip is only
 * asked for (state, then media) once the tile scrolls into view, so a long queue
 * does not fetch a video per card.
 */
export function ClipThumbnail({
  clipId,
  size,
}: {
  clipId: number;
  size: Exclude<ThumbnailSize, "none">;
}) {
  const { language } = useI18n();
  const [phase, setPhase] = useState<Phase>("waiting");
  const [stateKnown, setStateKnown] = useState(false);
  const tile = useRef<HTMLDivElement>(null);
  const seen = phase !== "waiting";

  useEffect(() => {
    const element = tile.current;
    if (seen || !element) return;
    const observer = new IntersectionObserver(
      (entries) => {
        if (!entries.some((entry) => entry.isIntersecting)) return;
        observer.disconnect();
        setPhase("loading");
      },
      { rootMargin: "100px" },
    );
    observer.observe(element);
    return () => observer.disconnect();
  }, [seen]);

  // A missing or unreadable original is known from the clip: no media request is
  // made for it, hence no failed request in the console.
  useEffect(() => {
    if (!seen || stateKnown) return;
    let disposed = false;
    void api
      .GET("/api/clips/{id}", { params: { path: { id: clipId } } })
      .then((result) => {
        if (disposed) return;
        const status = result.data?.status;
        if (status === "missing" || status === "unreadable")
          setPhase("missing");
        else setStateKnown(true);
      })
      .catch(() => {
        if (!disposed) setStateKnown(true);
      });
    return () => {
      disposed = true;
    };
  }, [seen, stateKnown, clipId]);

  const label =
    language === "fr"
      ? { loading: "En chargement", missing: "Original introuvable" }
      : { loading: "Loading", missing: "Original not found" };
  const shape = size === "large" ? "w-full" : "w-24 shrink-0";
  return (
    <div
      ref={tile}
      data-thumbnail={phase}
      data-size={size}
      className={`relative aspect-video ${shape} overflow-hidden rounded-md border border-line bg-linear-to-br ${tiles[clipId % tiles.length]}`}
    >
      {stateKnown && phase !== "missing" && (
        <video
          src={`/api/clips/${clipId}/media#t=0.1`}
          preload="metadata"
          muted
          playsInline
          aria-hidden="true"
          onLoadedData={() => setPhase("ready")}
          onError={() => setPhase("missing")}
          className="absolute inset-0 size-full object-cover"
        />
      )}
      {phase === "missing" ? (
        <span className="absolute inset-0 flex flex-col items-center justify-center gap-1 p-1 text-center text-[11px] leading-tight text-mute">
          <ImageOff size={size === "large" ? 20 : 14} aria-hidden="true" />
          {label.missing}
        </span>
      ) : (
        phase !== "ready" && (
          <span className="absolute inset-0 flex items-center justify-center p-1 text-center text-[11px] text-mute motion-safe:animate-pulse">
            {label.loading}
          </span>
        )
      )}
    </div>
  );
}
