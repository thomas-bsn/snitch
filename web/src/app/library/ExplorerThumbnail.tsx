import { useState } from "react";

/**
 * Small picture of an indexed clip in the icons view, served and cached by the server
 * (GET /api/clips/{id}/thumbnail). It loads lazily; if it fails or does not exist, the
 * parent keeps its dark tile and this component shows nothing.
 */
export function ExplorerThumbnail({ clipId }: { clipId: number }) {
  const [failed, setFailed] = useState(false);
  const [loaded, setLoaded] = useState(false);
  if (failed) return null;
  return (
    <img
      src={`/api/clips/${clipId}/thumbnail`}
      alt=""
      loading="lazy"
      decoding="async"
      onLoad={() => setLoaded(true)}
      onError={() => setFailed(true)}
      data-thumbnail={loaded ? "ready" : "loading"}
      className={`absolute inset-0 size-full rounded-md object-cover ${loaded ? "opacity-100" : "opacity-0"}`}
    />
  );
}
