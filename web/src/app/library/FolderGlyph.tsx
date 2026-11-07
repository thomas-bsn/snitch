import { Folder, FolderLock } from "lucide-react";
import { useEffect, useState } from "react";

// Keys of the covers the server answered 404 for: not asked again while the page lives. The key contains
// the logo version, so a changed logo is never hidden by a previous miss.
const withoutLogo = new Set<string>();

/** The URL of a folder's logo: the custom image when one is active, else the automatic Steam cover. */
export function folderLogoUrl(
  name: string,
  path?: string,
  version?: number,
): string {
  const params = new URLSearchParams({ name });
  if (path) params.set("path", path);
  if (version) params.set("version", String(version));
  return `/api/library/folder-logo?${params}`;
}

/** The icon of a folder: the cover of the game it is named after, else the folder icon. */
export function FolderGlyph({
  name,
  path,
  version,
  locked,
  size,
}: {
  name: string;
  /** Path relative to the video root: lets the server find a custom logo. */
  path?: string;
  /** Active custom logo version: breaks the cache when it changes. */
  version?: number;
  /** A private folder the viewer may not open shows a lock, never a cover. */
  locked?: boolean;
  /** 16 in lists and the tree, 56 in the icon view. */
  size: 16 | 56;
}) {
  const key = `${name}|${path ?? ""}|${version ?? 0}`;
  const [missing, setMissing] = useState(withoutLogo.has(key));
  useEffect(() => {
    setMissing(withoutLogo.has(key));
  }, [key]);
  const big = size === 56;
  if (locked) {
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
        src={folderLogoUrl(name, path, version)}
        alt=""
        aria-hidden="true"
        loading="lazy"
        className={`shrink-0 rounded-sm object-cover ${big ? "h-14 w-10 rounded-md" : "size-4"}`}
        onError={() => {
          withoutLogo.add(key);
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
