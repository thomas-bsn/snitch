import { useEffect, useRef, useState } from "react";

type Phase = "waiting" | "loading" | "ready" | "missing";

/**
 * First image of an indexed clip in the icons view, taken by the browser from the same
 * media endpoint as the queue cards (ClipThumbnail). The video is only requested once the
 * tile scrolls into view; while it is not there, or if it fails, the parent keeps its dark
 * tile (this component then shows nothing).
 */
export function ExplorerThumbnail({ clipId }: { clipId: number }) {
  const [phase, setPhase] = useState<Phase>("waiting");
  const holder = useRef<HTMLSpanElement>(null);

  useEffect(() => {
    const element = holder.current;
    if (phase !== "waiting" || !element) return;
    if (typeof IntersectionObserver === "undefined") {
      setPhase("loading");
      return;
    }
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
  }, [phase]);

  return (
    <span
      ref={holder}
      data-thumbnail={phase}
      aria-hidden="true"
      className="absolute inset-0 overflow-hidden rounded-md"
    >
      {(phase === "loading" || phase === "ready") && (
        <video
          src={`/api/clips/${clipId}/media#t=0.1`}
          preload="metadata"
          muted
          playsInline
          tabIndex={-1}
          onLoadedData={() => setPhase("ready")}
          onError={() => setPhase("missing")}
          className={`absolute inset-0 size-full object-cover ${phase === "ready" ? "opacity-100" : "opacity-0"}`}
        />
      )}
    </span>
  );
}
