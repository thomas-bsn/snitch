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
  e.state !== "ready" && e.state !== "failed";

/** Why an order for this release on this Worker cannot be given. Failed never blocks a new order. */
export type Refusal =
  | "engine_release_unavailable"
  | "engine_already_installed"
  | "engine_order_in_progress";

export function refusalFor(
  release: Release,
  worker: EngineWorker,
): Refusal | undefined {
  if (release.revoked) return "engine_release_unavailable";
  const entries = worker.engines.filter(
    (e) => e.engine === release.manifest.engine,
  );
  if (entries.some(inProgress)) return "engine_order_in_progress";
  if (entries.some((e) => e.digest === release.digest && e.state === "ready"))
    return "engine_already_installed";
  return undefined;
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
  const items = workers.flatMap((worker) =>
    releases.map((release) => ({
      worker,
      release,
      refusal: refusalFor(release, worker),
    })),
  );
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
