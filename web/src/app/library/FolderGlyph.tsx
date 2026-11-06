import { Folder, FolderLock } from "lucide-react";
import { useState } from "react";

// Names that have no game logo (the server answered 404): not asked again while the page lives.
const withoutLogo = new Set<string>();

/** The icon of a folder: the cover of the game it is named after, else the folder icon. */
export function FolderGlyph({
  name,
  isPrivate,
  size,
}: {
  name: string;
  isPrivate?: boolean;
  /** 16 in lists and the tree, 56 in the icon view. */
  size: 16 | 56;
}) {
  const [missing, setMissing] = useState(withoutLogo.has(name));
  const big = size === 56;
  if (isPrivate) {
    return (
      <FolderLock
        size={size}
        strokeWidth={big ? 1.2 : 2}
        aria-hidden="true"
        className={big ? "text-mute" : undefined}
      />
    );
  }
  if (!missing) {
    return (
      <img
        src={`/api/library/folder-logo?name=${encodeURIComponent(name)}`}
        alt=""
        aria-hidden="true"
        loading="lazy"
        className={`shrink-0 rounded-sm object-cover ${big ? "h-14 w-10 rounded-md" : "size-4"}`}
        onError={() => {
          withoutLogo.add(name);
          setMissing(true);
        }}
      />
    );
  }
  return (
    <Folder
      size={size}
      strokeWidth={big ? 1.2 : 2}
      aria-hidden="true"
      className="text-accent"
      fill={big ? "var(--accent-soft)" : undefined}
    />
  );
}
