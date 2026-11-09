import type { components } from "../../../api.gen";

export type EngineEntry = components["schemas"]["WorkerEngine"];
export type EngineWorker = components["schemas"]["WorkerEnginesRow"];
export type Release = components["schemas"]["VoiceEngineRelease"];

/** Real states of a release on a Worker, in the order a Worker goes through them. */
export const engineSteps = [
  "requested",
  "downloading_runtime",
  "installing",
  "verifying",
  "ready",
] as const;

export const inProgress = (e: EngineEntry) =>
  e.state !== "ready" && e.state !== "failed" && e.state !== "removed";

/** Why an order for this release on this Worker cannot be given. Failed never blocks a new order. */
export type ResourceCause =
  | "platform_incompatible"
  | "cuda_incompatible"
  | "insufficient_ram"
  | "insufficient_vram"
  | "insufficient_disk";

export type Refusal =
  | "engine_release_unavailable"
  | "engine_already_installed"
  | "engine_order_in_progress"
  | ResourceCause;

/** The exact reason a release cannot be installed on a Worker, with the numbers behind it (bytes). */
export type Verdict = {
  cause: ResourceCause;
  required?: number;
  available?: number;
};

/**
 * Mirror of the server's resource check, to show the reason before any order; the server decides.
 * A Worker that never reported its hardware is not constrained here. `reserved` is the disk that
 * the open orders and the plan under construction already claim on this Worker.
 */
export function resourceVerdict(
  m: Release["manifest"],
  worker: EngineWorker,
  reserved = 0,
): Verdict | undefined {
  const hw = worker.hardware;
  if (!hw) return undefined;
  if (
    (hw.python !== "" && hw.python !== m.python) ||
    (hw.platform !== "" && hw.platform !== m.platform)
  )
    return { cause: "platform_incompatible" };
  // A CPU runtime is its own approved release; a CUDA release never falls back on its own.
  const onCPU = m.cuda === "cpu";
  if (!onCPU && hw.cuda !== m.cuda) return { cause: "cuda_incompatible" };
  if (!onCPU && m.vram_bytes > hw.vram_bytes)
    return {
      cause: "insufficient_vram",
      required: m.vram_bytes,
      available: hw.vram_bytes,
    };
  const ram = m.ram_bytes ?? 0;
  if (ram > 0 && hw.ram_bytes > 0 && ram > hw.ram_bytes)
    return {
      cause: "insufficient_ram",
      required: ram,
      available: hw.ram_bytes,
    };
  const available = Math.max(0, hw.disk_free_bytes - reserved);
  if (m.disk_bytes > available)
    return { cause: "insufficient_disk", required: m.disk_bytes, available };
  return undefined;
}

export function refusalFor(
  release: Release,
  worker: EngineWorker,
  reserved = 0,
): Refusal | undefined {
  if (release.revoked) return "engine_release_unavailable";
  const entries = worker.engines.filter(
    (e) => e.engine === release.manifest.engine,
  );
  if (entries.some(inProgress)) return "engine_order_in_progress";
  if (entries.some((e) => e.digest === release.digest && e.state === "ready"))
    return "engine_already_installed";
  return resourceVerdict(
    release.manifest,
    worker,
    worker.disk_reserved_bytes + reserved,
  )?.cause;
}

export type PlanItem = {
  worker: EngineWorker;
  release: Release;
  refusal?: Refusal;
};

/** The plan shown before any order: every (Worker, release) pair, with the reason for each refusal. */
export function planOrders(
  workers: EngineWorker[],
  releases: Release[],
): {
  items: PlanItem[];
  accepted: PlanItem[];
  disk: number;
  refusals: [Refusal, number][];
} {
  // Disk is cumulated per Worker: what each accepted release claims is no longer free for the next one.
  const items = workers.flatMap((worker) => {
    let claimed = 0;
    const claimedEngines = new Set<string>();
    return releases.map((release) => {
      const refusal = claimedEngines.has(release.manifest.engine)
        ? "engine_order_in_progress"
        : refusalFor(release, worker, claimed);
      if (!refusal) {
        claimed += release.manifest.disk_bytes;
        claimedEngines.add(release.manifest.engine);
      }
      return { worker, release, refusal };
    });
  });
  const accepted = items.filter((i) => !i.refusal);
  const counts = new Map<Refusal, number>();
  for (const i of items)
    if (i.refusal) counts.set(i.refusal, (counts.get(i.refusal) ?? 0) + 1);
  return {
    items,
    accepted,
    disk: accepted.reduce((sum, i) => sum + i.release.manifest.disk_bytes, 0),
    refusals: [...counts],
  };
}

/** The scope is snapshotted into explicit Worker orders at confirmation. */
export function engineTargets(
  workers: EngineWorker[],
  current: EngineWorker | undefined,
  checked: string[],
  scope: "this" | "checked",
): EngineWorker[] {
  if (scope === "checked")
    return workers.filter((w) => checked.includes(w.worker_id));
  return current ? [current] : [];
}
