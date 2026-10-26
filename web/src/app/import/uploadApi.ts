import type { components } from "../../api.gen";
import { api, errorCode, mutation, origin } from "../api/client";

export type Upload = components["schemas"]["Upload"];
export type Visibility = "private" | "shared";

export type Result<T> = { ok: true; value: T } | { ok: false; code: string };

const failure = (error: unknown): { ok: false; code: string } => ({
  ok: false,
  code: errorCode(error) ?? "unknown",
});

/** Code given when the request never got an answer (connection cut, transfer aborted). */
export const NETWORK_FAILURE = "network_failure";

export async function listUploads(csrf: string): Promise<Result<Upload[]>> {
  try {
    const { data, error } = await api.GET("/api/uploads", {
      params: { header: mutation(csrf) },
    });
    return data ? { ok: true, value: data.items } : failure(error);
  } catch {
    return { ok: false, code: NETWORK_FAILURE };
  }
}

export async function createUpload(
  csrf: string,
  file: File,
  visibility: Visibility,
): Promise<Result<Upload>> {
  try {
    const { data, error } = await api.POST("/api/uploads", {
      params: { header: mutation(csrf) },
      body: { filename: file.name, size_bytes: file.size, visibility },
    });
    return data ? { ok: true, value: data } : failure(error);
  } catch {
    return { ok: false, code: NETWORK_FAILURE };
  }
}

/** Sends the bytes from `offset` to the end of the chunk; answers the new offset. */
export async function putChunk(
  csrf: string,
  upload: Pick<Upload, "id" | "chunk_size_bytes">,
  file: File,
  offset: number,
  signal: AbortSignal,
): Promise<Result<number>> {
  const end = Math.min(offset + upload.chunk_size_bytes, file.size);
  try {
    const response = await fetch(`/api/uploads/${upload.id}/chunks`, {
      method: "PUT",
      credentials: "same-origin",
      signal,
      headers: {
        ...origin(),
        "X-CSRF-Token": csrf,
        "Content-Type": "application/octet-stream",
        "Content-Range": `bytes ${offset}-${end - 1}/${file.size}`,
      },
      body: file.slice(offset, end),
    });
    const body: unknown = await response.json().catch(() => null);
    if (!response.ok) return failure(body);
    const next = (body as { offset_bytes?: unknown } | null)?.offset_bytes;
    return typeof next === "number"
      ? { ok: true, value: next }
      : { ok: false, code: NETWORK_FAILURE };
  } catch {
    return { ok: false, code: NETWORK_FAILURE };
  }
}

export async function completeUpload(
  csrf: string,
  id: string,
): Promise<Result<Upload>> {
  try {
    const { data, error } = await api.POST("/api/uploads/{id}/complete", {
      params: { path: { id }, header: mutation(csrf) },
    });
    return data ? { ok: true, value: data } : failure(error);
  } catch {
    return { ok: false, code: NETWORK_FAILURE };
  }
}

export async function cancelUpload(
  csrf: string,
  id: string,
): Promise<Result<undefined>> {
  try {
    const { response, error } = await api.DELETE("/api/uploads/{id}", {
      params: { path: { id }, header: mutation(csrf) },
    });
    return response.ok ? { ok: true, value: undefined } : failure(error);
  } catch {
    return { ok: false, code: NETWORK_FAILURE };
  }
}
