import type { Result } from "../api/loaded";
import { errorCode } from "../api/client";

/** A version of a folder's custom logo, newest first. */
export type FolderLogoVersion = { version: number; created_at: string };

/** The logos of one folder: the active choice and its history. */
export type FolderLogoState = {
  path: string;
  /** False: the automatic Steam cover is shown, the history is kept. */
  custom: boolean;
  /** Four at most, newest first. */
  versions: FolderLogoVersion[];
};

const failure = (error: unknown): { ok: false; code: string } => ({
  ok: false,
  code: errorCode(error) ?? "unknown",
});

async function errorOf(response: Response): Promise<unknown> {
  try {
    return await response.json();
  } catch {
    return undefined;
  }
}

export async function fetchFolderLogoState(
  path: string,
): Promise<Result<FolderLogoState>> {
  const response = await fetch(
    `/api/library/folder-logo/history?path=${encodeURIComponent(path)}`,
  );
  const body = await response.json().catch(() => undefined);
  if (!response.ok || !body) return failure(await errorOf(response));
  return { ok: true, value: body as FolderLogoState };
}

/** Sends the image itself; the server accepts PNG or JPEG up to 2 MB. */
export async function uploadFolderLogo(
  path: string,
  file: File,
  csrf: string,
): Promise<Result<FolderLogoState>> {
  const response = await fetch(
    `/api/library/folder-logo?path=${encodeURIComponent(path)}`,
    {
      method: "PUT",
      headers: {
        "Content-Type": file.type || "application/octet-stream",
        "X-CSRF-Token": csrf,
      },
      body: file,
    },
  );
  const body = await response.json().catch(() => undefined);
  if (!response.ok || !body) return failure(await errorOf(response));
  return { ok: true, value: body as FolderLogoState };
}

/** Back to the automatic Steam cover, without clearing the history. */
export async function resetFolderLogo(
  path: string,
  csrf: string,
): Promise<Result<null>> {
  const response = await fetch(
    `/api/library/folder-logo?path=${encodeURIComponent(path)}`,
    {
      method: "DELETE",
      headers: { "X-CSRF-Token": csrf },
    },
  );
  if (!response.ok) return failure(await errorOf(response));
  return { ok: true, value: null };
}

/** Restores an older image by making it the newest version. */
export async function restoreFolderLogo(
  path: string,
  version: number,
  csrf: string,
): Promise<Result<FolderLogoState>> {
  const response = await fetch("/api/library/folder-logo/history", {
    method: "POST",
    headers: { "Content-Type": "application/json", "X-CSRF-Token": csrf },
    body: JSON.stringify({ path, version }),
  });
  const body = await response.json().catch(() => undefined);
  if (!response.ok || !body) return failure(await errorOf(response));
  return { ok: true, value: body as FolderLogoState };
}
