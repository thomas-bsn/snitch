import { useMemo } from "react";

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

/** A clip thumbnail: a gradient tile picked from the clip's id. */
export function ExplorerThumbnail({ clipId }: { clipId: number }) {
  const tile = useMemo(() => tiles[clipId % tiles.length], [clipId]);
  return (
    <div
      className={`absolute inset-0 rounded-md bg-linear-to-br ${tile}`}
      aria-hidden="true"
    />
  );
}